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
	"regexp"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/bwplotka/bingo/pkg/envars"
	"github.com/bwplotka/bingo/pkg/version"
	"github.com/efficientgo/core/errors"
)

// Runner allows to run certain commands against module aware Go CLI.
type Runner struct {
	goCmd    string
	insecure bool

	verbose   bool
	goVersion *semver.Version

	logger *log.Logger
}

var versionRegexp = regexp.MustCompile(`^go version.* go((?:[0-9]+)(?:\.[0-9]+)?(?:\.[0-9]+)?)`)

// parseGoVersion ignores pre-release identifiers immediately following the
// patch version since we don't expect goVersionOutput to be SemVer-compliant.
func parseGoVersion(goVersionOutput string) (*semver.Version, error) {
	goVersion := versionRegexp.FindStringSubmatch(goVersionOutput)
	if goVersion == nil {
		return nil, errors.Newf("unexpected go version output; expected 'go version go<semver> ...; found %v", strings.TrimRight(goVersionOutput, "\n"))
	}
	return semver.NewVersion(goVersion[1])
}

func isSupportedVersion(v *semver.Version) error {
	if !v.LessThan(version.Go124) {
		return nil
	}
	return errors.Newf("found unsupported go version: %v; requires go 1.24.x or higher", v.String())
}

// NewRunner checks Go version compatibility then returns Runner.
func NewRunner(ctx context.Context, logger *log.Logger, insecure bool, goCmd string) (*Runner, error) {
	output := &bytes.Buffer{}
	r := &Runner{
		goCmd:    goCmd,
		insecure: insecure,
		logger:   logger,
	}

	if err := r.execGo(ctx, output, nil, "", "", "version"); err != nil {
		return nil, errors.Wrap(err, "exec go to detect the version")
	}

	goVersion, err := parseGoVersion(output.String())
	if err != nil {
		return nil, errors.Wrap(err, "parse go version")
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

func (r *Runner) execGo(ctx context.Context, output io.Writer, e envars.EnvSlice, cd string, modFile string, args ...string) error {
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
	return r.exec(ctx, output, e, cd, r.goCmd, args...)
}

func (r *Runner) exec(ctx context.Context, output io.Writer, e envars.EnvSlice, cd string, command string, args ...string) error {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = filepath.Join(cmd.Dir, cd)
	// TODO(bwplotka): Might be surprising, let's return err when this env variable is altered.
	e = envars.MergeEnvSlices(os.Environ(), e...)
	e.Set("GO111MODULE=on")
	e.Set("GOWORK=off")
	cmd.Env = e
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			if r.verbose {
				return errors.Newf("error while running command '%s %s'; err: %v", command, strings.Join(args, " "), err)
			}
			return errors.New("exit 1")
		}
		return errors.Newf("error while running command '%s %s'; err: %v", command, strings.Join(args, " "), err)
	}
	if r.verbose {
		r.logger.Printf("exec '%s %s'\n", command, strings.Join(args, " "))
	}
	return nil
}

type Runnable interface {
	GoVersion() *semver.Version
	List(args ...string) (string, error)
	GetD(packages ...string) (string, error)
	Build(pkg, out string, args ...string) error
	GoEnv(args ...string) (string, error)
	ModDownload(args ...string) error
}

type runnable struct {
	r *Runner

	ctx          context.Context
	modFile      string
	dir          string
	extraEnvVars envars.EnvSlice
}

// ModInit runs `go mod init` against separate go modules files if any.
func (r *Runner) ModInit(ctx context.Context, cd, modFile, moduleName string) error {
	out := &bytes.Buffer{}
	if err := r.execGo(ctx, out, nil, cd, modFile, append([]string{"mod", "init"}, moduleName)...); err != nil {
		return errors.Wrap(err, out.String())
	}
	return nil
}

// With returns runner that will be ran against give modFile (if any), in given directory (if any), with given extraEnvVars on top of Environ.
func (r *Runner) With(ctx context.Context, modFile string, dir string, extraEnvVars envars.EnvSlice) Runnable {
	ru := &runnable{
		r:            r,
		modFile:      modFile,
		dir:          dir,
		extraEnvVars: extraEnvVars,
		ctx:          ctx,
	}
	return ru
}

func (r *runnable) GoVersion() *semver.Version {
	return r.r.GoVersion()
}

// List runs `go list` against separate go modules files if any.
func (r *runnable) List(args ...string) (string, error) {
	a := []string{"list"}
	out := &bytes.Buffer{}
	if err := r.r.execGo(r.ctx, out, r.extraEnvVars, r.dir, r.modFile, append(a, args...)...); err != nil {
		return "", errors.Wrap(err, out.String())
	}
	return strings.Trim(out.String(), "\n"), nil
}

// GoEnv runs `go env` with given args.
func (r *runnable) GoEnv(args ...string) (string, error) {
	out := &bytes.Buffer{}
	if err := r.r.execGo(r.ctx, out, r.extraEnvVars, r.dir, "", append([]string{"env"}, args...)...); err != nil {
		return "", errors.Wrap(err, out.String())
	}
	return strings.Trim(out.String(), "\n"), nil
}

// GetD runs 'go get -d' against separate go modules file with given arguments.
func (r *runnable) GetD(packages ...string) (string, error) {
	args := []string{"get", "-d"}
	if r.r.insecure {
		args = append(args, "-insecure")
	}

	out := &bytes.Buffer{}
	if err := r.r.execGo(r.ctx, out, r.extraEnvVars, r.dir, r.modFile, append(args, packages...)...); err != nil {
		return "", errors.Wrap(err, out.String())
	}
	return strings.Trim(out.String(), "\n"), nil
}

// Build runs 'go build' against separate go modules file with given packages.
func (r *runnable) Build(pkg, out string, args ...string) error {
	args = append([]string{"build", "-o=" + out}, args...)
	output := &bytes.Buffer{}
	if err := r.r.execGo(r.ctx, output, r.extraEnvVars, r.dir, r.modFile, append(args, pkg)...); err != nil {
		return errors.Wrap(err, output.String())
	}

	trimmed := strings.TrimSpace(output.String())
	if r.r.verbose && trimmed != "" {
		r.r.logger.Println(trimmed)
	}
	return nil
}

// ModDownload runs 'go mod download' against separate go modules file.
func (r *runnable) ModDownload(args ...string) error {
	a := []string{"mod", "download"}
	if r.r.verbose {
		a = append(a, "-x")
	}
	a = append(a, fmt.Sprintf("-modfile=%s", r.modFile))

	out := &bytes.Buffer{}
	if err := r.r.execGo(r.ctx, out, r.extraEnvVars, r.dir, r.modFile, append(a, args...)...); err != nil {
		return errors.Wrap(err, out.String())
	}

	trimmed := strings.TrimSpace(out.String())
	if r.r.verbose && trimmed != "" {
		r.r.logger.Println(trimmed)
	}
	return nil
}
