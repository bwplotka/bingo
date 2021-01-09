package main

import (
	"log"

	"github.com/oklog/run"
	"github.com/pkg/errors"
)

func main() {
	g := run.Group{}
	if err := g.Run(); err != nil {
		log.Fatal(errors.Wrap(err, "run"))
	}
	log.Println("buildable2 ok run")
}
