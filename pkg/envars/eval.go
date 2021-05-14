// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package envars

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/scanner"

	"github.com/pkg/errors"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// EvalVariables evaluates dot env file in similar way `bash source` would do and returns all environment variables available at end of the
// execution of the script.
// Currently it supports any bash script and can cause side effects.
// TODO(bwplotka): Walk over syntax and allow list few syntax elements only?
func EvalVariables(ctx context.Context, r io.Reader, envSlice ...string) (ret EnvSlice, _ error) {
	const prefix = "[[dotenv.EvalVariables]]:"

	s, err := syntax.NewParser().Parse(r, "")
	if err != nil {
		return nil, errors.Wrap(err, "parse")
	}

	vars := listVarNames(s)
	if len(vars) == 0 {
		return nil, nil
	}

	// sh does not implement the declaration clauses like `declare` or `export`. Let's set to ignore it and get variable
	// values manually.
	trimDeclStmts(s)

	// Add env print at the end to get all environment variables that would be available at this point!
	// Create env slice to print at the end of script.
	var parts []syntax.WordPart
	for _, v := range vars {
		parts = append(parts,
			&syntax.Lit{Value: fmt.Sprintf("%v \"", v)},
			&syntax.ParamExp{Param: &syntax.Lit{Value: v}},
			&syntax.Lit{Value: "\" "},
		)
	}

	s.Stmts = append(
		s.Stmts, &syntax.Stmt{Cmd: &syntax.CallExpr{
			Args: []*syntax.Word{
				{Parts: []syntax.WordPart{&syntax.Lit{Value: "echo"}}},
				{Parts: []syntax.WordPart{&syntax.DblQuoted{Parts: append([]syntax.WordPart{&syntax.Lit{Value: prefix}}, parts...)}}},
			}}},
	)

	b := bytes.Buffer{}
	ru, err := interp.New(interp.StdIO(os.Stdin, &b, &b), interp.Env(expand.ListEnviron(envSlice...)))
	if err != nil {
		return nil, err
	}

	if err := ru.Run(ctx, s); err != nil {
		return nil, err
	}

	var sc scanner.Scanner
	sc.Init(strings.NewReader(b.String()[strings.Index(b.String(), prefix)+len(prefix):]))
	tok := sc.Scan()
	for tok != scanner.EOF {
		k := sc.TokenText()
		_ = sc.Scan()
		v := sc.TokenText()
		ret = append(ret, fmt.Sprintf("%s=%s", k, strings.Trim(v, "\"")))
		tok = sc.Scan()
	}
	return ret, nil
}

func listVarNames(ast *syntax.File) (vars []string) {
	dup := map[string]struct{}{}
	for _, s := range ast.Stmts {
		syntax.Walk(s, func(node syntax.Node) bool {
			switch n := node.(type) {
			case *syntax.Assign:
				if n.Name == nil {
					return false
				}
				if _, ok := dup[n.Name.Value]; ok {
					return false
				}
				dup[n.Name.Value] = struct{}{}
				vars = append(vars, n.Name.Value)
				return false
			}
			return true
		})
	}
	return vars
}

func trimDeclStmts(ast *syntax.File) {
	for _, s := range ast.Stmts {
		syntax.Walk(s, func(node syntax.Node) bool {
			switch node.(type) {
			case *syntax.DeclClause:
				// TODO(bwplotka): Right not just trim them, but in future pull out assignments to statements on the parent level.
				node = nil // nolint
				return false
			}
			return true
		})
	}
}

// MergeEnvSlices merges two slices into single, sorted, deduplicated slice by applying `over` slice into `base`.
// The `over` slice will be used if the key overlaps.
// See https://golang.org/pkg/os/exec/#Cmd `Env` field to read more about slice format.
func MergeEnvSlices(base []string, over ...string) (merged []string) {
	sort.Strings(base)
	sort.Strings(over)

	var b, o int
	for b < len(base) || o < len(over) {

		if b >= len(base) {
			appendOrReplaceDup(&merged, over[o])
			o++
			continue
		}

		if o >= len(over) {
			appendOrReplaceDup(&merged, base[b])
			b++
			continue
		}

		switch strings.Compare(strings.Split(base[b], "=")[0], strings.Split(over[o], "=")[0]) {
		case 0:
			// Same keys. Instead of picking over element, ignore base one. This ensure correct behaviour if base
			// has duplicate elements.
			b++
		case 1:
			appendOrReplaceDup(&merged, over[o])
			o++
		case -1:
			appendOrReplaceDup(&merged, base[b])
			b++
		}
	}
	return merged
}

func appendOrReplaceDup(appendable *[]string, item string) {
	if len(*appendable) == 0 {
		*appendable = append(*appendable, item)
		return
	}

	lastI := len(*appendable) - 1
	if strings.Compare(strings.Split((*appendable)[lastI], "=")[0], strings.Split(item, "=")[0]) == 0 {
		(*appendable)[lastI] = item
		return
	}
	*appendable = append(*appendable, item)
}
