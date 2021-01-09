// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package bingo

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/efficientgo/tools/core/pkg/errcapture"
	"github.com/pkg/errors"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

const (
	// FakeRootModFileName is a name for fake go module that we have to maintain, until https://github.com/bwplotka/bingo/issues/20 is fixed.
	FakeRootModFileName = "go.mod"

	NoReplaceCommand = "bingo:no_replace_fetch"
)

// NameFromModFile returns binary name from module file path.
func NameFromModFile(modFile string) (name string, oneOfMany bool) {
	n := strings.Split(strings.TrimSuffix(filepath.Base(modFile), ".mod"), ".")
	if len(n) > 1 {
		oneOfMany = true
	}
	return n[0], oneOfMany
}

type ModFile struct {
	fn string

	f *os.File
	m *modfile.File

	directPackage       *module.Version
	directModule        *module.Version
	autoReplaceDisabled bool
}

// OpenModFile opens bingo mod file and adds meta if missing.
func OpenModFile(modFile string) (_ *ModFile, err error) {
	f, err := os.OpenFile(modFile, os.O_RDWR, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			errcapture.Close(&err, f.Close, "close")
		}
	}()
	mf := &ModFile{f: f, fn: modFile}
	if err := mf.Reload(); err != nil {
		return nil, err
	}

	if err := onModHeaderComments(mf.m, func(comments *modfile.Comments) error {
		if err := errOnMetaMissing(comments); err != nil {
			mf.m.Module.Syntax.Suffix = append(mf.m.Module.Syntax.Suffix, modfile.Comment{Suffix: true, Token: metaComment})
			return mf.Flush()
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return mf, nil
}

func (mf *ModFile) Name() string {
	return mf.fn
}

func (mf *ModFile) AutoReplaceDisabled() bool {
	return mf.autoReplaceDisabled
}

func (mf *ModFile) Close() error {
	return mf.f.Close()
}

func (mf *ModFile) Reload() (err error) {
	if _, err := mf.f.Seek(0, 0); err != nil {
		return errors.Wrap(err, "seek")
	}

	mf.m, err = ParseModFileOrReader(mf.fn, mf.f)
	if err != nil {
		return err
	}

	mf.autoReplaceDisabled = false
	for _, c := range mf.m.Syntax.Comment().Suffix {
		if c.Token == NoReplaceCommand {
			mf.autoReplaceDisabled = true
			break
		}
	}

	// We expect just one direct import if any.
	mf.directPackage = nil
	mf.directModule = nil
	for _, r := range mf.m.Require {
		if r.Indirect {
			continue
		}

		pkg := r.Mod.Path
		if len(r.Syntax.Suffix) > 0 {
			pkg = path.Join(pkg, strings.Trim(r.Syntax.Suffix[0].Token[3:], "\n"))
		}
		mf.directModule = &module.Version{Path: r.Mod.Path, Version: r.Mod.Version}
		mf.directPackage = &module.Version{Path: pkg, Version: r.Mod.Version}
		break
	}
	return nil
}

// Flush saves all changes made to parsed syntax and reloads the parsed file.
func (mf *ModFile) Flush() error {
	newB := modfile.Format(mf.m.Syntax)
	if err := mf.f.Truncate(0); err != nil {
		return errors.Wrap(err, "truncate")
	}
	if _, err := mf.f.Seek(0, 0); err != nil {
		return errors.Wrap(err, "seek")
	}
	if _, err := mf.f.Write(newB); err != nil {
		return errors.Wrap(err, "write")
	}
	return mf.Reload()
}

// UpdateDirectPackage updates direct required module with the sub package path comment that recorded for package-level versioning.
// It's caller responsibility to Flush all changes.
func (mf *ModFile) UpdateDirectPackage(pkg string) (err error) {
	for _, r := range mf.m.Require {
		if !strings.HasPrefix(pkg, r.Mod.Path) {
			continue
		}

		r.Syntax.Suffix = r.Syntax.Suffix[:0]

		// Add sub package info if needed.
		if r.Mod.Path != pkg {
			subPkg, err := filepath.Rel(r.Mod.Path, pkg)
			if err != nil {
				return err
			}
			r.Syntax.Suffix = append(r.Syntax.Suffix, modfile.Comment{Suffix: true, Token: "// " + subPkg})
		}
		return nil

	}
	return errors.Errorf("empty or malformed module found in %s; expected require statement based on %v", mf.fn, pkg)
}

// SetReplace removes all replace statements and set to the given ones.
// It's caller responsibility to Flush all changes.
func (mf *ModFile) SetReplace(target ...*modfile.Replace) (err error) {
	for _, r := range mf.m.Replace {
		if err := mf.m.DropReplace(r.Old.Path, r.Old.Version); err != nil {
			return err
		}
	}
	for _, r := range target {
		if err := mf.m.AddReplace(r.Old.Path, r.Old.Version, r.New.Path, r.New.Version); err != nil {
			return err
		}
	}
	mf.m.Cleanup()
	return nil
}

// SetDirectRequire removes all require statements and set to the given ones.
// It's caller responsibility to Flush all changes.
func (mf *ModFile) SetDirectRequire(target ...*modfile.Require) (err error) {
	for _, r := range mf.m.Require {
		if err := mf.m.DropRequire(r.Mod.Path); err != nil {
			return err
		}
	}
	for _, r := range target {
		mf.m.AddNewRequire(r.Mod.Path, r.Mod.Version, false)
	}
	mf.m.Cleanup()
	return nil
}

func ParseModFileOrReader(modFile string, r io.Reader) (*modfile.File, error) {
	b, err := readAllFileOrReader(modFile, r)
	if err != nil {
		return nil, errors.Wrap(err, "read")
	}

	m, err := modfile.Parse(modFile, b, nil)
	if err != nil {
		return nil, errors.Wrap(err, "parse")
	}
	return m, nil
}

func readAllFileOrReader(modFile string, r io.Reader) (b []byte, err error) {
	if r != nil {
		return ioutil.ReadAll(r)
	}
	return ioutil.ReadFile(modFile)
}

func (mf *ModFile) DirectModule() *module.Version {
	return mf.directModule
}

func (mf *ModFile) DirectPackage() *module.Version {
	return mf.directPackage
}

func ModDirectPackage(modFile string) (pkg string, version string, err error) {
	mf, err := OpenModFile(modFile)
	if err != nil {
		return "", "", nil
	}
	defer errcapture.Close(&err, mf.Close, "close")

	if mf.directPackage == nil {
		return "", "", errors.Errorf("empty module found in %s", mf.fn)
	}
	return mf.directPackage.Path, mf.directPackage.Version, nil

}

const metaComment = "// Auto generated by https://github.com/bwplotka/bingo. DO NOT EDIT"

func onModHeaderComments(m *modfile.File, f func(*modfile.Comments) error) error {
	if m.Module == nil {
		return errors.New("failed to parse; no module")
	}
	if m.Module.Syntax == nil {
		return errors.New("failed to parse; no module's syntax")
	}
	if m.Module.Syntax.Comment() == nil {
		return errors.Errorf("expected %q comment on top of module, found no comment", metaComment)
	}
	return f(m.Module.Syntax.Comment())
}

func errOnMetaMissing(comments *modfile.Comments) error {
	for _, c := range comments.Suffix {
		tr := strings.Trim(c.Token, "\n")
		if tr != metaComment {
			return errors.Errorf("expected %q comment on top of module, found %q", metaComment, tr)
		}
	}
	return nil
}

type MainPackageVersion struct {
	Version string
	ModFile string
}

type MainPackage struct {
	Name        string
	PackagePath string
	EnvVarName  string
	Versions    []MainPackageVersion
}

// ListPinnedMainPackages lists all bingo pinned binaries (Go main packages).
func ListPinnedMainPackages(logger *log.Logger, modDir string, remMalformed bool) (pkgs []MainPackage, _ error) {
	modFiles, err := filepath.Glob(filepath.Join(modDir, "*.mod"))
	if err != nil {
		return nil, err
	}
ModLoop:
	for _, f := range modFiles {
		if filepath.Base(f) == FakeRootModFileName {
			continue
		}

		pkg, ver, err := ModDirectPackage(f)
		if err != nil {
			if remMalformed {
				logger.Printf("found malformed module file %v, removing due to error: %v\n", f, err)
				if err := os.RemoveAll(strings.TrimSuffix(f, ".") + "*"); err != nil {
					return nil, err
				}
			}
			continue
		}

		name, _ := NameFromModFile(f)
		varName := strings.ReplaceAll(strings.ReplaceAll(strings.ToUpper(name), ".", "_"), "-", "_")
		for i, p := range pkgs {
			if p.Name == name {
				pkgs[i].EnvVarName = varName + "_ARRAY"
				// Preserve order. Unfortunately first array mod file has no number, so it's last.
				if filepath.Base(f) == p.Name+".mod" {
					pkgs[i].Versions = append([]MainPackageVersion{{
						Version: ver,
						ModFile: filepath.Base(f),
					}}, pkgs[i].Versions...)
					continue ModLoop
				}

				pkgs[i].Versions = append(pkgs[i].Versions, MainPackageVersion{
					Version: ver,
					ModFile: filepath.Base(f),
				})
				continue ModLoop
			}
		}
		pkgs = append(pkgs, MainPackage{
			Name: name,
			Versions: []MainPackageVersion{
				{Version: ver, ModFile: filepath.Base(f)},
			},
			EnvVarName:  varName,
			PackagePath: pkg,
		})
	}
	return pkgs, nil
}
