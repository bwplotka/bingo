// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package main

import (
	"log"

	"github.com/bwplotka/bingo/testdata/module"
	"github.com/oklog/run"
	"github.com/pkg/errors"
)

func main() {
	log.SetFlags(0)

	g := run.Group{}
	if err := g.Run(); err != nil {
		log.Fatal(errors.Wrap(err, "run"))
	}
	log.Println("module.buildable", module.Version)
}
