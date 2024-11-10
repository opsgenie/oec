.PHONY: all
all: test build

.PHONY: build
build: fmt
	go build -a -o oec main/main.go

.PHONY: test
test:
	go test ./...

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: update
update:
	go get -u ./...
	go mod tidy
	go mod vendor

.PHONY: docker
docker:
	docker build -t oec:latest .
