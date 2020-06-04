// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package bingo

const (
	MakefileBinVarsName = "Variables.mk"
	// TODO(bwplotka): We might want to play with better escaping to allow spaces in dir names.
	// TODO(bwplotka): We get first binary as an example. It does not work if first one is array..
	makefileBinVarsTmpl = `# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo {{ .Version }}. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO     ?= $(shell which go)

# Bellow generated variables ensure that every time a tool under each variable is invoked, the correct version
# will be used; reinstalling only if needed.
# For example for {{ with (index .Binaries 0) }}{{ .Name }}{{ end }} variable:
#
# In your main Makefile (for non array binaries):
#
#include .bingo/Variables.mk # Assuming -dir was set to .bingo .
#
#command: $({{ with (index .Binaries 0) }}{{ .VarName }}{{ end }})
#	@echo "Running {{ with (index .Binaries 0) }}{{ .Name }}{{ end }}"
#	@$({{ with (index .Binaries 0) }}{{ .VarName }}{{ end }}) <flags/args..>
#
{{- range $b := .Binaries }}
{{ $b.VarName }} :={{- range $b.Versions }} $(GOBIN)/{{ .BinName }}{{- end }}
$({{ $b.VarName }}):{{- range $b.Versions }} {{ $.RelModDir }}/{{ .ModFile }}{{- end }}
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
{{- range $b.Versions }}
	@echo "(re)installing $(GOBIN)/{{ .BinName }}"
	@cd {{ $.RelModDir }} && $(GO) build -modfile={{ .ModFile }} -o=$(GOBIN)/{{ .BinName }} "{{ $b.PackagePath }}"
{{- end }}
{{ end}}
`
)
