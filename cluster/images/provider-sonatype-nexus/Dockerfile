FROM gcr.io/distroless/static@sha256:d20ee35777d1be105385ebbbf989178e5d3d0d254059e8a09a38ef64aebc741a

ARG TARGETOS
ARG TARGETARCH

ADD bin/${TARGETOS}_${TARGETARCH}/provider /usr/local/bin/provider

USER 65532
ENTRYPOINT ["provider"]
