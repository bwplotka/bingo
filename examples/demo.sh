#!/usr/bin/env bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# Import https://github.com/bwplotka/demo-nav bash library.
TYPE_SPEED=40
IMMEDIATE_REVEAL=false
NUMS=false
PREFIX="➜  "
curl https://raw.githubusercontent.com/bwplotka/demo-nav/master/demo-nav.sh -o ${DIR}/demo-nav.sh
. "${DIR}/demo-nav.sh"

rm -rf ${DIR}/tmp-demo
mkdir ${DIR}/tmp-demo
cp -r ${DIR}/../testdata/go.* ${DIR}/tmp-demo
cp -r ${DIR}/../testdata/main.go ${DIR}/tmp-demo
cd ${DIR}/tmp-demo

clear

# `r` registers command to be invoked.
#
# First argument specifies what should be printed.
# Second argument specifies what will be actually executed.
#
# NOTE: Use `'` to quote strings inside command.
r "${RED}# 'bingo' Demo time!"
r "${YELLOW}# Let's start with simple dev project (e.g Go project):" "ls -l"
r "${GREEN}export GOBIN=\`pwd\`/.bin && export PATH=\${PATH}:\${GOBIN} ${YELLOW}# We are exporting GOBIN envvar that controls where the binaries will be built."
r "${YELLOW}# Let's install 'bingo' (Go 1.14+ required):"
r "${GREEN}go get -u github.com/bwplotka/bingo"
r "${YELLOW}# Let's say we want to have proper lint tool, just 'bingo get' it!"
r "${GREEN}bingo get github.com/golangci/golangci-lint/cmd/golangci-lint"
r "${GREEN}ls -l .bingo"
r "${YELLOW}# As we can see above, 'bingo' created .bingo directory which stores separate .mod file for each pinned binary."
r "${GREEN}cat .bingo/golangci-lint.mod"
r "${YELLOW}# In fact, this is just standard Go modules with special preserved comments. You can modify it in place as normal Go .mod file!"
r "${GREEN}ls -l \${GOBIN}"
r "${YELLOW}# We also can see the golangci-lint was installed. NOTE that 'bingo' produces immutable names intentionally."
r "${GREEN}bingo list ${YELLOW}# 'bingo' can also list us what tools are pinned."
r "${YELLOW}# Let's install exact commit of goimports, 'bingo' can do that too:"
r "${GREEN}bingo get golang.org/x/tools/cmd/goimports@688b3c5d9fa5ae5ca974e3c59a6557c26007e0e6"
r "${GREEN}ls -l .bingo"
r "${GREEN}ls -l \${GOBIN}"
r "${GREEN}bingo list"
r "${GREEN}bingo get -u goimports ${YELLOW}# This is how your upgrade the pinned tool."
r "${GREEN}bingo get golangci-lint@v1.23.7 ${YELLOW}# This is how your downgrade if you need to."
r "${GREEN}bingo list"
r "${YELLOW}# Let's remove all installed binaries now to simulate freshly cloned repository:"
r "${GREEN}rm -rf \${GOBIN}"
r "${YELLOW}# Now, to install ALL required tools you can either use single 'bingo' command:"
r "${GREEN}go get github.com/bwplotka/bingo && bingo get && ls -l \${GOBIN}"
r "${YELLOW}# ...or if you don't want to depend on bingo for read access of you repo, just Go build command (no 'bingo' required):"
r "${GREEN}rm -rf \${GOBIN}"
r "${GREEN}go build -modfile=.bingo/golangci-lint.mod -o=\${GOBIN}/golangci-lint-v1.23.7 github.com/golangci/golangci-lint/cmd/golangci-lint && ls -l \${GOBIN}"
r "${YELLOW}# Thanks! See https://github.com/bwplotka/bingo for details. Demo created with https://github.com/bwplotka/demo-nav."

# Last entry to run navigation mode.
navigate
