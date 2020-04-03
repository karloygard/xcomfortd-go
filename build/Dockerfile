FROM golang:alpine3.11 as builder

RUN apk add --no-cache gcc musl-dev linux-headers

COPY . /xcomfortd-go

RUN \
    cd /xcomfortd-go && \
    go get && \
    go build

FROM alpine:3.11

COPY --from=builder /go/bin/xcomfortd-go /usr/bin/xcomfortd

CMD ["xcomfortd"]