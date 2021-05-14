// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package envars

import (
	"strings"

	"mvdan.cc/sh/v3/expand"
)

type EnvSlice []string

func (e EnvSlice) Lookup(k string) (string, bool) {
	for _, ev := range e {
		sp := strings.SplitN(ev, "=", 2)
		if sp[0] == k {
			return sp[1], true
		}
	}
	return "", false
}

func (e *EnvSlice) Set(kvs ...string) {
	*e = MergeEnvSlices(*e, kvs...)
}

// Get retrieves a variable by its name. To check if the variable is
// set, use Variable.IsSet.
func (e *EnvSlice) Get(name string) expand.Variable {
	return expand.ListEnviron(*e...).Get(name)
}

// Each iterates over all the currently set variables, calling the
// supplied function on each variable. Iteration is stopped if the
// function returns false.
//
// The names used in the calls aren't required to be unique or sorted.
// If a variable name appears twice, the latest occurrence takes
// priority.
//
// Each is required to forward exported variables when executing
// programs.
func (e *EnvSlice) Each(f func(name string, vr expand.Variable) bool) {
	expand.ListEnviron(*e...).Each(f)
}
