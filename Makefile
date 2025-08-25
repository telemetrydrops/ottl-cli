.PHONY: build test clean example fmt lint lint-go lint-markdown lint-yaml lint-all

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
	echo 'set(attributes["example"], "true")' | $(BUILD_DIR)/$(BINARY_NAME) transform --input-file ./testdata/traces.json

fmt:
	go fmt ./...

lint: lint-go

lint-go:
	golangci-lint run

lint-markdown:
	@echo "Checking markdown files..."
	@if command -v markdownlint > /dev/null 2>&1; then \
		markdownlint '**/*.md' --ignore node_modules --ignore vendor --ignore 'local/tasks/**'; \
	else \
		echo "markdownlint not found. Install with: npm install -g markdownlint-cli"; \
		exit 1; \
	fi

lint-yaml:
	@echo "Checking YAML files..."
	@if command -v yamllint > /dev/null 2>&1; then \
		yamllint .github/ .goreleaser.yaml; \
	else \
		echo "yamllint not found. Install with: pip install yamllint"; \
		exit 1; \
	fi

lint-all: lint-go lint-markdown lint-yaml
	@echo "All linting checks passed!"