// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package bingo

import (
	"os"
	"path/filepath"
	"text/template"

	"github.com/pkg/errors"
)

// RemoveHelpers deletes helpers from mod directory.
func RemoveHelpers(modDir string) error {
	if err := os.RemoveAll(filepath.Join(modDir, MakefileBinVarsName)); err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(modDir, EnvBinVarsName))
}

// GenHelpers generates helpers to allows reliable binaries use. Regenerate if needed.
// It is expected to have at least one mod file.
func GenHelpers(relModDir, version string, pkgs []MainPackage) error {
	if err := genHelper(MakefileBinVarsName, makefileBinVarsTmpl, relModDir, version, pkgs); err != nil {
		return errors.Wrap(err, MakefileBinVarsName)
	}
	if err := genHelper(EnvBinVarsName, envBinVarsTmpl, relModDir, version, pkgs); err != nil {
		return errors.Wrap(err, EnvBinVarsName)
	}
	return nil
}

type templateData struct {
	Version      string
	GobinPath    string
	MainPackages []MainPackage
	RelModDir    string
}

func genHelper(f, tmpl, relModDir, version string, pkgs []MainPackage) error {
	t, err := template.New(f).Parse(tmpl)
	if err != nil {
		return errors.Wrap(err, "parse template")
	}

	data := templateData{
		Version:      version,
		RelModDir:    relModDir,
		MainPackages: pkgs,
	}

	fb, err := os.Create(filepath.Join(relModDir, f))
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
