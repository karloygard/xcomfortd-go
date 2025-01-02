# Stage 1: Build
FROM golang:alpine3.20 as builder

RUN apk add --no-cache --update gcc musl-dev linux-headers pkgconfig libusb-dev

COPY . /xcomfortd-go

WORKDIR /xcomfortd-go
RUN go get
RUN go install

#RUN chmod +x entrypoint.sh

# Stage 2: Runtime
FROM alpine:3.20

RUN apk add --no-cache --update libusb bash

# Copy the built binary
COPY --from=builder /go/bin/xcomfortd-go /usr/bin/xcomfortd

# Make the entrypoint script executable
COPY build/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Set the entrypoint
ENTRYPOINT ["/entrypoint.sh"]

# Default command
CMD ["xcomfortd"]
