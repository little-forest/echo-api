FROM gcr.io/distroless/static-debian12

COPY echo-api /
ENTRYPOINT ["/echo-api"]
