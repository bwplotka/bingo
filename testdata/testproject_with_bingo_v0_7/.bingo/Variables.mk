# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo v0.7. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
BINGO_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO     ?= $(shell which go)

# Below generated variables ensure that every time a tool under each variable is invoked, the correct version
# will be used; reinstalling only if needed.
# For example for buildable-v2 variable:
#
# In your main Makefile (for non array binaries):
#
#include .bingo/Variables.mk # Assuming -dir was set to .bingo .
#
#command: $(BUILDABLE_V2)
#	@echo "Running buildable-v2"
#	@$(BUILDABLE_V2) <flags/args..>
#
BUILDABLE_V2 := $(GOBIN)/buildable-v2-v2.0.0
$(BUILDABLE_V2): $(BINGO_DIR)/buildable-v2.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/buildable-v2-v2.0.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=buildable-v2.mod -o=$(GOBIN)/buildable-v2-v2.0.0 "github.com/bwplotka/bingo-testmodule/v2/buildable"

BUILDABLE_WITHREPLACE := $(GOBIN)/buildable-withReplace-v0.0.0-20221007091003-fe4d42a37d92
$(BUILDABLE_WITHREPLACE): $(BINGO_DIR)/buildable-withReplace.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/buildable-withReplace-v0.0.0-20221007091003-fe4d42a37d92"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=buildable-withReplace.mod -o=$(GOBIN)/buildable-withReplace-v0.0.0-20221007091003-fe4d42a37d92 "github.com/bwplotka/bingo-testmodule/buildable"

BUILDABLE_ARRAY := $(GOBIN)/buildable-v0.0.0-20221007091146-39a7f0ae0b1e $(GOBIN)/buildable-v1.0.0 $(GOBIN)/buildable-v1.1.0
$(BUILDABLE_ARRAY): $(BINGO_DIR)/buildable.mod $(BINGO_DIR)/buildable.1.mod $(BINGO_DIR)/buildable.2.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/buildable-v0.0.0-20221007091146-39a7f0ae0b1e"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=buildable.mod -o=$(GOBIN)/buildable-v0.0.0-20221007091146-39a7f0ae0b1e "github.com/bwplotka/bingo-testmodule/buildable"
	@echo "(re)installing $(GOBIN)/buildable-v1.0.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=buildable.1.mod -o=$(GOBIN)/buildable-v1.0.0 "github.com/bwplotka/bingo-testmodule/buildable"
	@echo "(re)installing $(GOBIN)/buildable-v1.1.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=buildable.2.mod -o=$(GOBIN)/buildable-v1.1.0 "github.com/bwplotka/bingo-testmodule/buildable"

BUILDABLE2 := $(GOBIN)/buildable2-v0.0.0-20221007091238-9d83f47b84c5
$(BUILDABLE2): $(BINGO_DIR)/buildable2.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/buildable2-v0.0.0-20221007091238-9d83f47b84c5"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=buildable2.mod -o=$(GOBIN)/buildable2-v0.0.0-20221007091238-9d83f47b84c5 "github.com/bwplotka/bingo-testmodule/buildable2"

