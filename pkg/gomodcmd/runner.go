package gomodcmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// Runner allows to run certain commands against module aware Go CLI.
type Runner struct {
	goCmd    string
	modFile  string
	insecure bool
}

// NewRunner checks Go version compatibility and initialize new go.mod in the modDir if not yet present, then returns Runner.
func NewRunner(ctx context.Context, insecure bool, modDir string, goCmd string) (*Runner, error) {
	r := &Runner{
		goCmd:    goCmd,
		modFile:  filepath.Join(modDir, "go.mod"),
		insecure: insecure,
	}

	ver, err := r.execGo(ctx, false, "version")
	if err != nil {
		return nil, errors.Wrap(err, "exec go to detect the version")
	}
	fmt.Println(ver) // TODO

	if err := os.MkdirAll(modDir, 0); err != nil {
		return nil, errors.Wrapf(err, "create moddir %s", modDir)
	}

	if _, err := os.Stat(r.modFile); err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Wrapf(err, "stat moddir %s", modDir)
		}
		currMod, err := r.execGo(ctx, false, "list", "-m")
		if err != nil {
			return nil, err
		}

		// TODO(bwplotka): Check if currMod is not gobin..

		if _, err := r.execGo(ctx, true, "mod", "init", r.modFileArg(), fmt.Sprintf("%s/gobin", currMod)); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (c *Runner) modFileArg() string { return fmt.Sprintf("-modfile=%s", c.modFile) }

func (c *Runner) execGo(ctx context.Context, verbose bool, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, c.goCmd, args...)
	var b bytes.Buffer
	cmd.Stdout = &b
	cmd.Stderr = &b

	if verbose {
		// All to stdout.
		cmd.Stdout = io.MultiWriter(cmd.Stdout, os.Stdout)
		cmd.Stderr = io.MultiWriter(cmd.Stdout, os.Stdout)
	}
	if err := cmd.Run(); err != nil {
		out := b.String()
		if verbose {
			out = ""
		}
		return "", errors.Errorf("error: %v; Command %s %s out: %s", err, c.goCmd, strings.Join(args, " "), out)
	}

	return b.String(), nil
}

type GetUpdatePolicy string

const (
	NoUpdatePolicy    = GetUpdatePolicy("")
	UpdatePolicy      = GetUpdatePolicy("-u")
	UpdatePatchPolicy = GetUpdatePolicy("-u=patch")
)

// GetD runs 'go get -d' against separate go modules file with given arguments.
func (c *Runner) GetD(ctx context.Context, update GetUpdatePolicy, packages ...string) error {
	args := []string{"get", "-d", c.modFileArg()}
	if c.insecure {
		args = append(args, "-insecure")
	}

	if update != NoUpdatePolicy {
		args = append(args, string(update))
	}
	_, err := c.execGo(ctx, false, append(args, packages...)...)
	return err
}

// Installs runs 'go install' against separate go modules file with given packages.
func (c *Runner) Install(ctx context.Context, packages ...string) error {
	args := []string{"install", c.modFileArg()}
	_, err := c.execGo(ctx, false, append(args, packages...)...)
	return err
}
