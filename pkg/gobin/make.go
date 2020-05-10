// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package gobin

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/bwplotka/gobin/pkg/makefile"
	"github.com/pkg/errors"
)

const (
	makefileBinVarsName = "Makefile.binary-variables"
	makefileBinVarsTmpl = `# bwplotka/gobin {{ .Version }} generated tools helper. Every time 'gobin get' is ran, this helper will regenerate
# if needed. Those generated variables ensure that every time a tool under each variable is invoked, the correct version
# will be used (reinstall will happen transparently if needed). See more details: https://{{ .GobinPath }}
GOBIN ?= $(firstword $(subst :, ,${GOPATH}))/bin

GOBIN_TOOL ?= $(GOBIN)/{{ .GobinBinName }}
$(GOBIN_TOOL): {{ .RelDir }}/{{ .GobinBinName }}.mod
	@# Install gobin allowing to install all pinned binaries.
	@GOBIN=$(GOBIN) go get -modfile={{ .RelDir }}/{{ .GobinBinName }}.mod {{ .GobinPath }}
{{ .GobinBinName }}.mod: ;

{{- range .Binaries }}

{{ .VarName }} ?= $(GOBIN)/{{ .BinName }}
$({{ .VarName }}): {{ $.RelDir}}/{{ .BinName }}.mod
{{ $.RelDir }}/{{ .BinName }}.mod: $(GOBIN_TOOL)
	@# Install binary using separate go module with pinned dependency.
	@$(GOBIN_TOOL) get {{ .BinName }}
{{ .BinName }}.mod: ;
{{- end}}
`
)

// GenMakeHelper generates helper Makefile variables to allows reliable binaries use. Regenerate if needed.
// It is expected to have at least one mod file.
func GenMakeHelperAndHook(makeFile, version, gobinInstallPath, gobinBinName string, modFiles ...string) error {
	modDir := filepath.Dir(modFiles[0])
	tmpl, err := template.New(makefileBinVarsName).Parse(makefileBinVarsTmpl)
	if err != nil {
		return errors.Wrap(err, "parse makefile variables template")
	}

	relDir, err := filepath.Rel(filepath.Dir(makeFile), modDir)
	if err != nil {
		return errors.Wrap(err, "rel")
	}

	type binary struct {
		VarName string
		BinName string
	}
	data := struct {
		Version string

		RelDir       string
		GobinBinName string
		GobinPath    string
		Binaries     []binary
	}{
		Version:      version,
		GobinBinName: gobinBinName,
		GobinPath:    gobinInstallPath,
		RelDir:       relDir,
	}

	for _, m := range modFiles {
		binName := strings.TrimSuffix(filepath.Base(m), ".mod")
		if binName == gobinBinName {
			continue
		}

		data.Binaries = append(data.Binaries, binary{
			BinName: binName,
			VarName: strings.ReplaceAll(
				strings.ReplaceAll(
					strings.ToUpper(binName),
					".", "_",
				),
				"-", "_",
			),
		})
	}

	makefileBinVarsFile := filepath.Join(modDir, makefileBinVarsName)
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

	makefileBinVarsFile, err = filepath.Rel(filepath.Dir(makeFile), makefileBinVarsFile)
	if err != nil {
		return errors.Wrap(err, "ref")
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
		inc, ok := n.(makefile.Include)
		if !ok {
			continue
		}

		if inc.Value == makefileBinVarsFile {
			// Nothing to do, include exists.
			return nil
		}
	}
	return ioutil.WriteFile(makeFile, append([]byte(fmt.Sprintf("include %s\n", makefileBinVarsFile)), b...), os.ModePerm)
}
