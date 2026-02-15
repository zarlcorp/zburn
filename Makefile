.PHONY: build test lint run clean

build:
	go build -o bin/zburn ./cmd/zburn

test:
	go test -race ./...

lint:
	golangci-lint run

run:
	go run ./cmd/zburn

clean:
	rm -rf bin/
