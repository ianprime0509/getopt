// Copyright 2019 Ian Johnson
//
// This file is part of getopt. Getopt is free software: you are free to use it
// for any purpose, make modified versions and share it with others, subject to
// the terms of the Apache license (version 2.0), a copy of which is provided
// alongside this project.

package getopt

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// End is the special "sentinel" error returned by the Getopt method when there
// are no more options left to parse (only positional arguments).
var End = errors.New("end of options")

// A Parser holds the internal state of a command-line parser.
//
// A Parser does not rely on any external state, so multiple parsers can be
// used simultaneously without interfering with one another (even on the same
// input). However, an individual Parser is not safe for concurrent use; any
// access from multiple goroutines must be controlled by the user to avoid race
// conditions.
type Parser struct {
	reorder bool     // whether to reorder the input, like GNU getopt
	input   []string // the arguments left to parse
	opts    []option // the options that this parser understands
}

// An option describes a single command-line option.
type option struct {
	short  rune   // the short form of the option
	long   string // the long form of the option
	hasArg bool   // whether the option accepts a (required) argument
}

// Flag describes a flag (option with no argument) to be recognized by the
// parser. 0 and "" may be passed as the short and long names in order to
// disable one or the other, but not both.
//
// This method will panic if not given either a short or long name, or if one
// of the names conflicts with a name already in use.
func (p *Parser) Flag(short rune, long string) {
	p.addOpt(short, long, false)
}

// Option describes an option (with a required argument) to be recognized by
// the parser. 0 and "" may be passed as the short and long names in order to
// disable one or the other, but not both.
//
// This method will panic if not given either a short or long name, or if one
// of the names conflicts with a name already in use.
func (p *Parser) Option(short rune, long string) {
	p.addOpt(short, long, true)
}

// addOpt is the common base of behavior for the Flag and Option methods.
func (p *Parser) addOpt(short rune, long string, hasArg bool) {
	if short == 0 && long == "" {
		panic("short and long names are both blank")
	}
	for _, opt := range p.opts {
		if (short != 0 && short == opt.short) || (long != "" && long == opt.long) {
			panic("name conflicts with existing option")
		}
	}
	p.opts = append(p.opts, option{short, long, hasArg})
}

// ConsumeArgs adds the command-line arguments passed to the current program to
// the arguments to be parsed. This is equivalent to calling
// ConsumeSlice(os.Args[1:]).
func (p *Parser) ConsumeArgs() {
	p.ConsumeSlice(os.Args[1:])
}

// ConsumeSlice adds the given arguments to the internal list of arguments to
// be parsed. Most users will only need to call this method once, to provide
// the arguments to be parsed.
//
// Changes to args after this method is called will not affect the parser's
// internal state.
func (p *Parser) ConsumeSlice(args []string) {
	p.input = append(p.input, args...)
}

// ReorderInput specifies whether the parser should reorder its input while
// parsing, in an attempt to find options given after some positional arguments
// (like what GNU getopt does). This behavior is disabled by default.
//
// This reordering does not affect the relative positions of arguments, except
// that options will be seen before positional arguments. The special option
// "--" can still be used to force the parser to treat everything afterwards as
// a positional argument.
func (p *Parser) ReorderInput(b bool) {
	p.reorder = b
}

// Args returns all the arguments remaining to be parsed. Any changes to the
// data in the returned slice will be reflected in any future parser
// operations.
//
// This method is most commonly used to get any positional arguments, once
// Getopt has signaled that there are no options left to parse.
func (p *Parser) Args() []string {
	return p.input
}

// Getopt returns the next option available to the parser. The name will be the
// long option name if one is available, or the short option name (converted to
// a string) if not. Any argument for the option is returned as well; if the
// option does not accept an argument, this string will be empty.
//
// If there are no more options available to the parser, the error will be the
// special sentinel value getopt.End. In any case where the error is non-nil,
// the name and arg strings will both be empty.
func (p *Parser) Getopt() (name string, arg string, err error) {
	if len(p.input) == 0 {
		return "", "", End
	}

	idx := 0
	if !isOption(p.input[idx]) {
		if !p.reorder {
			return "", "", End
		}
		// Try to find an option.
		for idx < len(p.input) {
			if isOption(p.input[idx]) {
				break
			}
			idx++
		}
		if idx == len(p.input) {
			// No option found.
			return "", "", End
		}
	}
	opt := p.input[idx]

	if opt == "--" {
		// We need to get rid of this "option", as it's really only a
		// signal to the option parser that there are no more options.
		p.removeInput(idx, idx+1)
		return "", "", End
	}

	if opt[1] == '-' {
		end := strings.IndexByte(opt, '=')
		if end == -1 {
			end = len(opt)
		}
		long, err := p.findLong(opt[2:end])
		if err != nil {
			return "", "", err
		}

		if long.hasArg {
			if end != len(opt) {
				// Option of the form '--option=arg'
				p.removeInput(idx, idx+1)
				return long.long, opt[end+1:], nil
			} else if idx == len(p.input)-1 {
				return "", "", fmt.Errorf("expected argument to '--%v'", long.long)
			}
			arg := p.input[idx+1]
			p.removeInput(idx, idx+2)
			return long.long, arg, nil
		} else if end != len(opt) {
			return "", "", fmt.Errorf("unexpected argument to '--%v'", long.long)
		}
		p.removeInput(idx, idx+1)
		return long.long, "", nil
	}

	rs := []rune(opt)
	short, err := p.findShort(rs[1])
	if err != nil {
		return "", "", err
	}
	optName := short.long
	if optName == "" {
		optName = string(short.short)
	}

	if short.hasArg {
		if len(rs) > 2 {
			rs := []rune(opt)
			p.removeInput(idx, idx+1)
			return optName, string(rs[2:]), nil
		} else if idx == len(p.input)-1 {
			return "", "", fmt.Errorf("expected argument to '-%v'", short.short)
		}
		arg := p.input[idx+1]
		p.removeInput(idx, idx+2)
		return optName, arg, nil
	} else if len(rs) > 2 {
		// We need to replace this option with one that can be parsed
		// by another call later to find the rest of the options in the
		// same argument.
		p.input[idx] = "-" + string(rs[2:])
	} else {
		p.removeInput(idx, idx+1)
	}
	return optName, "", nil
}

// removeInput removes the input arguments from the given start index
// (inclusive) to the given end index (exclusive).
func (p *Parser) removeInput(start, end int) {
	p.input = append(p.input[:start], p.input[end:]...)
}

// findShort returns the short option corresponding to the given rune, or an
// error indicating that no such option exists.
func (p *Parser) findShort(r rune) (option, error) {
	for _, opt := range p.opts {
		if opt.short == r {
			return opt, nil
		}
	}
	return option{}, fmt.Errorf("unrecognized option: '-%v'", r)
}

// findLong returns the long option corresponding to the given string, or an
// error indicating that no such option exists.
func (p *Parser) findLong(s string) (option, error) {
	for _, opt := range p.opts {
		if opt.long == s {
			return opt, nil
		}
	}
	return option{}, fmt.Errorf("unrecognized option: '--%v'", s)
}

// isOption returns whether the given string is an option. The special "option"
// "--" is considered to be an option by this function.
func isOption(arg string) bool {
	return len(arg) > 1 && arg[0] == '-'
}
