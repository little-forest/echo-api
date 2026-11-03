FROM gcr.io/distroless/static-debian12

ARG TARGETPLATFORM
COPY $TARGETPLATFORM/echo-api /
ENTRYPOINT ["/echo-api"]
