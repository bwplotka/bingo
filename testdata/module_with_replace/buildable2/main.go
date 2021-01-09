package main

import (
	"log"

	errors "github.com/bwplotka/bingo"
	module "github.com/bwplotka/bingo/testdata/module_with_replace"

	"github.com/oklog/run"
)

func main() {
	log.SetFlags(0)

	g := run.Group{}
	if err := g.Run(); err != nil {
		log.Fatal(errors.Wrap(err, "run"))
	}
	log.Println("module_with_replace.buildable2", module.Version)
}
