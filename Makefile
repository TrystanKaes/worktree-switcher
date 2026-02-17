BINARY    := wt-bin
MODULE    := github.com/trystankaes/worktree-switcher
BUILD_DIR := dist
VERSION   := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS   := -s -w

.PHONY: build build-all clean install

# Build for the current platform
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

# Build for multiple platforms
build-all: $(BUILD_DIR)
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 .
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 .
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-amd64 .

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

# Install binary to GOPATH/bin and configure shell integration
install: build
	@DEST=$${GOPATH:-$$HOME/go}/bin; \
	cp $(BINARY) "$$DEST/$(BINARY)"; \
	echo "Installed $(BINARY) to $$DEST/$(BINARY)"
	@bash $(CURDIR)/install.sh

clean:
	rm -f $(BINARY)
	rm -rf $(BUILD_DIR)
