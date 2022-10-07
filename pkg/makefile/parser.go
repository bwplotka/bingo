// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package makefile

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/efficientgo/core/errors"
)

// Copied from https://github.com/tj/mmake/blob/b15229aac1a8ea3f0875f064a0864f7250bd7850/parser & improved.

// Node interface.
type Node interface {
	Lines() []int
}

type node struct {
	lines []int
}

func (n node) Lines() []int {
	return n.lines
}

// Comment node.
type Comment struct {
	node

	Target  string
	Value   string
	Default bool
}

// Include node.
type Include struct {
	node

	Value string
}

// Parser is a quick-n-dirty Makefile "parser", not
// really, just comments and a few directives, but
// you'll forgive me.
type Parser struct {
	i          int
	lines      []string
	nodeBuf    node
	commentBuf []string
	target     string
	nodes      []Node
}

// Parse the given input reader.
// TODO(bwplotka): Streaming version would be nice.
func (p *Parser) Parse(r io.Reader) ([]Node, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "reading")
	}

	p.lines = strings.Split(string(b), "\n")

	if err := p.parse(); err != nil {
		return nil, errors.Wrap(err, "parsing")
	}

	return p.nodes, nil
}

// Peek at the next line.
func (p *Parser) peek() string {
	return p.lines[p.i]
}

// Advance the next line.
func (p *Parser) advance() string {
	s := p.lines[p.i]
	p.i++
	return s
}

// Buffer comment.
func (p *Parser) bufferComment() {
	s := p.advance()[1:]

	if len(s) > 0 {
		if s[0] == '-' {
			return
		}

		// leading space
		if s[0] == ' ' {
			s = s[1:]
		}
	}
	p.commentBuf = append(p.commentBuf, s)
}

// Push comment node.
func (p *Parser) pushComment() {
	if len(p.commentBuf) == 0 {
		return
	}

	s := strings.Join(p.commentBuf, "\n")

	p.nodes = append(p.nodes, Comment{
		node:   p.nodeBuf,
		Target: p.target,
		Value:  strings.Trim(s, "\n"),
	})
	p.nodeBuf = node{}
	p.commentBuf = nil
	p.target = ""
}

// Push include node.
func (p *Parser) pushInclude() {
	s := strings.Trim(strings.Replace(p.advance(), "include ", "", 1), " ")
	p.nodeBuf.lines = append(p.nodeBuf.lines, p.i-1)
	p.nodes = append(p.nodes, Include{
		node:  p.nodeBuf,
		Value: s,
	})
	p.nodeBuf = node{}
}

// Parse the input.
func (p *Parser) parse() error {
	for {
		switch {
		case p.i == len(p.lines)-1:
			return nil
		case strings.HasPrefix(p.peek(), ".PHONY"):
			p.advance()
		case len(p.peek()) == 0:
			p.pushComment()
			p.advance()
		case p.peek()[0] == '#':
			p.bufferComment()
		case strings.HasPrefix(p.peek(), "include "):
			p.pushInclude()
		case strings.ContainsRune(p.peek(), ':'):
			p.target = strings.Split(p.advance(), ":")[0]
			p.nodeBuf.lines = append(p.nodeBuf.lines, p.i-1)
			p.pushComment()
		default:
			p.advance()
		}
	}
}

// Parse the given input.
func Parse(r io.Reader) ([]Node, error) {
	return (&Parser{}).Parse(r)
}

// ParseRecursive parses the given input recursively
// relative to the given dir such as /usr/local/include.
func ParseRecursive(r io.Reader, dir string) ([]Node, error) {
	nodes, err := parseRecursiveHelper(r, dir)

	for i := range nodes {
		defaultComment, ok := nodes[i].(Comment)
		if !ok {
			continue
		}

		defaultComment.Default = true
		nodes[i] = Comment{
			Target:  defaultComment.Target,
			Value:   defaultComment.Value,
			Default: true,
		}
		break
	}

	return nodes, err
}

func parseRecursiveHelper(r io.Reader, dir string) ([]Node, error) {
	nodes, err := Parse(r)

	if err != nil {
		return nil, errors.Wrap(err, "parsing")
	}

	otherNodes := []Node{}
	for _, n := range nodes {
		otherNodes = append(otherNodes, n)

		inc, ok := n.(Include)

		if !ok {
			continue
		}

		path := filepath.Join(dir, inc.Value)
		f, err := os.Open(path)
		if err != nil {
			return nil, errors.Wrapf(err, "opening %q", path)
		}

		more, err := parseRecursiveHelper(f, dir)

		if err != nil {
			return nil, errors.Wrapf(err, "parsing %q", path)
		}

		otherNodes = append(otherNodes, more...)

		f.Close()
	}

	return otherNodes, nil
}
