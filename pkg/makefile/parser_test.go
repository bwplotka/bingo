// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package makefile

import (
	"fmt"
	"strings"
)

func ExampleParser_Parse_withComments() {
	contents := `
include github.com/tj/foo
# Stuff here:
#
#    :)
#
# Start the dev server.
start:
	@gopherjs -m -v serve --http :3000 github.com/tj/docs/client
.PHONY: start
# Start the API server.
api:
	@go run server/cmd/api/api.go
.PHONY: api
# Display dependency graph.
deps:
	@godepgraph github.com/tj/docs/client | dot -Tsvg | browser
.PHONY: deps
# Display size of dependencies.
#
# - foo
# - bar
# - baz
#
size:
	@gopherjs build client/*.go -m -o /tmp/out.js
	@du -h /tmp/out.js
	@gopher-count /tmp/out.js | sort -nr
.PHONY: size
.PHONY: dummy
# Just a comment.
# Just another comment.
dummy:
	@ls
`

	nodes, err := Parse(strings.NewReader(contents))
	if err != nil {
		panic(err)
	}

	for _, node := range nodes {
		fmt.Printf("%#v\n", node)
	}

	// Output:
	// makefile.Include{node:makefile.node{lines:[]int{1}}, Value:"github.com/tj/foo"}
	// makefile.Comment{node:makefile.node{lines:[]int{7}}, Target:"start", Value:"Stuff here:\n\n   :)\n\nStart the dev server.", Default:false}
	// makefile.Comment{node:makefile.node{lines:[]int{8, 11}}, Target:"api", Value:"Start the API server.", Default:false}
	// makefile.Comment{node:makefile.node{lines:[]int{15}}, Target:"deps", Value:"Display dependency graph.", Default:false}
	// makefile.Comment{node:makefile.node{lines:[]int{24}}, Target:"size", Value:"Display size of dependencies.\n\n- foo\n- bar\n- baz", Default:false}
	// makefile.Comment{node:makefile.node{lines:[]int{32}}, Target:"dummy", Value:"Just a comment.\nJust another comment.", Default:false}
}

func ExampleParser_Parse_withoutComments() {
	contents := `
include github.com/tj/foo
include github.com/tj/bar
include github.com/tj/something/here
start:
	@gopherjs -m -v serve --http :3000 github.com/tj/docs/client
.PHONY: start
api:
	@go run server/cmd/api/api.go
.PHONY: api
deps:
	@godepgraph github.com/tj/docs/client | dot -Tsvg | browser
.PHONY: deps
`

	nodes, err := Parse(strings.NewReader(contents))
	if err != nil {
		panic(err)
	}

	for _, node := range nodes {
		fmt.Printf("%#v\n", node)
	}

	// Output:
	// makefile.Include{node:makefile.node{lines:[]int{1}}, Value:"github.com/tj/foo"}
	// makefile.Include{node:makefile.node{lines:[]int{2}}, Value:"github.com/tj/bar"}
	// makefile.Include{node:makefile.node{lines:[]int{3}}, Value:"github.com/tj/something/here"}
}
