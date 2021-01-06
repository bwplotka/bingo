package bingo

var (
	templatesByFileExt = map[string]string{
		// TODO(bwplotka): We might want to play with better escaping to allow spaces in dir names.
		// TODO(bwplotka): We get first binary as an example. It does not work if first one is array.
		"mk": `# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo {{ .Version }}. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
BINGO_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO     ?= $(shell which go)

# Bellow generated variables ensure that every time a tool under each variable is invoked, the correct version
# will be used; reinstalling only if needed.
# For example for {{ with (index .MainPackages 0) }}{{ .Name }}{{ end }} variable:
#
# In your main Makefile (for non array binaries):
#
#include .bingo/Variables.mk # Assuming -dir was set to .bingo .
#
#command: $({{ with (index .MainPackages 0) }}{{ .EnvVarName }}{{ end }})
#	@echo "Running {{ with (index .MainPackages 0) }}{{ .Name }}{{ end }}"
#	@$({{ with (index .MainPackages 0) }}{{ .EnvVarName }}{{ end }}) <flags/args..>
#
{{- range $p := .MainPackages }}
{{ $p.EnvVarName }} :={{- range $p.Versions }} $(GOBIN)/{{ $p.Name }}-{{ .Version }}{{- end }}
$({{ $p.EnvVarName }}):{{- range $p.Versions }} $(BINGO_DIR)/{{ .ModFile }}{{- end }}
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
{{- range $p.Versions }}
	@echo "(re)installing $(GOBIN)/{{ $p.Name }}-{{ .Version }}"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile={{ .ModFile }} -o=$(GOBIN)/{{ $p.Name }}-{{ .Version }} "{{ $p.PackagePath }}"
{{- end }}
{{ end}}
`,
		"env": `# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo {{ .Version }}. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
# Those variables will work only until 'bingo get' was invoked, or if tools were installed via Makefile's Variables.mk.
GOBIN=${GOBIN:=$(go env GOBIN)}

if [ -z "$GOBIN" ]; then
	GOBIN="$(go env GOPATH)/bin"
fi

{{range $p := .MainPackages }}
{{ $p.EnvVarName }}="{{- range $i, $v := $p.Versions }}{{- if ne $i 0}} {{- end }}${GOBIN}/{{ $p.Name }}-{{ $v.Version }}{{- end }}"
{{ end}}
`,
		"go": `// Auto generated binary variables helper managed by https://github.com/bwplotka/bingo {{ .Version }}. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
# Those variables will work only until 'bingo get' was invoked, or if tools were installed via Makefile's Variables.mk.
package bingovars

import "os"

vars (
{{range $p := .MainPackages }}
{{ $p.EnvVarName }} = "{{- range $i, $v := $p.Versions }}{{- if ne $i 0}} {{- end }}os.Getenv("GOBIN") + "/{{ $p.Name }}-{{ $v.Version }}"{{- end }}"
{{ end}}
)
`}
)
