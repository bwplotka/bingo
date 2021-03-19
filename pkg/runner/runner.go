// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/bwplotka/bingo/pkg/version"
	"github.com/pkg/errors"
)

// Runner allows to run certain commands against module aware Go CLI.
type Runner struct {
	goCmd    string
	insecure bool

	verbose   bool
	goVersion *semver.Version

	logger *log.Logger
}

func parseGoVersion(goVersionOutput string) (*semver.Version, error) {
	el := strings.Fields(strings.TrimRight(goVersionOutput, "\n"))
	if len(el) < 2 {
		return nil, errors.Errorf("unexpected go version output; expected 'go version go<semver> ...; found %v", strings.TrimRight(goVersionOutput, "\n"))
	}
	goVersion, err := semver.NewVersion(strings.TrimPrefix(el[2], "go"))
	if err != nil {
		return nil, err
	}
	return goVersion, nil
}

func isSupportedVersion(v *semver.Version) error {
	if !v.LessThan(version.Go114) {
		return nil
	}
	return errors.Errorf("found unsupported go version: %v; requires go 1.14.x or higher", v.String())
}

// NewRunner checks Go version compatibility then returns Runner.
func NewRunner(ctx context.Context, logger *log.Logger, insecure bool, goCmd string) (*Runner, error) {
	output := &bytes.Buffer{}
	r := &Runner{
		goCmd:    goCmd,
		insecure: insecure,
		logger:   logger,
	}

	if err := r.execGo(ctx, output, "", "", "version"); err != nil {
		return nil, errors.Wrap(err, "exec go to detect the version")
	}

	goVersion, err := parseGoVersion(output.String())
	if err != nil {
		return nil, err
	}

	r.goVersion = goVersion
	return r, isSupportedVersion(r.goVersion)
}

func (r *Runner) GoVersion() *semver.Version {
	return r.goVersion
}

func (r *Runner) Verbose() {
	r.verbose = true
}

var cmdsSupportingModFileArg = map[string]struct{}{
	"init":    {},
	"get":     {},
	"install": {},
	"list":    {},
	"build":   {},
}

func (r *Runner) execGo(ctx context.Context, output io.Writer, cd string, modFile string, args ...string) error {
	if modFile != "" {
		for i, arg := range args {
			if _, ok := cmdsSupportingModFileArg[arg]; ok {
				if i == len(args)-1 {
					args = append(args, fmt.Sprintf("-modfile=%s", modFile))
					break
				}
				args = append(args[:i+1], append([]string{fmt.Sprintf("-modfile=%s", modFile)}, args[i+1:]...)...)
				break
			}
		}
	}
	return r.exec(ctx, output, cd, r.goCmd, args...)
}

func (r *Runner) exec(ctx context.Context, output io.Writer, cd string, command string, args ...string) error {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = filepath.Join(cmd.Dir, cd)
	// TODO(bwplotka): Might be surprising, let's return err when this env variable is altered.
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			if r.verbose {
				return errors.Errorf("error while running command '%s %s'; err: %v", command, strings.Join(args, " "), err)
			}
			return errors.New("exit 1")
		}
		return errors.Errorf("error while running command '%s %s'; err: %v", command, strings.Join(args, " "), err)
	}
	if r.verbose {
		r.logger.Printf("exec '%s %s'\n", command, strings.Join(args, " "))
	}
	return nil
}

type Runnable interface {
	GoVersion() *semver.Version
	List(update GetUpdatePolicy, args ...string) (string, error)
	GetD(update GetUpdatePolicy, packages ...string) (string, error)
	Build(pkg, out string) error
	GoEnv(args ...string) (string, error)
	ModDownload() error
}

type runnable struct {
	r *Runner

	ctx     context.Context
	modFile string
	dir     string
}

// ModInit runs `go mod init` against separate go modules files if any.
func (r *Runner) ModInit(ctx context.Context, cd, modFile, moduleName string) error {
	out := &bytes.Buffer{}
	if err := r.execGo(ctx, out, cd, modFile, append([]string{"mod", "init"}, moduleName)...); err != nil {
		return errors.Wrap(err, out.String())
	}
	return nil
}

// With returns runner that will be ran against give modFile (if any) and in given directory (if any).
func (r *Runner) With(ctx context.Context, modFile string, dir string) Runnable {
	ru := &runnable{
		r:       r,
		modFile: modFile,
		dir:     dir,
		ctx:     ctx,
	}
	return ru
}

type GetUpdatePolicy string

const (
	NoUpdatePolicy    = GetUpdatePolicy("")
	UpdatePolicy      = GetUpdatePolicy("-u")
	UpdatePatchPolicy = GetUpdatePolicy("-u=patch")
)

func (r *runnable) GoVersion() *semver.Version {
	return r.r.GoVersion()
}

// List runs `go list` against separate go modules files if any.
func (r *runnable) List(update GetUpdatePolicy, args ...string) (string, error) {
	a := []string{"list"}
	if update != NoUpdatePolicy {
		a = append(a, string(update))
	}
	out := &bytes.Buffer{}
	if err := r.r.execGo(r.ctx, out, r.dir, r.modFile, append(a, args...)...); err != nil {
		return "", errors.Wrap(err, out.String())
	}
	return strings.Trim(out.String(), "\n"), nil
}

// GoEnv runs `go env` with given args.
func (r *runnable) GoEnv(args ...string) (string, error) {
	out := &bytes.Buffer{}
	if err := r.r.execGo(r.ctx, out, r.dir, "", append([]string{"env"}, args...)...); err != nil {
		return "", errors.Wrap(err, out.String())
	}
	return strings.Trim(out.String(), "\n"), nil
}

// GetD runs 'go get -d' against separate go modules file with given arguments.
func (r *runnable) GetD(update GetUpdatePolicy, packages ...string) (string, error) {
	args := []string{"get", "-d"}
	if r.r.insecure {
		args = append(args, "-insecure")
	}
	if update != NoUpdatePolicy {
		args = append(args, string(update))
	}

	out := &bytes.Buffer{}
	if err := r.r.execGo(r.ctx, out, r.dir, r.modFile, append(args, packages...)...); err != nil {
		return "", errors.Wrap(err, out.String())
	}
	return strings.Trim(out.String(), "\n"), nil
}

// Build runs 'go build' against separate go modules file with given packages.
func (r *runnable) Build(pkg, out string) error {
	output := &bytes.Buffer{}
	if err := r.r.execGo(r.ctx, output, r.dir, r.modFile, append([]string{"build", "-o=" + out}, pkg)...); err != nil {
		return errors.Wrap(err, output.String())
	}

	trimmed := strings.TrimSpace(output.String())
	if r.r.verbose && trimmed != "" {
		r.r.logger.Println(trimmed)
	}
	return nil
}

// ModDownload runs 'go mod download' against separate go modules file with given arguments.
func (r *runnable) ModDownload() error {
	args := []string{"mod", "download"}
	if r.r.verbose {
		args = append(args, "-x")
	}
	args = append(args, fmt.Sprintf("-modfile=%s", r.modFile))

	out := &bytes.Buffer{}
	if err := r.r.execGo(r.ctx, out, r.dir, r.modFile, args...); err != nil {
		return errors.Wrap(err, out.String())
	}

	trimmed := strings.TrimSpace(out.String())
	if r.r.verbose && trimmed != "" {
		r.r.logger.Println(trimmed)
	}
	return nil
}
