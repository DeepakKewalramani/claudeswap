BINARY_NAME=claudeswap
VERSION=1.0.0
BUILD_DIR=dist
MAIN_PACKAGE=./cmd/claudeswap

.PHONY: all build test vet lint clean release install

all: vet test build

build:
	@mkdir -p bin
	go build -ldflags "-s -w" -o bin/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "✓ Built bin/$(BINARY_NAME)"

test:
	go test -v ./...

vet:
	go vet ./...

clean:
	rm -rf bin $(BUILD_DIR)

install: build
	@mkdir -p $(HOME)/.claudeswap/bin
	cp bin/$(BINARY_NAME) $(HOME)/.claudeswap/bin/$(BINARY_NAME)
	@echo "✓ Installed $(BINARY_NAME) to $(HOME)/.claudeswap/bin"

release: clean
	@mkdir -p $(BUILD_DIR)
	@echo "Building release matrix for v$(VERSION)..."
	GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-arm64 $(MAIN_PACKAGE)
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-amd64 $(MAIN_PACKAGE)
	GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-linux-arm64 $(MAIN_PACKAGE)
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64 $(MAIN_PACKAGE)
	GOOS=windows GOARCH=arm64 go build -ldflags "-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-windows-arm64.exe $(MAIN_PACKAGE)
	GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-windows-amd64.exe $(MAIN_PACKAGE)
	@echo "✓ Cross-compilation complete! Artifacts in $(BUILD_DIR)/"
