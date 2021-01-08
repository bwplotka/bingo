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

	stdout, stderr io.Writer
	output         *bytes.Buffer
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
		stdout:   output,
		stderr:   output,
		output:   output,
	}

	if err := r.execGo(ctx, "", "", "version"); err != nil {
		return nil, errors.Wrap(err, "exec go to detect the version")
	}
	return r, isSupportedVersion(strings.TrimRight(r.output.String(), "\n"))
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

func (r *Runner) execGo(ctx context.Context, cd string, modFile string, args ...string) error {
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
	return r.exec(ctx, cd, r.goCmd, args...)
}

func (r *Runner) exec(ctx context.Context, cd string, command string, args ...string) error {
	r.output.Truncate(0)

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = filepath.Join(cmd.Dir, cd)
	// TODO(bwplotka): Might be surpring, let's return err when this env variable is altered.
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	cmd.Stdout = r.stdout
	cmd.Stderr = r.stderr
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			if r.verbose {
				return errors.Errorf("error while running command '%s %s'; out: %s; err: %v", command, strings.Join(args, " "), r.output.String(), err)
			}
			return errors.New(r.output.String())

		}
		return errors.Errorf("error while running command '%s %s'; out: %s; err: %v", command, strings.Join(args, " "), r.output.String(), err)
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
}

// With returns runner that will be ran against give modFile (if any) and in given directory (if any).
func (r *Runner) With(ctx context.Context, modFile string, dir string) Runnable {
	ru := &runnable{
		r:       r,
		modFile: modFile,
		dir:     dir,
		ctx:     ctx,
	}
	ru.enableOSStdOutput(true)
	return ru
}

// WithDisabledOutput returns runner that will be ran against give modFile (if any) and in given directory (if any).
func (r *Runner) WithDisabledOutput(ctx context.Context, modFile string, dir string) Runnable {
	ru := r.With(ctx, modFile, dir)
	ru.(*runnable).enableOSStdOutput(false)
	return ru
}

type GetUpdatePolicy string

const (
	NoUpdatePolicy    = GetUpdatePolicy("")
	UpdatePolicy      = GetUpdatePolicy("-u")
	UpdatePatchPolicy = GetUpdatePolicy("-u=patch")
)

func (r *runnable) enableOSStdOutput(enable bool) {
	if enable {
		r.r.stdout = io.MultiWriter(r.r.output, os.Stdout)
		r.r.stderr = io.MultiWriter(r.r.output, os.Stderr)
		return
	}
	r.r.stdout = r.r.output
	r.r.stderr = r.r.output
}

// ModInit runs `go mod init` against separate go modules files if any. REMOVE
func (r *runnable) ModInit(moduleName string) error {
	r.enableOSStdOutput(false)
	defer r.enableOSStdOutput(true)

	return r.r.execGo(r.ctx, r.dir, r.modFile, append([]string{"mod", "init"}, moduleName)...)
}

// List runs `go list` against separate go modules files if any.
func (r *runnable) List(args ...string) (string, error) {
	r.enableOSStdOutput(false)
	defer r.enableOSStdOutput(true)

	err := r.r.execGo(r.ctx, r.dir, r.modFile, append([]string{"list"}, args...)...)
	return strings.TrimRight(r.r.output.String(), "\n"), err
}

// GoEnv runs `go env` with given args.
func (r *runnable) GoEnv(args ...string) (string, error) {
	r.enableOSStdOutput(false)
	defer r.enableOSStdOutput(true)

	err := r.r.execGo(r.ctx, r.dir, r.modFile, append([]string{"env"}, args...)...)
	return strings.TrimRight(r.r.output.String(), "\n"), err
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
	return r.r.execGo(r.ctx, r.dir, r.modFile, append(args, packages...)...)
}

// Build runs 'go build' against separate go modules file with given packages.
func (r *runnable) Build(pkg, outPath string) error {
	binPath := os.Getenv("GOBIN")
	if gpath := os.Getenv("GOPATH"); gpath != "" && binPath == "" {
		binPath = filepath.Join(gpath, "bin")
	}
	outPath = filepath.Join(binPath, outPath)

	// go install does not define -o so we mimic go install with go build instead.
	return r.r.execGo(r.ctx, r.dir, r.modFile, append([]string{"build", "-o=" + outPath}, pkg)...)
}
