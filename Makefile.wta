GO      ?= go
BIN_DIR ?= bin
CMDS    := $(notdir $(wildcard cmd/*))

.PHONY: all build test clean fmt vet

all: build

build:
	@mkdir -p $(BIN_DIR)
	@for c in $(CMDS); do \
		echo "==> building $$c"; \
		$(GO) build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/$$c ./cmd/$$c || exit 1; \
	done

test:
	$(GO) test ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

clean:
	rm -rf $(BIN_DIR)
