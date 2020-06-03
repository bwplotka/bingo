// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package bingo

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/bwplotka/bingo/pkg/makefile"
	"github.com/pkg/errors"
)

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
#include .bingo/Variables.mk # (If not generated automatically by bingo).
#
#command: $({{ with (index .Binaries 0) }}{{ .VarName }}{{ end }})
#	@echo "Running {{ with (index .Binaries 0) }}{{ .Name }}{{ end }}"
#	@$({{ with (index .Binaries 0) }}{{ .VarName }}{{ end }}) <flags/args..>
#
{{- range $b := .Binaries }}
{{ $b.VarName }} ?={{- range $b.Versions }} $(GOBIN)/{{ .BinName }}{{- end }}
$({{ $b.VarName }}):{{- range $b.Versions }} {{ .RelModFile }}{{- end }}
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
{{- range $b.Versions }}
	@echo "(re)installing $(GOBIN)/{{ .BinName }}"
	@$(GO) build -modfile={{ .RelModFile }} -o=$(GOBIN)/{{ .BinName }} "{{ $b.PackagePath }}"
{{- end }}
{{- range $b.Versions }}
{{ .RelModFile }}: ;
{{- end }}
{{ end}}
`
)

type binaryVersion struct {
	BinName    string
	RelModFile string
}

type binary struct {
	Name        string
	VarName     string
	PackagePath string
	Versions    []binaryVersion
}

// RemoveMakeHelper deletes Variables.mk from mod directory.
func RemoveMakeHelper(modDir string) error {
	// TODO(bwplotka): This will NOT remove include, detect this?
	return os.RemoveAll(filepath.Join(modDir, MakefileBinVarsName))
}

// NameFromModFile returns binary name from module file path.
func NameFromModFile(modFile string) (name string, oneOfMany bool) {
	n := strings.Split(strings.TrimSuffix(filepath.Base(modFile), ".mod"), ".")
	if len(n) > 1 {
		oneOfMany = true
	}
	return n[0], oneOfMany
}

// GenMakeHelper generates helper Makefile variables to allows reliable binaries use. Regenerate if needed.
// It is expected to have at least one mod file.
func GenMakeHelperAndHook(modDir, makeFile, version string, modFiles ...string) error {
	makefileBinVarsFile := filepath.Join(modDir, MakefileBinVarsName)
	if len(modFiles) == 0 {
		return errors.New("no mod files")
	}

	tmpl, err := template.New(MakefileBinVarsName).Parse(makefileBinVarsTmpl)
	if err != nil {
		return errors.Wrap(err, "parse makefile variables template")
	}

	makeFile, err = filepath.Abs(makeFile)
	if err != nil {
		return errors.Wrap(err, "abs")
	}
	relDir, err := filepath.Rel(filepath.Dir(makeFile), modDir)
	if err != nil {
		return err
	}

	data := struct {
		Version   string
		GobinPath string
		Binaries  []binary
	}{
		Version: version,
	}

ModLoop:
	for _, m := range modFiles {
		pkg, version, err := ModDirectPackage(m, nil)
		if err != nil {
			return err
		}
		name, _ := NameFromModFile(m)
		varName := strings.ReplaceAll(
			strings.ReplaceAll(
				strings.ToUpper(name),
				".", "_",
			),
			"-", "_",
		)
		for i, b := range data.Binaries {
			if b.Name == name {
				data.Binaries[i].VarName = varName + "_ARRAY"
				data.Binaries[i].Versions = append(data.Binaries[i].Versions, binaryVersion{
					BinName:    fmt.Sprintf("%s-%s", name, version),
					RelModFile: filepath.Join(relDir, filepath.Base(m)),
				})
				continue ModLoop
			}
		}
		data.Binaries = append(data.Binaries, binary{
			Name: name,
			Versions: []binaryVersion{
				{
					BinName:    fmt.Sprintf("%s-%s", name, version),
					RelModFile: filepath.Join(relDir, filepath.Base(m)),
				},
			},
			VarName:     varName,
			PackagePath: pkg,
		})
	}

	fb, err := os.Create(makefileBinVarsFile)
	if err != nil {
		return errors.Wrap(err, "create")
	}
	defer func() {
		if cerr := fb.Close(); cerr != nil {
			if err != nil {
				err = errors.Wrapf(err, "additionally error on close: %v", cerr)
				return
			}
			err = cerr
		}
	}()

	if err := tmpl.Execute(fb, data); err != nil {
		return errors.Wrap(err, "tmpl exec")
	}

	if makeFile == "" {
		return nil
	}
	// Optionally include this file in given Makefile.
	relMakefileBinVarsFile, err := filepath.Rel(filepath.Dir(makeFile), makefileBinVarsFile)
	if err != nil {
		return errors.Wrap(err, "getting relative path for makefileBinVarsFile")
	}
	b, err := ioutil.ReadFile(makeFile)
	if err != nil {
		return err
	}
	nodes, err := makefile.Parse(bytes.NewReader(b))
	if err != nil {
		return err
	}
	for _, n := range nodes {
		if inc, ok := n.(makefile.Include); ok && inc.Value == relMakefileBinVarsFile {
			// Nothing to do, include exists.
			return nil
		}
	}
	return ioutil.WriteFile(makeFile, append([]byte(fmt.Sprintf("include %s\n", relMakefileBinVarsFile)), b...), os.ModePerm)
}
