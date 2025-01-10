.PHONY: all local clean go-mod-tidy build-arch build-all-arch

# Color Control Sequences for easy printing
_RESET=\033[0m
_RED=\033[31;1m
_GREEN=\033[32;1m
_YELLOW=\033[33;1m
_BLUE=\033[34;1m
_MAGENTA=\033[35;1m
_CYAN=\033[36;1m
_WHITE=\033[37;1m

# Variables
_PKG_NAME=og

_CURRENT_DIR := $(shell pwd)
_ROOT_DIR ?= ${_CURRENT_DIR}
_BIN_DIR ?= ${_ROOT_DIR}/bin
_TIMESTAMP_NOW_RFC3339:=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
_TIMESTAMP_NOW_STR:=$(shell date +%Y_%m_%d_%H%M%S)
_GLOBAL_BIN_DIR ?= $(HOME)/Development/bin
# ^ Specific to my machine

_GO=go
_GOOS ?= $(shell go env GOOS)
_GOARCH ?= $(shell go env GOARCH)
_GOBIN ?= ${GOBIN}
_BIN_NAME = ${_PKG_NAME}.${_GOOS}_${_GOARCH}
_GO_BUILD_LDL_FLAGS = -ldflags "-X main._buildTimeCompiledAtStr=${_TIMESTAMP_NOW_RFC3339}"
# ^ Specific to this project, otherwise empty. 
_GO_BUILD_CMD=$(_GO) build -o ${_BIN_DIR}/${_BIN_NAME} ${_GO_BUILD_LDL_FLAGS} -v cmd/${_PKG_NAME}/main.go

# Group commands: do more than one thing at once

all: build-all-arch local

local: go-mod-tidy build-arch
	cp ${_BIN_DIR}/${_BIN_NAME} ${_GLOBAL_BIN_DIR}/${_PKG_NAME}

# Main commands

clean:
	@echo "$(_YELLOW)Removing all existing binaries (if any)...$(_RESET)"
	rm -rf ${_BIN_DIR}/*

go-mod-tidy:
	@echo "$(_YELLOW)Running go mod tidy...$(_RESET)" && \
	cd ${_ROOT_DIR} && $(_GO) mod tidy && cd ${_CURRENT_DIR}

build-arch:
	@echo "$(_YELLOW)Compiling... (OS: ${_GOOS}, ARCH: ${_GOARCH})$(_RESET)" && \
	env GOOS=$(_GOOS) GOARCH=$(_GOARCH) ${_GO_BUILD_CMD} && \
	${_BIN_DIR}/${_BIN_NAME} --version

# Build binaries for all OS and ARCH.
# Make a list of all GOOS and GOARCH. Most containers/VMs use linux/amd64.
OS_LIST := linux darwin
ARCH_LIST := amd64 arm64
build-all-arch: go-mod-tidy
	@echo "$(_YELLOW)Compiling for all OS (OS: ${OS_LIST}, ARCH: ${ARCH_LIST})$(_RESET)"
	@$(foreach OS,$(OS_LIST), \
		$(foreach ARCH,$(ARCH_LIST), \
			$(MAKE) build-arch _GOOS=$(OS) _GOARCH=$(ARCH);))