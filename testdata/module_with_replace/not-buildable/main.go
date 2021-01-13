// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package notmain

import (
	"log"

	errors "github.com/efficientgo/tools/copyright"

	"github.com/oklog/run"
)

func main() {
	log.SetFlags(0)

	g := run.Group{}
	if err := g.Run(); err != nil {
		log.Fatal(errors.Wrap(err, "run"))
	}
	log.Println("it might look like buildable but it's not - non main package.")
}
