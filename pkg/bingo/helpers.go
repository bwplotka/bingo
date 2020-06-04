// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package bingo

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// RemoveHelpers deletes helpers from mod directory.
func RemoveHelpers(modDir string) error {
	if err := os.RemoveAll(filepath.Join(modDir, MakefileBinVarsName)); err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(modDir, EnvBinVarsName))
}

type binaryVersion struct {
	BinName string
	ModFile string
}

type binary struct {
	Name        string
	VarName     string
	PackagePath string
	Versions    []binaryVersion
}

type templateData struct {
	Version   string
	GobinPath string
	Binaries  []binary
	RelModDir string
}

// GenHelpers generates helpers to allows reliable binaries use. Regenerate if needed.
// It is expected to have at least one mod file.
func GenHelpers(relModDir, version string, modFiles ...string) error {
	// TODO(bwplotka): Print fmt.Sprintf("include %s\n", relMakefileBinVarsFile)
	if err := genHelper(MakefileBinVarsName, makefileBinVarsTmpl, relModDir, version, modFiles...); err != nil {
		return errors.Wrap(err, MakefileBinVarsName)
	}
	if err := genHelper(EnvBinVarsName, envBinVarsTmpl, relModDir, version, modFiles...); err != nil {
		return errors.Wrap(err, EnvBinVarsName)
	}
	return nil
}

func genHelper(f, tmpl, relModDir, version string, modFiles ...string) error {
	file := filepath.Join(relModDir, f)
	if len(modFiles) == 0 {
		return errors.New("no mod files")
	}

	t, err := template.New(f).Parse(tmpl)
	if err != nil {
		return errors.Wrap(err, "parse template")
	}

	data := templateData{
		Version:   version,
		RelModDir: relModDir,
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
					BinName: fmt.Sprintf("%s-%s", name, version),
					ModFile: filepath.Base(m),
				})
				continue ModLoop
			}
		}
		data.Binaries = append(data.Binaries, binary{
			Name: name,
			Versions: []binaryVersion{
				{
					BinName: fmt.Sprintf("%s-%s", name, version),
					ModFile: filepath.Base(m),
				},
			},
			VarName:     varName,
			PackagePath: pkg,
		})
	}

	fb, err := os.Create(file)
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
	return t.Execute(fb, data)
}
