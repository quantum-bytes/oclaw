.PHONY: build run clean test

build:
	go build -o oclaw .

run: build
	./oclaw

test:
	go test ./...

clean:
	rm -f oclaw

install:
	go install .

lint:
	golangci-lint run ./...
