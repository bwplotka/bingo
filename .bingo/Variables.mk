# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo v0.1.0.rc.4. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
GOBIN ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO    ?= $(shell which go)

# Bellow generated variables ensure that every time a tool under each variable is invoked, the correct version
# will be used; reinstalling only if needed.
# For example for copyright variable:
#
# In your main Makefile (for non array binaries):
#
#include .bingo/Variables.mk # (If not generated automatically by bingo).
#
#command: $(COPYRIGHT)
#	@echo "Running copyright"
#	@$(COPYRIGHT) <flags/args..>
#
COPYRIGHT ?= $(GOBIN)/copyright-v0.9.0
$(COPYRIGHT): .bingo/copyright.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/copyright-v0.9.0"
	@$(GO) build -modfile=.bingo/copyright.mod -o=$(GOBIN)/copyright-v0.9.0 "github.com/bwplotka/flagarize/scripts/copyright"
.bingo/copyright.mod: ;

FAILLINT ?= $(GOBIN)/faillint-v1.5.0
$(FAILLINT): .bingo/faillint.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/faillint-v1.5.0"
	@$(GO) build -modfile=.bingo/faillint.mod -o=$(GOBIN)/faillint-v1.5.0 "github.com/fatih/faillint"
.bingo/faillint.mod: ;

GOIMPORTS ?= $(GOBIN)/goimports-v0.0.0-20200509030707-2212a7e161a5
$(GOIMPORTS): .bingo/goimports.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/goimports-v0.0.0-20200509030707-2212a7e161a5"
	@$(GO) build -modfile=.bingo/goimports.mod -o=$(GOBIN)/goimports-v0.0.0-20200509030707-2212a7e161a5 "golang.org/x/tools/cmd/goimports"
.bingo/goimports.mod: ;

GOLANGCI_LINT ?= $(GOBIN)/golangci-lint-v1.26.0
$(GOLANGCI_LINT): .bingo/golangci-lint.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/golangci-lint-v1.26.0"
	@$(GO) build -modfile=.bingo/golangci-lint.mod -o=$(GOBIN)/golangci-lint-v1.26.0 "github.com/golangci/golangci-lint/cmd/golangci-lint"
.bingo/golangci-lint.mod: ;

MISSPELL ?= $(GOBIN)/misspell-v0.3.4
$(MISSPELL): .bingo/misspell.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/misspell-v0.3.4"
	@$(GO) build -modfile=.bingo/misspell.mod -o=$(GOBIN)/misspell-v0.3.4 "github.com/client9/misspell/cmd/misspell"
.bingo/misspell.mod: ;

