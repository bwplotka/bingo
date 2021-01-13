# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo v0.3.0. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
BINGO_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO     ?= $(shell which go)

# Bellow generated variables ensure that every time a tool under each variable is invoked, the correct version
# will be used; reinstalling only if needed.
# For example for copyright variable:
#
# In your main Makefile (for non array binaries):
#
#include .bingo/Variables.mk # Assuming -dir was set to .bingo .
#
#command: $(COPYRIGHT)
#	@echo "Running copyright"
#	@$(COPYRIGHT) <flags/args..>
#
COPYRIGHT := $(GOBIN)/copyright-v0.0.0-20210112004814-138d5e5695fe
$(COPYRIGHT): $(BINGO_DIR)/copyright.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/copyright-v0.0.0-20210112004814-138d5e5695fe"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=copyright.mod -o=$(GOBIN)/copyright-v0.0.0-20210112004814-138d5e5695fe "github.com/efficientgo/tools/copyright"

EMBEDMD := $(GOBIN)/embedmd-v1.0.0
$(EMBEDMD): $(BINGO_DIR)/embedmd.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/embedmd-v1.0.0"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=embedmd.mod -o=$(GOBIN)/embedmd-v1.0.0 "github.com/campoy/embedmd"

FAILLINT := $(GOBIN)/faillint-v1.5.0
$(FAILLINT): $(BINGO_DIR)/faillint.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/faillint-v1.5.0"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=faillint.mod -o=$(GOBIN)/faillint-v1.5.0 "github.com/fatih/faillint"

GOIMPORTS := $(GOBIN)/goimports-v0.0.0-20210112230658-8b4aab62c064
$(GOIMPORTS): $(BINGO_DIR)/goimports.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/goimports-v0.0.0-20210112230658-8b4aab62c064"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=goimports.mod -o=$(GOBIN)/goimports-v0.0.0-20210112230658-8b4aab62c064 "golang.org/x/tools/cmd/goimports"

GOLANGCI_LINT := $(GOBIN)/golangci-lint-v1.26.0
$(GOLANGCI_LINT): $(BINGO_DIR)/golangci-lint.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/golangci-lint-v1.26.0"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=golangci-lint.mod -o=$(GOBIN)/golangci-lint-v1.26.0 "github.com/golangci/golangci-lint/cmd/golangci-lint"

MDOX := $(GOBIN)/mdox-v0.2.0
$(MDOX): $(BINGO_DIR)/mdox.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/mdox-v0.2.0"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=mdox.mod -o=$(GOBIN)/mdox-v0.2.0 "github.com/bwplotka/mdox"

MISSPELL := $(GOBIN)/misspell-v0.3.4
$(MISSPELL): $(BINGO_DIR)/misspell.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/misspell-v0.3.4"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=misspell.mod -o=$(GOBIN)/misspell-v0.3.4 "github.com/client9/misspell/cmd/misspell"

PROXY := $(GOBIN)/proxy-v0.10.0
$(PROXY): $(BINGO_DIR)/proxy.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/proxy-v0.10.0"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=proxy.mod -o=$(GOBIN)/proxy-v0.10.0 "github.com/gomods/athens/cmd/proxy"

