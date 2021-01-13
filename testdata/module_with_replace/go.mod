module github.com/bwplotka/bingo/testdata/module_with_replace

go 1.15

require (
	github.com/efficientgo/tools/copyright v0.0.0-20210109155620-3d3e7cfcbe22
	github.com/oklog/run v1.1.0
)

replace github.com/efficientgo/tools/copyright => github.com/pkg/errors v0.9.1 // For testing purposes, don't judge (:
