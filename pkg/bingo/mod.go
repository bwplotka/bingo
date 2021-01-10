// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package bingo

import (
	"io"
	"io/ioutil"
	"log"
	"os"
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

// A Package (for clients, a bingo.Package) is defined by a module path, package relative path and version pair.
// These are stored in their plain (unescaped) form.
type Package struct {
	Module module.Version

	// RelPath is a path that together with module compose a package path, like "/pkg/makefile".
	// Empty if the module is a full package path.
	RelPath string
}

// String returns a representation of the Package suitable for `go` tools and logging.
// (Module.Path/RelPath@Module.Version, or Module.Path/RelPath if Version is empty).
func (m Package) String() string {
	if m.Module.Version == "" {
		return m.Path()
	}
	return m.Path() + "@" + m.Module.Version
}

// Path returns a full package path.
func (m Package) Path() string {
	return filepath.Join(m.Module.Path, m.RelPath)
}

// ModFile represents bingo tool .mod file.
type ModFile struct {
	filename string

	f *os.File
	m *modfile.File

	directPackage       *Package
	autoReplaceDisabled bool
}

// OpenModFile opens bingo mod file.
// It also adds meta if missing and trims all require direct module imports except first within the parsed syntax.
// Use `Flush` to persist those changes.
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
	mf := &ModFile{f: f, filename: modFile}
	if err := mf.Reload(); err != nil {
		return nil, err
	}

	if err := onModHeaderComments(mf.m, func(comments *modfile.Comments) error {
		if err := errOnMetaMissing(comments); err != nil {
			mf.m.Module.Syntax.Suffix = append(mf.m.Module.Syntax.Suffix, modfile.Comment{Suffix: true, Token: metaComment})
			return nil
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return mf, nil
}

func (mf *ModFile) FileName() string {
	return mf.filename
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

	mf.m, err = ParseModFileOrReader(mf.filename, mf.f)
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
	for _, r := range mf.m.Require {
		if r.Indirect {
			continue
		}

		mf.directPackage = &Package{Module: r.Mod}
		if len(r.Syntax.Suffix) > 0 {
			mf.directPackage.RelPath = strings.Trim(r.Syntax.Suffix[0].Token[3:], "\n")
		}
		break
	}
	// Remove rest.
	mf.dropAllRequire()
	if mf.directPackage != nil {
		return mf.SetDirectRequire(*mf.directPackage)
	}
	return nil
}

func (mf *ModFile) DirectPackage() *Package {
	return mf.directPackage
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

// SetDirectRequire removes all require statements and set to the given one. It supports package level versioning.
// It's caller responsibility to Flush all changes.
func (mf *ModFile) SetDirectRequire(target Package) (err error) {
	mf.dropAllRequire()
	mf.m.AddNewRequire(target.Module.Path, target.Module.Version, false)

	// Add sub package info if needed.
	if target.RelPath != "" && target.RelPath != "." {
		r := mf.m.Require[0]
		r.Syntax.Suffix = append(r.Syntax.Suffix[:0], modfile.Comment{Suffix: true, Token: "// " + target.RelPath})
	}
	mf.m.Cleanup()
	return nil
}

func (mf *ModFile) dropAllRequire() {
	for _, r := range mf.m.Require {
		if r.Syntax == nil {
			continue
		}
		_ = mf.m.DropRequire(r.Mod.Path)
	}
	mf.m.Require = mf.m.Require[:0]
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

// ParseModFileOrReader parses any module file or reader allowing to read it's content.
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

func readAllFileOrReader(file string, r io.Reader) (b []byte, err error) {
	if r != nil {
		return ioutil.ReadAll(r)
	}
	return ioutil.ReadFile(file)
}

// ModDirectPackage return the first direct package from bingo enhanced module file. The package suffix (if any) is
// encoded in the line comment, in the same line as module and version.
func ModDirectPackage(modFile string) (pkg Package, err error) {
	mf, err := OpenModFile(modFile)
	if err != nil {
		return Package{}, err
	}
	defer errcapture.Close(&err, mf.Close, "close")

	if mf.directPackage == nil {
		return Package{}, errors.Errorf("no direct package found in %s; empty module?", mf.filename)
	}
	return *mf.directPackage, nil
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

// PackageVersionRenderable is used in variables.go. Modify with care.
type PackageVersionRenderable struct {
	Version string
	ModFile string
}

// PackageRenderable is used in variables.go. Modify with care.
type PackageRenderable struct {
	Name        string
	ModPath     string
	PackagePath string
	EnvVarName  string
	Versions    []PackageVersionRenderable
}

func (p PackageRenderable) ToPackages() []Package {
	ret := make([]Package, 0, len(p.Versions))
	for _, v := range p.Versions {
		relPath, _ := filepath.Rel(p.ModPath, p.PackagePath)

		ret = append(ret, Package{
			Module: module.Version{
				Version: v.Version,
				Path:    p.ModPath,
			},
			RelPath: relPath,
		})
	}
	return ret
}

// ListPinnedMainPackages lists all bingo pinned binaries (Go main packages).
func ListPinnedMainPackages(logger *log.Logger, modDir string, remMalformed bool) (pkgs []PackageRenderable, _ error) {
	modFiles, err := filepath.Glob(filepath.Join(modDir, "*.mod"))
	if err != nil {
		return nil, err
	}
ModLoop:
	for _, f := range modFiles {
		if filepath.Base(f) == FakeRootModFileName {
			continue
		}

		pkg, err := ModDirectPackage(f)
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
					pkgs[i].Versions = append([]PackageVersionRenderable{{
						Version: pkg.Module.Version,
						ModFile: filepath.Base(f),
					}}, pkgs[i].Versions...)
					continue ModLoop
				}

				pkgs[i].Versions = append(pkgs[i].Versions, PackageVersionRenderable{
					Version: pkg.Module.Version,
					ModFile: filepath.Base(f),
				})
				continue ModLoop
			}
		}
		pkgs = append(pkgs, PackageRenderable{
			Name: name,
			Versions: []PackageVersionRenderable{
				{Version: pkg.Module.Version, ModFile: filepath.Base(f)},
			},
			EnvVarName:  varName,
			PackagePath: pkg.Path(),
			ModPath:     pkg.Module.Path,
		})
	}
	return pkgs, nil
}
