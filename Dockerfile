FROM golang:1.11.4 as builder
ADD https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64 /usr/bin/dep
RUN chmod +x /usr/bin/dep
ADD . $GOPATH/src/github.com/opsgenie/oec
WORKDIR $GOPATH/src/github.com/opsgenie/oec/main
RUN dep init
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags "-X main.OECCommitVersion=7c15256c12761a133f76e9567e2e331260af9e5d -X main.OECVersion=1.0.0" -o nocgo -o /oec .
FROM alpine:latest as base
RUN addgroup -S opsgenie && adduser -S opsgenie -G opsgenie
RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates
COPY --from=builder /oec /opt/oec
RUN chown -R opsgenie:opsgenie /var/log
RUN chown -R opsgenie:opsgenie /opt/oec
USER opsgenie
ENTRYPOINT ["/opt/oec"]