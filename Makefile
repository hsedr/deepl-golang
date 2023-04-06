LDFLAGS=-ldflags "-s -w"

.PHONY: build serve test check

all: check serve test build

build: ## Build all executables
	@echo "Building..."
	@CGO_ENABLED=0 go build -o .bin/example -trimpath $(LDFLAGS) ./example/... 
	@if test ! -e ./cmd/deepl-mock/index.js ; then \
		git clone https://github.com/DeepLcom/deepl-mock ./cmd/deepl-mock ; \
		npm install --prefix ./cmd/deepl-mock ; \
	fi

serve: ## Run Deepl-Mock Server
	@echo "Running Deepl-Mock Server..."
	@npm start --prefix ./cmd/deepl-mock

test: ## Run Go Tests
	@go test -v -shuffle=on ./...

check: ## Check Go Code
	@golangci-lint run -v ./...
	@govulncheck ./...
