// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package mod

import (
	"io"
	"os"

	"github.com/efficientgo/core/errcapture"
	"github.com/efficientgo/core/errors"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

// File represents .mod file for Go Module use.
type File struct {
	path string

	f *os.File
	m *modfile.File
}

// OpenFile opens mod file for edits in place.
// It's a caller responsibility to Close the file when not using anymore.
func OpenFile(modFile string) (_ *File, err error) {
	f, err := os.OpenFile(modFile, os.O_RDWR, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			errcapture.Do(&err, f.Close, "close")
		}
	}()

	mf := &File{f: f, path: modFile}
	return mf, mf.Reload()
}

// OpenFileForRead opens mod file for reads.
// It's a caller responsibility to Close the file when not using anymore.
func OpenFileForRead(modFile string) (_ FileForRead, err error) {
	f, err := os.OpenFile(modFile, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			errcapture.Do(&err, f.Close, "close")
		}
	}()

	mf := &File{f: f, path: modFile}
	return mf, mf.Reload()
}

type FileForRead interface {
	Reload() error
	Filepath() string

	Module() (path string, comment string)
	Comments() (comments []string)
	GoVersion() string
	RequireDirectives() []RequireDirective
	ReplaceDirectives() []ReplaceDirective
	ExcludeDirectives() []ExcludeDirective
	RetractDirectives() []RetractDirective

	Close() error
}

// Reload re-parses module file from the latest state on the disk.
func (mf *File) Reload() (err error) {
	if _, err := mf.f.Seek(0, 0); err != nil {
		return errors.Wrap(err, "seek")
	}

	mf.m, err = parseModFileOrReader(mf.path, mf.f)
	return err
}

func (mf *File) Filepath() string {
	return mf.path
}

// Close closes file.
// TODO(bwplotka): Ensure other methods will return error on use after Close.
func (mf *File) Close() error {
	return mf.f.Close()
}

func (mf *File) Module() (path string, comment string) {
	if mf.m.Module == nil {
		return "", ""
	}
	if len(mf.m.Module.Syntax.Comment().Suffix) > 0 {
		comment = mf.m.Module.Syntax.Comment().Suffix[0].Token[3:]
	}
	return mf.m.Module.Mod.Path, comment
}

func (mf *File) SetModule(path string, comment string) error {
	if err := mf.m.AddModuleStmt(path); err != nil {
		return err
	}

	mf.m.Module.Syntax.Suffix = append(mf.m.Module.Syntax.Suffix, modfile.Comment{Suffix: true, Token: "// " + comment})

	return mf.flush()
}

func (mf *File) Comments() (comments []string) {
	for _, e := range mf.m.Syntax.Stmt {
		for _, c := range e.Comment().Before {
			comments = append(comments, c.Token[3:])
		}
	}
	return comments
}

func (mf *File) AddComment(comment string) error {
	mf.m.AddComment("// " + comment)

	return mf.flush()
}

// GoVersion returns a semver string containing the value of of the go directive.
// For example, it will return "1.2.3" if the go.mod file contains the line "go 1.2.3".
// If no go directive is found, it returns "1.0" because:
// 1. "1.0" is a valid semver string, so it's always safe to parse this value using semver.MustParse().
// 2. The semantics of the absence of a go directive in a go.mod file means all versions of Go should be able to compile it.
func (mf *File) GoVersion() string {
	if mf.m.Go == nil {
		return "1.0"
	}
	return mf.m.Go.Version
}

func (mf *File) SetGoVersion(version string) error {
	if err := mf.m.AddGoStmt(version); err != nil {
		return err
	}

	return mf.flush()
}

// Flush saves all changes made to parsed syntax and reloads the parsed file.
func (mf *File) flush() error {
	mf.m.Cleanup()
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
	// Reload, so syntax gets rebuilt. It might change due to format.
	return mf.Reload()
}

type RequireDirective struct {
	Module   module.Version
	Indirect bool

	// ExtraSuffixComment represents comment (without '// ') after potential indirect comment
	// that can contain additional information.
	ExtraSuffixComment string
}

func (mf *File) RequireDirectives() []RequireDirective {
	ret := make([]RequireDirective, len(mf.m.Require))
	for i, r := range mf.m.Require {
		ret[i] = RequireDirective{
			Module:   r.Mod,
			Indirect: r.Indirect,
		}
		if len(r.Syntax.Suffix) > 0 {
			ret[i].ExtraSuffixComment = r.Syntax.Suffix[0].Token[3:]
			if r.Indirect {
				ret[i].ExtraSuffixComment = r.Syntax.Suffix[0].Token[11:]
			}
		}
	}
	return ret
}

// SetRequireDirectives removes all require statements and set to the given ones.
func (mf *File) SetRequireDirectives(directives ...RequireDirective) (err error) {
	for _, r := range mf.m.Require {
		_ = mf.m.DropRequire(r.Mod.Path)
	}
	mf.m.Require = mf.m.Require[:0]

	for i, d := range directives {
		mf.m.AddNewRequire(d.Module.Path, d.Module.Version, d.Indirect)

		if len(d.ExtraSuffixComment) > 0 {
			r := mf.m.Require[i]
			// TODO(bwplotka): How it works with indirect on ?
			r.Syntax.Suffix = append(r.Syntax.Suffix[:0], modfile.Comment{Suffix: true, Token: "// " + d.ExtraSuffixComment})
		}
	}
	return mf.flush()
}

type ReplaceDirective struct {
	Old module.Version
	New module.Version
}

func (mf *File) ReplaceDirectives() []ReplaceDirective {
	ret := make([]ReplaceDirective, len(mf.m.Replace))
	for i, r := range mf.m.Replace {
		ret[i] = ReplaceDirective{
			Old: r.Old,
			New: r.New,
		}
	}
	return ret
}

// SetReplaceDirectives removes all replace statements and set to the given ones.
func (mf *File) SetReplaceDirectives(directives ...ReplaceDirective) (err error) {
	for _, r := range mf.m.Replace {
		_ = mf.m.DropReplace(r.Old.Path, r.Old.Version)
	}
	mf.m.Replace = mf.m.Replace[:0]

	// TODO(bwplotka): Backup before malformation?
	for _, d := range directives {
		if err := mf.m.AddReplace(d.Old.Path, d.Old.Version, d.New.Path, d.New.Version); err != nil {
			return err
		}
	}
	return mf.flush()
}

type ExcludeDirective struct {
	Module module.Version
}

func (mf *File) ExcludeDirectives() []ExcludeDirective {
	ret := make([]ExcludeDirective, len(mf.m.Exclude))
	for i, r := range mf.m.Exclude {
		ret[i] = ExcludeDirective{
			Module: r.Mod,
		}
	}
	return ret
}

// SetExcludeDirectives removes all replace statements and set to the given ones.
func (mf *File) SetExcludeDirectives(directives ...ExcludeDirective) (err error) {
	for _, r := range mf.m.Exclude {
		_ = mf.m.DropExclude(r.Mod.Path, r.Mod.Version)
	}
	mf.m.Exclude = mf.m.Exclude[:0]

	// TODO(bwplotka): Backup before malformation?
	for _, d := range directives {
		if err := mf.m.AddExclude(d.Module.Path, d.Module.Version); err != nil {
			return err
		}
	}
	return mf.flush()
}

type VersionInterval = modfile.VersionInterval

type RetractDirective struct {
	VersionInterval
	Rationale string
}

func (mf *File) RetractDirectives() []RetractDirective {
	ret := make([]RetractDirective, len(mf.m.Retract))
	for i, r := range mf.m.Retract {
		ret[i] = RetractDirective{
			VersionInterval: r.VersionInterval,
			Rationale:       r.Rationale,
		}
	}
	return ret
}

// SetRetractDirectives removes all replace statements and set to the given ones.
func (mf *File) SetRetractDirectives(directives ...RetractDirective) (err error) {
	for _, r := range mf.m.Retract {
		_ = mf.m.DropRetract(r.VersionInterval)
	}
	mf.m.Retract = mf.m.Retract[:0]

	for _, d := range directives {
		if err := mf.m.AddRetract(d.VersionInterval, d.Rationale); err != nil {
			return err
		}
	}
	return mf.flush()
}

// parseModFileOrReader parses any module file or reader allowing to read it's content.
func parseModFileOrReader(modFile string, r io.Reader) (*modfile.File, error) {
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
		return io.ReadAll(r)
	}
	return os.ReadFile(file)
}
