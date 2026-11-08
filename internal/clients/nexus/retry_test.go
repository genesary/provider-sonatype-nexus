package nexus

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// recordedSleeps returns a sleep function recording the delays it is asked
// to wait, without waiting.
func recordedSleeps(delays *[]time.Duration) func(context.Context, time.Duration) error {
	return func(ctx context.Context, delay time.Duration) error {
		*delays = append(*delays, delay)

		return ctx.Err()
	}
}

// testJitterOffset is what offsetJitter adds to every backoff, so that tests
// can tell a jittered backoff from a delay that is waited out as it is.
const testJitterOffset = time.Millisecond

// offsetJitter is a deterministic jitter adding testJitterOffset.
func offsetJitter(backoff time.Duration) time.Duration {
	return backoff + testJitterOffset
}

// TestRetryTransportStatuses checks which responses are retried, how many
// times and after which delays, and which response is finally returned.
func TestRetryTransportStatuses(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		method     string
		statuses   []int
		retryAfter string
		wantCalls  int
		wantStatus int
		wantDelays []time.Duration
	}{
		"SuccessIsNotRetried": {
			method:     http.MethodGet,
			statuses:   []int{http.StatusOK},
			wantCalls:  1,
			wantStatus: http.StatusOK,
		},
		"ClientErrorIsNotRetried": {
			method:     http.MethodGet,
			statuses:   []int{http.StatusNotFound},
			wantCalls:  1,
			wantStatus: http.StatusNotFound,
		},
		"ServerErrorIsNotRetried": {
			method:     http.MethodGet,
			statuses:   []int{http.StatusInternalServerError},
			wantCalls:  1,
			wantStatus: http.StatusInternalServerError,
		},
		"BadGatewayIsRetriedUntilItSucceeds": {
			method:     http.MethodGet,
			statuses:   []int{http.StatusBadGateway, http.StatusOK},
			wantCalls:  2,
			wantStatus: http.StatusOK,
			wantDelays: []time.Duration{initialRetryDelay + testJitterOffset},
		},
		"GatewayTimeoutIsRetriedWithBackoff": {
			method:     http.MethodPut,
			statuses:   []int{http.StatusGatewayTimeout, http.StatusGatewayTimeout, http.StatusGatewayTimeout},
			wantCalls:  maxRequestAttempts,
			wantStatus: http.StatusGatewayTimeout,
			wantDelays: []time.Duration{initialRetryDelay + testJitterOffset, 2*initialRetryDelay + testJitterOffset},
		},
		"UnavailableHonorsRetryAfter": {
			method:     http.MethodDelete,
			statuses:   []int{http.StatusServiceUnavailable, http.StatusNoContent},
			retryAfter: "2",
			wantCalls:  2,
			wantStatus: http.StatusNoContent,
			wantDelays: []time.Duration{2 * time.Second},
		},
		"TooManyRequestsAskingTooLongIsReturned": {
			method:     http.MethodGet,
			statuses:   []int{http.StatusTooManyRequests},
			retryAfter: "60",
			wantCalls:  1,
			wantStatus: http.StatusTooManyRequests,
		},
		"RetryAfterTooLongForADurationIsReturned": {
			method:     http.MethodGet,
			statuses:   []int{http.StatusTooManyRequests},
			retryAfter: "99999999999999999",
			wantCalls:  1,
			wantStatus: http.StatusTooManyRequests,
		},
		"InvalidRetryAfterFallsBackToBackoff": {
			method:     http.MethodGet,
			statuses:   []int{http.StatusTooManyRequests, http.StatusOK},
			retryAfter: "Wed, 21 Oct 2026 07:28:00 GMT",
			wantCalls:  2,
			wantStatus: http.StatusOK,
			wantDelays: []time.Duration{initialRetryDelay + testJitterOffset},
		},
		"PostIsNeverRetried": {
			method:     http.MethodPost,
			statuses:   []int{http.StatusBadGateway},
			wantCalls:  1,
			wantStatus: http.StatusBadGateway,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var calls atomic.Int32

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				call := int(calls.Add(1))

				body, _ := io.ReadAll(r.Body)
				if string(body) != "payload" {
					t.Errorf("attempt %d sent body %q, want %q", call, body, "payload")
				}

				if test.retryAfter != "" {
					w.Header().Set("Retry-After", test.retryAfter)
				}

				w.WriteHeader(test.statuses[min(call, len(test.statuses))-1])
			}))
			t.Cleanup(srv.Close)

			var delays []time.Duration

			transport := newRetryTransport(http.DefaultTransport)
			transport.sleep = recordedSleeps(&delays)
			transport.jitter = offsetJitter

			req, err := http.NewRequestWithContext(context.Background(), test.method, srv.URL, strings.NewReader("payload"))
			if err != nil {
				t.Fatalf("NewRequest() error = %v", err)
			}

			resp, err := transport.RoundTrip(req)
			if err != nil {
				t.Fatalf("RoundTrip() error = %v", err)
			}

			_ = resp.Body.Close()

			if resp.StatusCode != test.wantStatus {
				t.Errorf("RoundTrip() status = %d, want %d", resp.StatusCode, test.wantStatus)
			}

			if int(calls.Load()) != test.wantCalls {
				t.Errorf("server received %d requests, want %d", calls.Load(), test.wantCalls)
			}

			if !slices.Equal(delays, test.wantDelays) {
				t.Errorf("RoundTrip() waited %v, want %v", delays, test.wantDelays)
			}
		})
	}
}

// errorTransport fails every request with err, counting them.
type errorTransport struct {
	err   error
	calls int
}

// RoundTrip implements [http.RoundTripper].
func (t *errorTransport) RoundTrip(*http.Request) (*http.Response, error) {
	t.calls++

	return nil, t.err
}

// timeoutError is a [net.Error] reporting a timeout.
type timeoutError struct{}

// Error implements [error].
func (timeoutError) Error() string { return "i/o timeout" }

// Timeout reports that the error is a timeout.
func (timeoutError) Timeout() bool { return true }

// Temporary implements the deprecated part of [net.Error].
func (timeoutError) Temporary() bool { return true }

// TestRetryTransportErrors checks which transport errors are retried.
func TestRetryTransportErrors(t *testing.T) {
	t.Parallel()

	var _ net.Error = timeoutError{}

	tests := map[string]struct {
		err       error
		wantCalls int
	}{
		"ConnectionResetIsRetried": {
			err:       errors.New("connection reset by peer"),
			wantCalls: maxRequestAttempts,
		},
		"TimeoutIsNotRetried": {
			err:       timeoutError{},
			wantCalls: 1,
		},
		"CancellationIsNotRetried": {
			err:       context.Canceled,
			wantCalls: 1,
		},
		"UnknownHostIsNotRetried": {
			err:       &net.OpError{Op: "dial", Net: "tcp", Err: &net.DNSError{Err: "no such host", Name: "nexus.invalid", IsNotFound: true}},
			wantCalls: 1,
		},
		"CertificateVerificationFailureIsNotRetried": {
			err:       &tls.CertificateVerificationError{Err: errors.New("x509: certificate signed by unknown authority")},
			wantCalls: 1,
		},
		"PlainHTTPServerIsNotRetried": {
			err:       tls.RecordHeaderError{Msg: "first record does not look like a TLS handshake"},
			wantCalls: 1,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			base := &errorTransport{err: test.err}

			var delays []time.Duration

			transport := newRetryTransport(base)
			transport.sleep = recordedSleeps(&delays)

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://nexus.invalid/service/rest/v1/status", http.NoBody)
			if err != nil {
				t.Fatalf("NewRequest() error = %v", err)
			}

			resp, err := transport.RoundTrip(req)
			if resp != nil {
				_ = resp.Body.Close()
			}

			if !errors.Is(err, test.err) {
				t.Errorf("RoundTrip() error = %v, want %v", err, test.err)
			}

			if base.calls != test.wantCalls {
				t.Errorf("RoundTrip() sent %d requests, want %d", base.calls, test.wantCalls)
			}
		})
	}
}

// TestRetryTransportStopsWhenCancelled checks that a cancelled request stops
// waiting for its next attempt.
func TestRetryTransportStopsWhenCancelled(t *testing.T) {
	t.Parallel()

	base := &errorTransport{err: errors.New("connection reset by peer")}
	transport := newRetryTransport(base)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://nexus.invalid/", http.NoBody)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	resp, err := transport.RoundTrip(req)
	if resp != nil {
		_ = resp.Body.Close()
	}

	if err == nil {
		t.Fatal("RoundTrip() error = nil, want an error")
	}

	if base.calls != 1 {
		t.Errorf("RoundTrip() sent %d requests, want 1", base.calls)
	}
}

// TestRetryTransportReturnsWhenDeadlineIsTooClose checks that a response is
// returned as it is, rather than waited on, when the request deadline would
// expire before the next attempt.
func TestRetryTransportReturnsWhenDeadlineIsTooClose(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	var delays []time.Duration

	transport := newRetryTransport(http.DefaultTransport)
	transport.sleep = recordedSleeps(&delays)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	t.Cleanup(cancel)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, http.NoBody)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}

	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable || calls.Load() != 1 || len(delays) != 0 {
		t.Errorf("RoundTrip() = %d after %d requests and waits %v, want %d after 1 request and no wait",
			resp.StatusCode, calls.Load(), delays, http.StatusServiceUnavailable)
	}
}

// TestSpreadBackoff checks that backoffs are spread within
// retryJitterFraction of their value, and actually spread.
func TestSpreadBackoff(t *testing.T) {
	t.Parallel()

	const draws = 1000

	backoff := initialRetryDelay
	lowest := backoff - time.Duration(float64(backoff)*retryJitterFraction)
	highest := backoff + time.Duration(float64(backoff)*retryJitterFraction)

	seen := make(map[time.Duration]struct{})

	for range draws {
		delay := spreadBackoff(backoff)
		if delay < lowest || delay > highest {
			t.Fatalf("spreadBackoff(%v) = %v, want within [%v, %v]", backoff, delay, lowest, highest)
		}

		seen[delay] = struct{}{}
	}

	if len(seen) < 2 {
		t.Errorf("spreadBackoff(%v) returned %d distinct delays over %d draws, want them spread", backoff, len(seen), draws)
	}

	if got := spreadBackoff(0); got != 0 {
		t.Errorf("spreadBackoff(0) = %v, want 0", got)
	}
}

// TestRetryTransportUnreplayableBody checks that a request whose body cannot
// be sent again is sent once.
func TestRetryTransportUnreplayableBody(t *testing.T) {
	t.Parallel()

	base := &errorTransport{err: errors.New("connection reset by peer")}
	transport := newRetryTransport(base)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, "http://nexus.invalid/", io.NopCloser(strings.NewReader("payload")))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	resp, err := transport.RoundTrip(req)
	if resp != nil {
		_ = resp.Body.Close()
	}

	if err == nil {
		t.Fatal("RoundTrip() error = nil, want an error")
	}

	if base.calls != 1 {
		t.Errorf("RoundTrip() sent %d requests, want 1", base.calls)
	}
}

// TestNewClientRetries checks that the clients built by NewClient retry, with
// the credentials attached to every attempt.
func TestNewClientRetries(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "secret" {
			t.Errorf("attempt %d sent no or wrong credentials", calls.Load()+1)
		}

		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusBadGateway)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"role","name":"Role"}`))
	}))
	t.Cleanup(srv.Close)

	client, err := NewClient(Credentials{URL: srv.URL, Username: "admin", Password: "secret"})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	role, resp, err := client.SecurityManagementRolesAPI.GetSecurityRoles(context.Background(), "role").Execute()
	if resp != nil {
		_ = resp.Body.Close()
	}

	if err != nil {
		t.Fatalf("GetSecurityRoles() error = %v", err)
	}

	if role.GetId() != "role" || calls.Load() != 2 {
		t.Errorf("GetSecurityRoles() = %q after %d requests, want %q after 2", role.GetId(), calls.Load(), "role")
	}
}
