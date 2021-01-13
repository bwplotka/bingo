// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package gomodcache

import (
	"os"
	"os/exec"

	"github.com/bwplotka/bingo/pkg/envars"
	"github.com/efficientgo/tools/pkg/merrors"
	"github.com/pkg/errors"
)

const URL = "http://localhost:3000"

// TODO(bwplotka): Adding this code only slows down get_e2e_test.go
// Investigate.
// See 8d757f35b63be8ae76aa0bd3c3c5c4481114055b for details.
func Start(athensBin string, cacheDir string) (func() error, error) {
	env := envars.EnvSlice(os.Environ())

	// Format: https://raw.githubusercontent.com/gomods/athens/main/config.dev.toml.
	env.Set(
		"ATHENS_LOG_LEVEL="+"error",
		"ATHENS_GO_BINARY_ENV_VARS=\"[GOPROXY=https://proxy.golang.org]\"",
	)

	if cacheDir != "" {
		if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
			return nil, err
		}
		env.Set(
			"ATHENS_DISK_STORAGE_ROOT="+cacheDir,
			"ATHENS_STORAGE_TYPE="+"disk",
		)
	}

	c := exec.Command(athensBin)
	c.Env = env
	c.Stdout = os.Stdout
	c.Stderr = os.Stdout

	errc := make(chan error)
	go func() {
		errc <- c.Run()
		close(errc)
	}()

	return func() error {
		select {
		case err := <-errc:
			return errors.Errorf("proxy command unexpectedly went down; err: %v", err)
		default:
		}

		merr := merrors.New()
		if c.Process != nil {
			merr.Add(c.Process.Kill())
		}

		<-errc
		return merr.Err()
	}, nil
}
