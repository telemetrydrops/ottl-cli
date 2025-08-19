.PHONY: build test clean

BINARY_NAME := ottl
BUILD_DIR := ./bin

build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/ottl

test:
	go test -v ./...

clean:
	rm -rf $(BUILD_DIR)

example:
	echo 'set(attributes["example"], "true")' | $(BUILD_DIR)/$(BINARY_NAME) transform --input-file ./local/payload-examples/trace.json

fmt:
	go fmt ./...

lint:
	golangci-lint run