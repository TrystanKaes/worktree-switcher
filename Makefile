BINARY    := worktree-switcher
MODULE    := github.com/trystankaes/worktree-switcher
BUILD_DIR := dist
VERSION   := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS   := -s -w -X main.Version=$(VERSION)

INSTALL_DIR ?= $(HOME)/.local/bin

.PHONY: build build-all clean install uninstall

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

# Build and install binary + shell integration
install: build
	INSTALL_DIR="$(INSTALL_DIR)" bash $(CURDIR)/install.sh ./$(BINARY)

# Remove the installed binary
uninstall:
	@if [[ -f "$(INSTALL_DIR)/$(BINARY)" ]]; then \
	  rm "$(INSTALL_DIR)/$(BINARY)"; \
	  echo "Removed $(INSTALL_DIR)/$(BINARY)"; \
	else \
	  echo "$(INSTALL_DIR)/$(BINARY) not found — nothing to remove"; \
	fi
	@echo ""
	@echo "Shell integration was not removed automatically."
	@echo "To finish uninstalling, remove the following lines from your shell profile:"
	@echo ""
	@echo '  export PATH="$(INSTALL_DIR):$$PATH"'
	@echo '  eval "$$(worktree-switcher init)"'

clean:
	rm -f $(BINARY)
	rm -rf $(BUILD_DIR)
