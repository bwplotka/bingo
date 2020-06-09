# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

NOTE: As semantic versioning states all 0.y.z releases can contain breaking changes in API (flags, grpc API, any backward compatibility)

We use *breaking* word for marking changes that are not backward compatible (relates only to v0.y.z releases.)

## [v0.2.1](https://github.com/bwplotka/bingo/releases/tag/v0.2.1) - 2020.06.04

### Fixed

* [#25](https://github.com/bwplotka/bingo/issues/25) Fixed support of `bingo get` for arrays.
* Fixed versioning binaries with `+incompatible` version (wrong templating used).
* Fixed support `bingo list` for arrays.
* Added rename / clone logic
* Always print to stdout no matter of verbose level.

### Changed

* `bingo list` output format. (table `\t`-delimited now)

## [v0.2.1](https://github.com/bwplotka/bingo/releases/tag/v0.2.1) - 2020.06.04

### Fixed

*  Fixed extra whitespace in variables.env.

## [v0.2.0](https://github.com/bwplotka/bingo/releases/tag/v0.2.0) - 2020.06.04

### Changed

* Added `.variables.env` file to bingo moddir for easy export of all environment variables to the current shell. Removed `-m` and `-makefile` flags.
Bingo now always creates makefile and env file and never generate `include` to avoid many corner cases. It's now documented how to add `include` in the documentation.

## [v0.1.1](https://github.com/bwplotka/bingo/releases/tag/v0.1.1) - 2020.06.03

### Fixed

* [#22](https://github.com/bwplotka/bingo/pull/22) Fixed problem with running bingo in non-Go project. From now on it also maintains
fake go.mod to resolve issues like:

```
`Error: get command failed: 0: getting : go get -d: go: cannot find main module, but -modfile was set.
	-modfile cannot be used to set the module root directory.
```

## [v0.1.0](https://github.com/bwplotka/bingo/releases/tag/v0.1.0) - 2020.05.30

Initial release.

Why 0.1.0? Well, because we plan to release 1.0 once we introduce this tool to [Thanos](http://github.com/thanos-io/thanos) and [go-grpc-middleware](https://github.com/grpc-ecosystem/go-grpc-middleware) as the final test (:
After having this usage stable for a bit, and we are sure flags will not change, we can claim 1.0.
