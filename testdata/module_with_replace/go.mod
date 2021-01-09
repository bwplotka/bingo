module github.com/bwplotka/bingo/testdata/module_with_replace

go 1.15

require (
	golang.org/x/crypto/openpgp/errors v0.0.0-20201221181555-eec23a3978ad
	github.com/oklog/run v1.1.0
)

replace golang.org/x/crypto/openpgp/errors => github.com/pkg/errors v0.9.1 // For testing purposes, don't judge (:
