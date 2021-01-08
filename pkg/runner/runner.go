// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Runner allows to run certain commands against module aware Go CLI.
type Runner struct {
	goCmd    string
	insecure bool

	verbose bool
}

var semVerRegexp = regexp.MustCompile(`^go version go([0-9]+)(\.[0-9]+)?(\.[0-9]+)?`)

func isSupportedVersion(foundVersion string) error {
	groups := semVerRegexp.FindAllStringSubmatch(foundVersion, -1)
	if len(groups) > 0 && len(groups[0]) >= 2 {
		major, err := strconv.ParseInt(groups[0][1], 10, 64)
		if err == nil && major >= 1 {
			foundVersion = fmt.Sprintf("v%v", strings.Join(groups[0][1:], ""))
			if major >= 2 {
				return nil
			}
			if len(groups[0]) >= 3 && len(groups[0][2]) > 1 {
				minor, err := strconv.ParseInt(groups[0][2][1:], 10, 64)
				if err == nil && minor >= 14 {
					return nil
				}
			}
		}
	}
	return errors.Errorf("found unsupported go version: %v; requires go1.14.x or higher", foundVersion)
}

// NewRunner checks Go version compatibility then returns Runner.
func NewRunner(ctx context.Context, insecure bool, goCmd string) (*Runner, error) {
	output := &bytes.Buffer{}
	r := &Runner{
		goCmd:    goCmd,
		insecure: insecure,
	}

	if err := r.execGo(ctx, output, "", "", "version"); err != nil {
		return nil, errors.Wrap(err, "exec go to detect the version")
	}
	return r, isSupportedVersion(strings.TrimRight(output.String(), "\n"))
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
		fmt.Printf("exec '%s %s'\n", command, strings.Join(args, " "))
	}
	return nil
}

type Runnable interface {
	ModInit(moduleName string) error
	List(args ...string) (string, error)
	GetD(update GetUpdatePolicy, packages ...string) error
	Build(pkg, out string) error
	GoEnv(args ...string) (string, error)
}

type runnable struct {
	r *Runner

	ctx     context.Context
	modFile string
	dir     string
	silent  bool
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

// WithSilent returns runner that will be ran against give modFile (if any) and in given directory (if any).
func (r *Runner) WithSilent(ctx context.Context, modFile string, dir string) Runnable {
	ru := r.With(ctx, modFile, dir)
	ru.(*runnable).silent = true
	return ru
}

type GetUpdatePolicy string

const (
	NoUpdatePolicy    = GetUpdatePolicy("")
	UpdatePolicy      = GetUpdatePolicy("-u")
	UpdatePatchPolicy = GetUpdatePolicy("-u=patch")
)

// ModInit runs `go mod init` against separate go modules files if any. REMOVE
func (r *runnable) ModInit(moduleName string) error {
	out := &bytes.Buffer{}
	if err := r.r.execGo(r.ctx, out, r.dir, r.modFile, append([]string{"mod", "init"}, moduleName)...); err != nil {
		return errors.Wrap(err, out.String())
	}
	return nil
}

// List runs `go list` against separate go modules files if any.
func (r *runnable) List(args ...string) (string, error) {
	out := &bytes.Buffer{}
	if err := r.r.execGo(r.ctx, out, r.dir, r.modFile, append([]string{"list"}, args...)...); err != nil {
		return "", errors.Wrap(err, out.String())
	}
	return strings.TrimRight(out.String(), "\n"), nil
}

// GoEnv runs `go env` with given args.
func (r *runnable) GoEnv(args ...string) (string, error) {
	out := &bytes.Buffer{}
	if err := r.r.execGo(r.ctx, out, r.dir, r.modFile, append([]string{"env"}, args...)...); err != nil {
		return "", errors.Wrap(err, out.String())
	}
	return strings.TrimRight(out.String(), "\n"), nil
}

// GetD runs 'go get -d' against separate go modules file with given arguments.
func (r *runnable) GetD(update GetUpdatePolicy, packages ...string) error {
	args := []string{"get", "-d"}
	if r.r.insecure {
		args = append(args, "-insecure")
	}
	if update != NoUpdatePolicy {
		args = append(args, string(update))
	}

	if !r.silent {
		return r.r.execGo(r.ctx, os.Stdout, r.dir, r.modFile, append(args, packages...)...)
	}

	out := &bytes.Buffer{}
	if err := r.r.execGo(r.ctx, out, r.dir, r.modFile, append(args, packages...)...); err != nil {
		return errors.Wrap(err, out.String())
	}
	return nil
}

// Build runs 'go build' against separate go modules file with given packages.
func (r *runnable) Build(pkg, outPath string) error {
	// go install does not define -o so we mimic go install with go build instead.
	binPath := os.Getenv("GOBIN")
	if gpath := os.Getenv("GOPATH"); gpath != "" && binPath == "" {
		binPath = filepath.Join(gpath, "bin")
	}
	outPath = filepath.Join(binPath, outPath)

	if !r.silent {
		return r.r.execGo(r.ctx, os.Stdout, r.dir, r.modFile, append([]string{"build", "-o=" + outPath}, pkg)...)
	}

	out := &bytes.Buffer{}
	if err := r.r.execGo(r.ctx, out, r.dir, r.modFile, append([]string{"build", "-o=" + outPath}, pkg)...); err != nil {
		return errors.Wrap(err, out.String())
	}
	return nil
}
