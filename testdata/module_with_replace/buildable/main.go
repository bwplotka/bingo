package main

import (
	"log"

	module "github.com/bwplotka/bingo/testdata/module_with_replace"
	errors "github.com/efficientgo/tools/core/pkg/runutil"

	"github.com/oklog/run"
)

func main() {
	log.SetFlags(0)

	g := run.Group{}
	if err := g.Run(); err != nil {
		log.Fatal(errors.Wrap(err, "run"))
	}
	log.Println("module_with_replace.buildable", module.Version)
}
