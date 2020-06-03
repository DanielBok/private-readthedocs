FROM golang:1.14-stretch as builder

WORKDIR /build

COPY . .
RUN go build -o sphinx .

FROM debian:buster
WORKDIR /app
RUN mkdir -p /var/readthedocs

COPY config.yml .
COPY --from=builder /build/sphinx   sphinx

ENTRYPOINT ["./sphinx"]
