package main

import (
	"log"

	errors "github.com/efficientgo/tools/core"

	"github.com/oklog/run"
)

func main() {
	g := run.Group{}
	if err := g.Run(); err != nil {
		log.Fatal(errors.Wrap(err, "run"))
	}
	log.Println("buildable1 ok run")
}
