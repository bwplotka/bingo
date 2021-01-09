module github.com/bwplotka/bingo/testdata/module_with_replace

go 1.15

require (
	github.com/bwplotka/bingo v0.2.5-0.20210109165007-c7f0d0510e70
	github.com/oklog/run v1.1.0
)

replace github.com/bwplotka/bingo => github.com/pkg/errors v0.9.1 // For testing purposes, don't judge (:
