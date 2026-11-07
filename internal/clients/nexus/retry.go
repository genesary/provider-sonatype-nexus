package nexus

import (
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

const (
	// maxRequestAttempts is the number of times an idempotent request is
	// sent before its failure is reported to the reconciler, which then
	// requeues it with its own backoff.
	maxRequestAttempts = 3

	// initialRetryDelay is the base of the exponential backoff between
	// attempts. Delays stay short so that they use little of RequestTimeout,
	// which bounds all the attempts of a request together.
	initialRetryDelay = 500 * time.Millisecond

	// maxRetryDelay caps the backoff between attempts, and the wait that a
	// Retry-After header may ask for. A response asking for longer is
	// returned as it is, so that a rate-limiting proxy cannot hold every
	// reconcile worker.
	maxRetryDelay = 5 * time.Second

	// maxDrainBytes is how much of a discarded response body is read, so
	// that its connection can be reused for the next attempt.
	maxDrainBytes = 64 << 10
)

// retryTransport is an [http.RoundTripper] that resends idempotent requests
// that failed for a reason likely to be brief: Nexus restarting behind a
// proxy, a proxy momentarily losing its backend, or a connection reset.
//
// Only GET, HEAD, OPTIONS, PUT and DELETE are resent, as resending a POST
// could repeat its effect. A resent DELETE whose first response was lost
// reports a 404, which the controllers already treat as deleted. Requests
// whose body cannot be replayed are sent once.
type retryTransport struct {
	base http.RoundTripper

	// sleep waits for delay, or until ctx is done. Tests replace it.
	sleep func(ctx context.Context, delay time.Duration) error
}

// newRetryTransport returns a retryTransport sending its requests through
// base.
func newRetryTransport(base http.RoundTripper) *retryTransport {
	return &retryTransport{base: base, sleep: sleepContext}
}

// RoundTrip implements [http.RoundTripper].
func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !isReplayable(req) {
		return t.base.RoundTrip(req)
	}

	for attempt := 1; ; attempt++ {
		attemptReq, err := rewind(req, attempt)
		if err != nil {
			return nil, err
		}

		resp, err := t.base.RoundTrip(attemptReq)

		delay, retry := retryDelay(resp, err, attempt)
		if !retry || req.Context().Err() != nil {
			if err != nil {
				return nil, err
			}

			return resp, nil
		}

		discard(resp)

		sleepErr := t.sleep(req.Context(), delay)
		if sleepErr != nil {
			return nil, sleepErr
		}
	}
}

// isReplayable reports whether req is idempotent and its body, if any, can be
// sent again.
func isReplayable(req *http.Request) bool {
	switch req.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut, http.MethodDelete:
	default:
		return false
	}

	return req.Body == nil || req.Body == http.NoBody || req.GetBody != nil
}

// rewind returns the request to send for attempt: req itself for the first
// one, and a copy with a fresh body for the next ones, as the previous
// attempt consumed the body.
func rewind(req *http.Request, attempt int) (*http.Request, error) {
	if attempt == 1 || req.GetBody == nil {
		return req, nil
	}

	body, err := req.GetBody()
	if err != nil {
		return nil, errors.Wrap(err, "cannot rewind the request body")
	}

	rewound := req.Clone(req.Context())
	rewound.Body = body

	return rewound, nil
}

// retryDelay returns how long to wait before sending attempt+1, and whether
// to send it at all, given the outcome of attempt.
func retryDelay(resp *http.Response, err error, attempt int) (time.Duration, bool) {
	if attempt >= maxRequestAttempts {
		return 0, false
	}

	backoff := min(initialRetryDelay<<(attempt-1), maxRetryDelay)

	if err != nil {
		return backoff, isRetryableError(err)
	}

	switch resp.StatusCode {
	case http.StatusTooManyRequests, http.StatusServiceUnavailable:
		retryAfter, set := parseRetryAfter(resp.Header.Get("Retry-After"))
		if !set {
			return backoff, true
		}

		return retryAfter, retryAfter <= maxRetryDelay
	case http.StatusBadGateway, http.StatusGatewayTimeout:
		return backoff, true
	default:
		return 0, false
	}
}

// isRetryableError reports whether a transport error is worth another
// attempt. A timeout is not: a Nexus that stalls would make each attempt wait
// out the whole request timeout. Neither is a cancelled or expired context.
func isRetryableError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var netErr net.Error

	return !errors.As(err, &netErr) || !netErr.Timeout()
}

// parseRetryAfter parses the delay-seconds form of a Retry-After header. The
// HTTP-date form is not supported, and reads as unset.
func parseRetryAfter(value string) (time.Duration, bool) {
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds < 0 {
		return 0, false
	}

	return time.Duration(seconds) * time.Second, true
}

// discard reads a little of the body of a response that will not be returned
// and closes it, so that its connection can be reused.
func discard(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}

	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxDrainBytes))
	_ = resp.Body.Close()
}

// sleepContext waits for delay, or until ctx is done.
func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "request cancelled while waiting to retry")
	case <-timer.C:
		return nil
	}
}
