FROM gcr.io/distroless/static-debian13

ARG TARGETPLATFORM
COPY $TARGETPLATFORM/echo-api /
ENTRYPOINT ["/echo-api"]
