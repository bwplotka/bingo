package main

import (
	"log"

	errors "golang.org/x/crypto/openpgp/errors"

	"github.com/oklog/run"
)

func not_main() {
	log.SetFlags(0)

	g := run.Group{}
	if err := g.Run(); err != nil {
		log.Fatal(errors.Wrap(err, "run"))
	}
	log.Println("it might look like buildable but it's not - no main package.")
}
