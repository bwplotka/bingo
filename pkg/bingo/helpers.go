// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package bingo

import (
	"os"
	"path/filepath"
	"text/template"

	"github.com/efficientgo/core/errors"
)

// RemoveHelpers deletes helpers from mod directory.
func RemoveHelpers(modDir string) error {
	for ext := range templatesByFileExt {
		v := "variables." + ext
		if ext == "mk" {
			// Exception: for backward compatibility.
			v = "Variables.mk"
		}
		if err := os.RemoveAll(filepath.Join(modDir, v)); err != nil {
			return err
		}
	}
	return nil
}

// GenHelpers generates helpers to allows reliable binaries use. Regenerate if needed.
// It is expected to have at least one mod file.
// TODO(bwplotka): Allow installing those optionally?
func GenHelpers(relModDir, version string, pkgs []PackageRenderable) error {
	for ext, tmpl := range templatesByFileExt {
		v := "variables." + ext
		if ext == "mk" {
			// Exception: for backward compatibility.
			v = "Variables.mk"
		}
		if err := genHelper(v, tmpl, relModDir, version, pkgs); err != nil {
			return errors.Wrap(err, v)
		}
	}
	return nil
}

type templateData struct {
	Version      string
	GobinPath    string
	MainPackages []PackageRenderable
	RelModDir    string
}

func genHelper(f, tmpl, relModDir, version string, pkgs []PackageRenderable) error {
	t, err := template.New(f).Parse(tmpl)
	if err != nil {
		return errors.Wrap(err, "parse template")
	}

	data := templateData{
		Version:      version,
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
