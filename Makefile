.PHONY: build run clean test fmt vet lint install

build:
	go build -o oclaw .

run: build
	./oclaw

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -f oclaw

install:
	go install .

lint:
	golangci-lint run ./...
