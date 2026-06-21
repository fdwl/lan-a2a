.PHONY: build build-agent build-relay test lint clean fmt vet

build: build-agent build-relay

build-agent:
	go build -o lan-a2a ./cmd/lan-a2a

build-relay:
	go build -o lan-relay ./cmd/lan-relay

test:
	go test ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .

vet:
	go vet ./...

clean:
	rm -f lan-a2a lan-relay

docker-build:
	docker build --target agent -t lan-a2a:latest .
	docker build --target relay -t lan-relay:latest .

install:
	go install ./cmd/lan-a2a
	go install ./cmd/lan-relay
