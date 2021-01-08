package envars

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/efficientgo/tools/core/pkg/testutil"
)

func TestEval(t *testing.T) {
	t.Run("simple.env", func(t *testing.T) {
		b, err := ioutil.ReadFile("testdata/simple.env")
		testutil.Ok(t, err)

		e, err := EvalVariables(
			context.TODO(),
			bytes.NewReader(b),
			"PATH="+os.Getenv("PATH"), // go executable has to be available.
			"GOBIN=/home/something/bin",
			"HOME="+os.Getenv("HOME"),
		)
		testutil.Ok(t, err)
		testutil.Equals(t, []string{
			"VAR1=with space 124", "VAR2=with space 124-yolo", "VAR3=with\\n\\nnewline",
		}, e)
	})
	t.Run("bingo.env", func(t *testing.T) {
		b, err := ioutil.ReadFile("testdata/bingo.env")
		testutil.Ok(t, err)

		e, err := EvalVariables(
			context.TODO(),
			bytes.NewReader(b),
			"PATH="+os.Getenv("PATH"), // go executable has to be available.
			"GOBIN=/home/something/bin",
			"HOME="+os.Getenv("HOME"),
		)
		testutil.Ok(t, err)
		testutil.Equals(t, []string{
			"GOBIN=/home/something/bin",
			"COPYRIGHT=/home/something/bin/copyright-v0.0.0-20210107100701-44cf59f65a1b",
			"EMBEDMD=/home/something/bin/embedmd-v1.0.0",
			"FAILLINT=/home/something/bin/faillint-v1.5.0",
			"GOIMPORTS=/home/something/bin/goimports-v0.0.0-20200519204825-e64124511800",
			"GOLANGCI_LINT=/home/something/bin/golangci-lint-v1.26.0",
			"MDOX=/home/something/bin/mdox-v0.1.1-0.20201227133330-19093fdd9326",
			"MISSPELL=/home/something/bin/misspell-v0.3.4",
			"PROXY=/home/something/bin/proxy-v0.10.0",
		}, e)
	})
	t.Run("export.env", func(t *testing.T) {
		b, err := ioutil.ReadFile("testdata/export.env")
		testutil.Ok(t, err)

		e, err := EvalVariables(
			context.TODO(),
			bytes.NewReader(b),
			"PATH="+os.Getenv("PATH"), // go executable has to be available.
			"GOBIN=/home/something/bin",
			"HOME="+os.Getenv("HOME"),
		)
		testutil.Ok(t, err)
		// TODO(bwplotka): Support this assignments on decl statements.
		testutil.Equals(t, []string{
			"GOBIN=/home/something/bin",
		}, e)
	})

}

func TestMergeEnvSlices(t *testing.T) {
	t.Run("just base", func(t *testing.T) {
		testutil.Equals(t, []string{
			"OPTION_A=1",
			"OPTION_B=2",
			"OPTION_C=",
			"OPTION_D=\\n",
			"OPTION_E=1",
			"OPTION_F=2",
			"OPTION_G=",
			"OPTION_H=\n",
			"OPTION_I=echo 'asd'",
			"OPTION_J=postgres://localhost:5432/database?sslmode=disable",
		}, MergeEnvSlices([]string{
			"OPTION_A=1",
			"OPTION_B=2",
			"OPTION_C=",
			"OPTION_D=\\n",
			"OPTION_E=1",
			"OPTION_F=2",
			"OPTION_G=",
			"OPTION_H=\n",
			"OPTION_I=echo 'asd'",
			"OPTION_J=postgres://localhost:5432/database?sslmode=disable",
		}))
	})
	t.Run("just over", func(t *testing.T) {
		testutil.Equals(t, []string{
			"OPTION_A=1",
			"OPTION_B=2",
			"OPTION_C=",
			"OPTION_D=\\n",
			"OPTION_E=1",
			"OPTION_F=2",
			"OPTION_G=",
			"OPTION_H=\n",
			"OPTION_I=echo 'asd'",
			"OPTION_J=postgres://localhost:5432/database?sslmode=disable",
		}, MergeEnvSlices([]string{},
			"OPTION_A=1",
			"OPTION_B=2",
			"OPTION_C=",
			"OPTION_D=\\n",
			"OPTION_E=1",
			"OPTION_F=2",
			"OPTION_G=",
			"OPTION_H=\n",
			"OPTION_I=echo 'asd'",
			"OPTION_J=postgres://localhost:5432/database?sslmode=disable",
		))
	})
	t.Run("same", func(t *testing.T) {
		testutil.Equals(t, []string{
			"OPTION_A=1",
			"OPTION_B=2",
			"OPTION_C=",
			"OPTION_D=\\n",
			"OPTION_E=1",
			"OPTION_F=2",
			"OPTION_G=",
			"OPTION_H=\n",
			"OPTION_I=echo 'asd'",
			"OPTION_J=postgres://localhost:5432/database?sslmode=disable",
		}, MergeEnvSlices([]string{
			"OPTION_A=1",
			"OPTION_B=2",
			"OPTION_C=",
			"OPTION_D=\\n",
			"OPTION_E=1",
			"OPTION_F=2",
			"OPTION_G=",
			"OPTION_H=\n",
			"OPTION_I=echo 'asd'",
			"OPTION_J=postgres://localhost:5432/database?sslmode=disable",
		},
			"OPTION_A=1",
			"OPTION_B=2",
			"OPTION_C=",
			"OPTION_D=\\n",
			"OPTION_E=1",
			"OPTION_F=2",
			"OPTION_G=",
			"OPTION_H=\n",
			"OPTION_I=echo 'asd'",
			"OPTION_J=postgres://localhost:5432/database?sslmode=disable",
		))
	})
	t.Run("real with dups", func(t *testing.T) {
		testutil.Equals(t, []string{
			"OPTION_A=1",
			"OPTION_B=2",
			"OPTION_C=22",
			"OPTION_D=\\n22",
			"OPTION_E=122",
			"OPTION_F=222",
			"OPTION_G=22",
			"OPTION_H=\n22",
			"OPTION_I=echo 'asd'22",
			"OPTION_J=postgres://localhost:5432/database?sslmode=disable22",
		}, MergeEnvSlices([]string{
			"OPTION_A=1",
			"OPTION_B=2",
			"OPTION_C=",
			"OPTION_G=",
			"OPTION_H=\n",
			"OPTION_H=\n",
			"OPTION_H=\n",
			"OPTION_H=\n",
			"OPTION_J=postgres://localhost:5432/database?sslmode=disable",
		},
			"OPTION_C=22",
			"OPTION_D=\\n22",
			"OPTION_E=122",
			"OPTION_F=222",
			"OPTION_G=22",
			"OPTION_H=\n",
			"OPTION_H=\n",
			"OPTION_H=\n",
			"OPTION_H=\n22",
			"OPTION_I=echo 'asd'22",
			"OPTION_J=postgres://localhost:5432/database?sslmode=disable22",
		))
	})

}
