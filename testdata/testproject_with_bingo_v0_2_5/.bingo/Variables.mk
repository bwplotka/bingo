# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo v0.2.5. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
BINGO_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO     ?= $(shell which go)

# Bellow generated variables ensure that every time a tool under each variable is invoked, the correct version
# will be used; reinstalling only if needed.
# For example for f2 variable:
#
# In your main Makefile (for non array binaries):
#
#include .bingo/Variables.mk # Assuming -dir was set to .bingo .
#
#command: $(F2_ARRAY)
#	@echo "Running f2"
#	@$(F2_ARRAY) <flags/args..>
#
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

GOIMPORTS := $(GOBIN)/goimports-v0.0.0-20200522201501-cb1345f3a375
$(GOIMPORTS): $(BINGO_DIR)/goimports.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/goimports-v0.0.0-20200522201501-cb1345f3a375"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=goimports.mod -o=$(GOBIN)/goimports-v0.0.0-20200522201501-cb1345f3a375 "golang.org/x/tools/cmd/goimports"

GOIMPORTS2 := $(GOBIN)/goimports2-v0.0.0-20200519175826-7521f6f42533
$(GOIMPORTS2): $(BINGO_DIR)/goimports2.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/goimports2-v0.0.0-20200519175826-7521f6f42533"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=goimports2.mod -o=$(GOBIN)/goimports2-v0.0.0-20200519175826-7521f6f42533 "golang.org/x/tools/cmd/goimports"

