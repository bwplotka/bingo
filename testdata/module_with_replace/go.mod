module github.com/bwplotka/bingo/testdata/module_with_replace

go 1.15

require (
	github.com/oklog/run v1.1.0
	github.com/efficientgo/tools/core/pkg/runutil v0.0.0-20210106193344-1108f4e7d16b
)

replace github.com/efficientgo/tools/core/pkg/runutil => github.com/pkg/errors v0.9.1 // For testing purposes, don't judge (:
