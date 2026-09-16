VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GOFLAGS  = -trimpath -buildvcs=false -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build test vet fmt-check check
build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -o moneroid ./cmd/moneroid
test:
	go test -race ./...
vet:
	go vet ./...
fmt-check:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; echo "gofmt: files need formatting"; exit 1; }
check: fmt-check vet test build
