FROM golang:1.23 as builder

COPY . $GOPATH/src/github.com/opsgenie/oec

WORKDIR $GOPATH/src/github.com/opsgenie/oec/main
RUN export GIT_COMMIT=$(git rev-list -1 HEAD)

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo \
      -ldflags "-X main.OECCommitVersion=$GIT_COMMIT -X main.OECVersion=1.0.1" -o nocgo -o /oec .


FROM python:alpine3.20 as base
RUN --mount=type=bind,source=requirements.txt,target=requirements.txt pip install -r requirements.txt

RUN apk update && \
    apk add --virtual --no-cache git ca-certificates && \
    update-ca-certificates

RUN addgroup -S opsgenie && \
    adduser -S opsgenie -G opsgenie && \
    mkdir -p /var/log/opsgenie && \
    chown -R opsgenie:opsgenie /var/log/opsgenie

COPY --from=builder --chown=opsgenie:opsgenie /oec /opt/oec

USER opsgenie

ENTRYPOINT ["/opt/oec"]
