.PHONY: build install uninstall test smoke clean

BINARY := narrc
CMD := ./cmd/narrc
GOBIN ?= $(shell go env GOBIN)
GOPATH ?= $(shell go env GOPATH)
INSTALL_DIR := $(if $(GOBIN),$(GOBIN),$(firstword $(subst :, ,$(GOPATH)))/bin)
INSTALL_BIN := $(INSTALL_DIR)/$(BINARY)

build:
	go build -o bin/$(BINARY) $(CMD)

install:
	GOBIN="$(GOBIN)" go install $(CMD)

uninstall:
	rm -f "$(INSTALL_BIN)"

test:
	go test ./...

smoke:
	go run $(CMD) --version

clean:
	rm -rf bin
