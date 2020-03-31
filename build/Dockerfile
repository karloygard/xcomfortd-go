# build stage
FROM golang AS builder
COPY . /build
WORKDIR /build
RUN CGO_ENABLED=0 GOOS=linux go build -o xcomfortd-go
# second stage
FROM alpine:latest
WORKDIR /root/
RUN mkdir -p /etc/ssl/certs \
    && apk --no-cache add ca-certificates libusb \
    && update-ca-certificates
COPY --from=builder /build .
CMD ["./xcomfortd-go"]
