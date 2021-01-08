package envars

import (
	"strings"
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
