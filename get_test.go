// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/efficientgo/core/errors"
	"github.com/efficientgo/core/testutil"
)

func TestParseTarget(t *testing.T) {
	for _, tcase := range []struct {
		target string

		expectedName     string
		expectedPkgPath  string
		expectedVersions []string
		expectedErr      error
	}{
		{
			target:      "",
			expectedErr: errors.New("target is empty, this should be filtered earlier"),
		},
		{
			target:           "tool",
			expectedName:     "tool",
			expectedVersions: []string{""},
		},
		{
			target:       "github.com/bwplotka/bingo",
			expectedName: "bingo", expectedPkgPath: "github.com/bwplotka/bingo",
			expectedVersions: []string{""},
		},
		{
			target:       "sigs.k8s.io/kustomize/kustomize",
			expectedName: "kustomize", expectedPkgPath: "sigs.k8s.io/kustomize/kustomize",
			expectedVersions: []string{""},
		},
		{
			target:       "sigs.k8s.io/kustomize/kustomize/v3",
			expectedName: "kustomize", expectedPkgPath: "sigs.k8s.io/kustomize/kustomize/v3",
			expectedVersions: []string{""},
		},
		{
			target:       "github.com/bwplotka/bingo/v21314213532",
			expectedName: "bingo", expectedPkgPath: "github.com/bwplotka/bingo/v21314213532",
			expectedVersions: []string{""},
		},
		{
			target:       "tool@version1",
			expectedName: "tool", expectedVersions: []string{"version1"},
		},
		{
			target:       "tool@version1123,version3,version1241",
			expectedName: "tool", expectedVersions: []string{"version1123", "version3", "version1241"},
		},
		{
			target:       "tool@none",
			expectedName: "tool", expectedVersions: []string{"none"},
		},
		{
			target:      "tool@version1123,version13,version1123",
			expectedErr: errors.New("version duplicates are not allowed, got: [version1123 version13 version1123]"),
		},
		{
			target:      "tool@version1123,version13,none",
			expectedErr: errors.New("none is not allowed when there are more than one specified Version, got: [version1123 version13 none]"),
		},
		{
			target:       "github.com/bwplotka/bingo/v2@v0.2.5-rc.1214,bb92924b84d060515f8eb35f428a8fd816c1d938,version1241",
			expectedName: "bingo", expectedPkgPath: "github.com/bwplotka/bingo/v2", expectedVersions: []string{"v0.2.5-rc.1214", "bb92924b84d060515f8eb35f428a8fd816c1d938", "version1241"},
		},
	} {
		t.Run("", func(t *testing.T) {
			n, p, v, err := parseTarget(tcase.target)
			if tcase.expectedErr != nil {
				testutil.NotOk(t, err)
				testutil.Equals(t, tcase.expectedErr.Error(), err.Error())
				return
			}

			testutil.Ok(t, err)
			testutil.Equals(t, tcase.expectedName, n)
			testutil.Equals(t, tcase.expectedPkgPath, p)
			testutil.Equals(t, tcase.expectedVersions, v)
		})
	}

}

func TestInstallSymlink(t *testing.T) {
	dir := t.TempDir()
	oldName := "some-binary-1.2.3"
	newName := "some-binary"
	oldPath := filepath.Join(dir, oldName)
	newPath := filepath.Join(dir, newName)
	testData := "#!/bin/true"

	// simulate an existing symlink to test the removal logic
	f, err := os.Create(newPath)
	testutil.Ok(t, err)
	testutil.Ok(t, f.Close())

	// create a dummy target ...
	f, err = os.Create(oldPath)
	testutil.Ok(t, err)
	// ... and write some data for verification later
	_, err = f.Write([]byte(testData))
	testutil.Ok(t, err)
	testutil.Ok(t, f.Close())

	err = installSymlink(dir, oldName, newName)
	testutil.Ok(t, err)

	gotPath, err := os.Readlink(newPath)
	testutil.Ok(t, err)
	testutil.Equals(t, oldName, gotPath)

	gotData, err := os.ReadFile(newPath)
	testutil.Ok(t, err)

	// ensure the symlink leads to the desired target
	testutil.Equals(t, testData, string(gotData))
}
