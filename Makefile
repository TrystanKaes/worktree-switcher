BINARY    := worktree-switcher
DEV_BINARY := worktree-switcher-dev
MODULE    := github.com/trystankaes/worktree-switcher
BUILD_DIR := dist
VERSION   := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS   := -s -w -X main.Version=$(VERSION)

INSTALL_DIR ?= $(HOME)/.local/bin

.PHONY: build build-all build-dev install-dev dev-aliases clean clean-dev install uninstall

# Build for the current platform
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

# Build and install as worktree-switcher-dev so it coexists with the Homebrew formula.
# Shell usage:  wt-dev list / wt-dev interactive  (add alias to shell profile)
build-dev:
	go build -ldflags "$(LDFLAGS)" -o $(DEV_BINARY) .

dev-aliases:
	@bash scripts/dev-aliases.sh

install-dev: build-dev
	install -d "$(INSTALL_DIR)"
	install -m 755 $(DEV_BINARY) "$(INSTALL_DIR)/$(DEV_BINARY)"
	@echo "Installed $(INSTALL_DIR)/$(DEV_BINARY)"
	@echo "Run: source <(make dev-aliases)"

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

clean-dev:
	@if [[ -f "$(INSTALL_DIR)/$(DEV_BINARY)" ]]; then \
	  rm "$(INSTALL_DIR)/$(DEV_BINARY)"; \
	  echo "Removed $(INSTALL_DIR)/$(DEV_BINARY)"; \
	else \
	  echo "$(INSTALL_DIR)/$(DEV_BINARY) not found — nothing to remove"; \
	fi
	@echo "Run: unalias wt-dev wti-dev wts-dev 2>/dev/null; unset -f wt-dev wti-dev wts-dev 2>/dev/null"

clean:
	rm -f $(BINARY) $(DEV_BINARY)
	rm -rf $(BUILD_DIR)
