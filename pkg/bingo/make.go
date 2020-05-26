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
	makefileBinVarsName = "Makefile.binary-variables"
	// TODO(bwplotka): We might want to play with better escaping to allow spaces in dir names.
	makefileBinVarsTmpl = `# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo {{ .Version }}. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
GOBIN ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO    ?= $(which go)

# Bellow generated variables ensure that every time a tool under each variable is invoked, the correct version
# will be used; reinstalling only if needed.
# For example for {{ with (index .Binaries 0) }}{{ .BinName }}{{ end }} variable:
#
# In your main Makefile:
#
#include .bingo/Makefile.binary-variables # (If not generated automatically by bingo).
#
#command: $({{ with (index .Binaries 0) }}{{ .VarName }}{{ end }})
#	@echo "Running {{ with (index .Binaries 0) }}{{ .BinName }}{{ end }}"
#	@$({{ with (index .Binaries 0) }}{{ .VarName }}{{ end }}) <flags/args..>
#
{{- range .Binaries }}

{{ .VarName }} ?= $(GOBIN)/{{ .BinName }}
$({{ .VarName }}): {{ $.RelDir}}/{{ .BinName }}.mod
{{ $.RelDir }}/{{ .BinName }}.mod:
	@# Install binary using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@$(GO) build -modfile={{ $.RelDir}}/{{ .BinName }}.mod -o=$({{ .BinName }}) "{{ .PackagePath }}"
{{ .BinName }}.mod: ;
{{- end}}
`
)

type binary struct {
	VarName     string
	BinName     string
	PackagePath string
}

// RemoveMakeHelper deletes Makefile.binary-variables from mod directory.
func RemoveMakeHelper(modDir string) error {
	// TODO(bwplotka): This will NOT remove include, detect this?
	return os.RemoveAll(filepath.Join(modDir, makefileBinVarsName))
}

// GenMakeHelper generates helper Makefile variables to allows reliable binaries use. Regenerate if needed.
// It is expected to have at least one mod file.
func GenMakeHelperAndHook(modDir, makeFile, version string, modFiles ...string) error {
	makefileBinVarsFile := filepath.Join(modDir, makefileBinVarsName)
	if len(modFiles) == 0 {
		return errors.New("no mod files")
	}

	tmpl, err := template.New(makefileBinVarsName).Parse(makefileBinVarsTmpl)
	if err != nil {
		return errors.Wrap(err, "parse makefile variables template")
	}

	relDir, err := filepath.Rel(filepath.Dir(makeFile), modDir)
	if err != nil {
		return err
	}

	data := struct {
		Version string

		RelDir    string
		GobinPath string
		Binaries  []binary
	}{
		Version: version,
		RelDir:  relDir,
	}
	for _, m := range modFiles {
		pkg, _, err := ModDirectPackage(m, nil)
		if err != nil {
			return err
		}

		binName := strings.TrimSuffix(filepath.Base(m), ".mod")
		data.Binaries = append(data.Binaries, binary{
			BinName: binName,
			VarName: strings.ReplaceAll(
				strings.ReplaceAll(
					strings.ToUpper(binName),
					".", "_",
				),
				"-", "_",
			),
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
