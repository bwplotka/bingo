# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo v0.2.0. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO     ?= $(shell which go)

# Bellow generated variables ensure that every time a tool under each variable is invoked, the correct version
# will be used; reinstalling only if needed.
# For example for faillint variable:
#
# In your main Makefile (for non array binaries):
#
#include .bingo/Variables.mk # Assuming -dir was set to .bingo .
#
#command: $(FAILLINT)
#	@echo "Running faillint"
#	@$(FAILLINT) <flags/args..>
#
FAILLINT := $(GOBIN)/faillint-v1.3.0
$(FAILLINT): .bingo/faillint.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/faillint-v1.3.0"
	@cd .bingo && $(GO) build -modfile=faillint.mod -o=$(GOBIN)/faillint-v1.3.0 "github.com/fatih/faillint"

GOIMPORTS := $(GOBIN)/goimports-v0.0.0-20200522201501-cb1345f3a375
$(GOIMPORTS): .bingo/goimports.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/goimports-v0.0.0-20200522201501-cb1345f3a375"
	@cd .bingo && $(GO) build -modfile=goimports.mod -o=$(GOBIN)/goimports-v0.0.0-20200522201501-cb1345f3a375 "golang.org/x/tools/cmd/goimports"

GOIMPORTS2 := $(GOBIN)/goimports2-v0.0.0-20200519175826-7521f6f42533
$(GOIMPORTS2): .bingo/goimports2.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/goimports2-v0.0.0-20200519175826-7521f6f42533"
	@cd .bingo && $(GO) build -modfile=goimports2.mod -o=$(GOBIN)/goimports2-v0.0.0-20200519175826-7521f6f42533 "golang.org/x/tools/cmd/goimports"

