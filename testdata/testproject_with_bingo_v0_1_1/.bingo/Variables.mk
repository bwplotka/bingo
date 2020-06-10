# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo v0.1.1. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO     ?= $(shell which go)

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
F2_ARRAY ?= $(GOBIN)/f2-v1.5.0 $(GOBIN)/f2-v1.1.0 $(GOBIN)/f2-v1.2.0 $(GOBIN)/f2-v1.0.0
$(F2_ARRAY): .bingo/f2.mod .bingo/f2.1.mod .bingo/f2.2.mod .bingo/f2.3.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/f2-v1.5.0"
	@cd .bingo && $(GO) build -modfile=f2.mod -o=$(GOBIN)/f2-v1.5.0 "github.com/fatih/faillint"
	@echo "(re)installing $(GOBIN)/f2-v1.1.0"
	@cd .bingo && $(GO) build -modfile=f2.1.mod -o=$(GOBIN)/f2-v1.1.0 "github.com/fatih/faillint"
	@echo "(re)installing $(GOBIN)/f2-v1.2.0"
	@cd .bingo && $(GO) build -modfile=f2.2.mod -o=$(GOBIN)/f2-v1.2.0 "github.com/fatih/faillint"
	@echo "(re)installing $(GOBIN)/f2-v1.0.0"
	@cd .bingo && $(GO) build -modfile=f2.3.mod -o=$(GOBIN)/f2-v1.0.0 "github.com/fatih/faillint"
.bingo/f2.mod: ;
.bingo/f2.1.mod: ;
.bingo/f2.2.mod: ;
.bingo/f2.3.mod: ;

FAILLINT ?= $(GOBIN)/faillint-v1.3.0
$(FAILLINT): .bingo/faillint.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/faillint-v1.3.0"
	@cd .bingo && $(GO) build -modfile=faillint.mod -o=$(GOBIN)/faillint-v1.3.0 "github.com/fatih/faillint"
.bingo/faillint.mod: ;

GO_BINDATA ?= $(GOBIN)/go-bindata-v3.1.1+incompatible
$(GO_BINDATA): .bingo/go-bindata.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/go-bindata-v3.1.1+incompatible"
	@cd .bingo && $(GO) build -modfile=go-bindata.mod -o=$(GOBIN)/go-bindata-v3.1.1+incompatible "github.com/go-bindata/go-bindata/go-bindata"
.bingo/go-bindata.mod: ;

GOIMPORTS ?= $(GOBIN)/goimports-v0.0.0-20200522201501-cb1345f3a375
$(GOIMPORTS): .bingo/goimports.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/goimports-v0.0.0-20200522201501-cb1345f3a375"
	@cd .bingo && $(GO) build -modfile=goimports.mod -o=$(GOBIN)/goimports-v0.0.0-20200522201501-cb1345f3a375 "golang.org/x/tools/cmd/goimports"
.bingo/goimports.mod: ;

GOIMPORTS2 ?= $(GOBIN)/goimports2-v0.0.0-20200519175826-7521f6f42533
$(GOIMPORTS2): .bingo/goimports2.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/goimports2-v0.0.0-20200519175826-7521f6f42533"
	@cd .bingo && $(GO) build -modfile=goimports2.mod -o=$(GOBIN)/goimports2-v0.0.0-20200519175826-7521f6f42533 "golang.org/x/tools/cmd/goimports"
.bingo/goimports2.mod: ;
