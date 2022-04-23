# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo v0.5.2. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
BINGO_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO     ?= $(shell which go)

# Below generated variables ensure that every time a tool under each variable is invoked, the correct version
# will be used; reinstalling only if needed.
# For example for buildable variable:
#
# In your main Makefile (for non array binaries):
#
#include .bingo/Variables.mk # Assuming -dir was set to .bingo .
#
#command: $(BUILDABLE)
#	@echo "Running buildable"
#	@$(BUILDABLE) <flags/args..>
#
BUILDABLE := $(GOBIN)/buildable-v0.0.0-20210109094001-375d0606849d
$(BUILDABLE): $(BINGO_DIR)/buildable.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/buildable-v0.0.0-20210109094001-375d0606849d"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=buildable.mod -o=$(GOBIN)/buildable-v0.0.0-20210109094001-375d0606849d "github.com/bwplotka/bingo/testdata/module/buildable"

BUILDABLE2 := $(GOBIN)/buildable2-v0.0.0-20210109093942-2e6391144e85
$(BUILDABLE2): $(BINGO_DIR)/buildable2.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/buildable2-v0.0.0-20210109093942-2e6391144e85"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=buildable2.mod -o=$(GOBIN)/buildable2-v0.0.0-20210109093942-2e6391144e85 "github.com/bwplotka/bingo/testdata/module/buildable2"

BUILDABLE_OLD := $(GOBIN)/buildable_old-v0.0.0-20210109093942-2e6391144e85
$(BUILDABLE_OLD): $(BINGO_DIR)/buildable_old.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/buildable_old-v0.0.0-20210109093942-2e6391144e85"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=buildable_old.mod -o=$(GOBIN)/buildable_old-v0.0.0-20210109093942-2e6391144e85 "github.com/bwplotka/bingo/testdata/module/buildable"

F2_ARRAY := $(GOBIN)/f2-v1.5.0 $(GOBIN)/f2-v1.1.0 $(GOBIN)/f2-v1.2.0 $(GOBIN)/f2-v1.0.0
$(F2_ARRAY): $(BINGO_DIR)/f2.mod $(BINGO_DIR)/f2.1.mod $(BINGO_DIR)/f2.2.mod $(BINGO_DIR)/f2.3.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/f2-v1.5.0"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=f2.mod -o=$(GOBIN)/f2-v1.5.0 "github.com/fatih/faillint"
	@echo "(re)installing $(GOBIN)/f2-v1.1.0"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=f2.1.mod -o=$(GOBIN)/f2-v1.1.0 "github.com/fatih/faillint"
	@echo "(re)installing $(GOBIN)/f2-v1.2.0"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=f2.2.mod -o=$(GOBIN)/f2-v1.2.0 "github.com/fatih/faillint"
	@echo "(re)installing $(GOBIN)/f2-v1.0.0"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=f2.3.mod -o=$(GOBIN)/f2-v1.0.0 "github.com/fatih/faillint"

FAILLINT := $(GOBIN)/faillint-v1.3.0
$(FAILLINT): $(BINGO_DIR)/faillint.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/faillint-v1.3.0"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=faillint.mod -o=$(GOBIN)/faillint-v1.3.0 "github.com/fatih/faillint"

GO_BINDATA := $(GOBIN)/go-bindata-v3.1.1+incompatible
$(GO_BINDATA): $(BINGO_DIR)/go-bindata.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/go-bindata-v3.1.1+incompatible"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=go-bindata.mod -o=$(GOBIN)/go-bindata-v3.1.1+incompatible "github.com/go-bindata/go-bindata/go-bindata"

WR_BUILDABLE := $(GOBIN)/wr_buildable-v0.0.0-20210109165512-ccbd4039b94a
$(WR_BUILDABLE): $(BINGO_DIR)/wr_buildable.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/wr_buildable-v0.0.0-20210109165512-ccbd4039b94a"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=wr_buildable.mod -o=$(GOBIN)/wr_buildable-v0.0.0-20210109165512-ccbd4039b94a "github.com/bwplotka/bingo/testdata/module_with_replace/buildable"

