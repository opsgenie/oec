FROM golang:1.21 as builder
ADD . $GOPATH/src/github.com/opsgenie/oec
WORKDIR $GOPATH/src/github.com/opsgenie/oec/main
RUN export GIT_COMMIT=$(git rev-list -1 HEAD) && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo \
        -ldflags "-X main.OECCommitVersion=$GIT_COMMIT -X main.OECVersion=1.0.1" -o nocgo -o /oec .
FROM python:alpine3.16 as base
RUN pip install requests
RUN addgroup -S opsgenie && \
    adduser -S opsgenie -G opsgenie && \
    apk update && \
    apk add --no-cache git ca-certificates && \
    update-ca-certificates
COPY --from=builder /oec /opt/oec
RUN mkdir -p /var/log/opsgenie && \
    chown -R opsgenie:opsgenie /var/log/opsgenie && \
    chown -R opsgenie:opsgenie /opt/oec
USER opsgenie
ENTRYPOINT ["/opt/oec"]