// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package bingo

const (
	EnvBinVarsName = "variables.env"
	// TODO(bwplotka): Produce some scripts that would install if missing, same to makefile?
	envBinVarsTmpl = `# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo {{ .Version }}. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
# Those variables will work only until 'bingo get' was invoked, or if tools were installed via Makefile's Variables.mk.
GOBIN=${GOBIN:=$(go env GOBIN)}

if [ -z "$GOBIN" ]; then
	GOBIN="$(go env GOPATH)/bin"
fi

{{range $p := .MainPackages }}
{{ $p.EnvVarName }}="{{- range $i, $v := $p.Versions }}{{- if ne $i 0}} {{- end }}${GOBIN}/{{ $p.Name }}-{{ $v.Version }}{{- end }}"
{{ end}}
`
)
