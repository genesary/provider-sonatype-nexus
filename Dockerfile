# Build the provider binary
FROM golang:1.21 AS builder

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the source code
COPY apis/ apis/
COPY cmd/ cmd/
COPY internal/ internal/
COPY hack/ hack/

# Build
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -a -o /provider ./cmd/provider

# Use distroless as minimal base image to package the provider binary
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /provider .
USER 65532:65532

ENTRYPOINT ["/provider"]
