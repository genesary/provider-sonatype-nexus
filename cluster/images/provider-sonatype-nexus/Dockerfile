FROM gcr.io/distroless/static@sha256:15a69c654ed239b3faf5bc3725ff1dd580462eb882c7d5b9c02cdf37756657c2

ARG TARGETOS
ARG TARGETARCH

ADD bin/${TARGETOS}_${TARGETARCH}/provider /usr/local/bin/provider

USER 65532
ENTRYPOINT ["provider"]
