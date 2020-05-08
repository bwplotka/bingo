// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package gomodcmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// Runner allows to run certain commands against module aware Go CLI.
type Runner struct {
	goCmd    string
	insecure bool

	verbose bool
}

// NewRunner checks Go version compatibility then returns Runner.
func NewRunner(ctx context.Context, insecure bool, goCmd string) (*Runner, error) {
	r := &Runner{
		goCmd:    goCmd,
		insecure: insecure,
	}

	ver, err := r.execGo(ctx, "", "", "version")
	if err != nil {
		return nil, errors.Wrap(err, "exec go to detect the version")
	}

	// TODO(bwplotka): Make it more robust and accept newer Go.
	if !strings.HasPrefix(ver, "go version go1.14.") {
		return nil, errors.Errorf("found unsupported go version: %v. Requires go1.14.x", ver)
	}
	return r, nil
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

func (r *Runner) execGo(ctx context.Context, cd string, modFile string, args ...string) (string, error) {
	if modFile != "" {
		for i, arg := range args {
			if _, ok := cmdsSupportingModFileArg[arg]; ok {
				args = append(args[:i+1], append([]string{fmt.Sprintf("-modfile=%s", modFile)}, args[i+1:]...)...)
				break
			}

			if i == len(args)-1 {
				args = append(args, fmt.Sprintf("-modfile=%s", modFile))
			}
		}
	}
	return r.exec(ctx, cd, r.goCmd, args...)
}

func (r *Runner) exec(ctx context.Context, cd string, command string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = filepath.Join(cmd.Dir, cd)
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	var b bytes.Buffer
	cmd.Stdout = &b
	cmd.Stderr = &b

	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			if r.verbose {
				return "", errors.Errorf("error while running command '%s %s'; out: %s; err: %v", command, strings.Join(args, " "), b.String(), err)
			}
			return "", errors.New(b.String())

		}
		return "", errors.Errorf("error while running command '%s %s'; out: %s; err: %v", command, strings.Join(args, " "), b.String(), err)
	}
	if r.verbose {
		fmt.Printf("exec '%s %s'\n", command, strings.Join(args, " "))
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

type Runnable interface {
	ModInit(moduleName string) error
	List(args ...string) (string, error)
	GetD(update GetUpdatePolicy, packages ...string) error
	Build(pkg, out string) error
	ModTidy() error
}

type runnable struct {
	r *Runner

	ctx     context.Context
	modFile string
	dir     string
}

// With returns runnable that will be ran against give modFile (if any) and in given directory (if any).
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

// ModInit runs `go mod init` against separate go modules files if any.
func (r *runnable) ModInit(moduleName string) error {
	_, err := r.r.execGo(r.ctx, r.dir, r.modFile, append([]string{"mod", "init"}, moduleName)...)
	return err
}

// List runs `go list` against separate go modules files if any.
func (r *runnable) List(args ...string) (string, error) {
	return r.r.execGo(r.ctx, r.dir, r.modFile, append([]string{"list"}, args...)...)
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
	_, err := r.r.execGo(r.ctx, r.dir, r.modFile, append(args, packages...)...)
	return err
}

// Build runs 'go build' against separate go modules file with given packages.
func (r *runnable) Build(pkg, out string) error {
	binPath := os.Getenv("GOBIN")
	if gpath := os.Getenv("GOPATH"); gpath != "" && binPath == "" {
		binPath = filepath.Join(gpath, "bin")
	}
	out = filepath.Join(binPath, out)

	// go install does not define -o so we mimic go install with go build instead.
	_, err := r.r.execGo(
		r.ctx,
		r.dir,
		r.modFile,
		append(
			[]string{"build", "-i", "-o=" + out}, pkg,
		)...,
	)
	return err
}

// ModTidy runs 'go mod tidy' against separate go modules file.
func (r *runnable) ModTidy() error {
	_, err := r.r.execGo(r.ctx, r.dir, r.modFile, "mod", "tidy")
	return err
}
