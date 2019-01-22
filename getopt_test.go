// Copyright 2019 Ian Johnson
//
// This file is part of getopt. Getopt is free software: you are free to use it
// for any purpose, make modified versions and share it with others, subject to
// the terms of the Apache license (version 2.0), a copy of which is provided
// alongside this project.

package getopt

import "testing"

func TestGetopt(t *testing.T) {
	type testOpt struct {
		option string
		arg    string
	}
	tests := []struct {
		input          []string
		opts           []testOpt
		additionalArgs []string
	}{
		{[]string{"-a"}, []testOpt{{"a", ""}}, []string{}},
		{[]string{"-aaa"}, []testOpt{{"a", ""}, {"a", ""}, {"a", ""}}, []string{}},
		{[]string{"-b", "25"}, []testOpt{{"bytes", "25"}}, []string{}},
		{[]string{"-b25"}, []testOpt{{"bytes", "25"}}, []string{}},
		{[]string{"--bytes", "25"}, []testOpt{{"bytes", "25"}}, []string{}},
		{[]string{"--bytes=25"}, []testOpt{{"bytes", "25"}}, []string{}},
		{[]string{"-cb25"}, []testOpt{{"c", "b25"}}, []string{}},
		{[]string{"-c", "5", "--bytes=7"}, []testOpt{{"c", "5"}, {"bytes", "7"}}, []string{}},
		{[]string{"--long", "--long"}, []testOpt{{"long", "--long"}}, []string{}},
		{[]string{"-c--long"}, []testOpt{{"c", "--long"}}, []string{}},
		{[]string{"-c", "--long"}, []testOpt{{"c", "--long"}}, []string{}},
		{[]string{"--flag", "--flag"}, []testOpt{{"flag", ""}, {"flag", ""}}, []string{}},
		{[]string{"-gg", "--go"}, []testOpt{{"go", ""}, {"go", ""}, {"go", ""}}, []string{}},
		{[]string{"-a", "arg"}, []testOpt{{"a", ""}}, []string{"arg"}},
		{[]string{"-bbytes", "2"}, []testOpt{{"bytes", "bytes"}}, []string{"2"}},
		{[]string{"-a", "--", "-a"}, []testOpt{{"a", ""}}, []string{"-a"}},
		{[]string{"--long", "--", "--", "--long"}, []testOpt{{"long", "--"}}, []string{"--long"}},
	}

	for _, test := range tests {
		p := new(Parser)
		p.Flag('a', "")
		p.Option('b', "bytes")
		p.Option('c', "")
		p.Option(0, "long")
		p.Flag(0, "flag")
		p.Flag('g', "go")
		p.ConsumeSlice(test.input)

		opt, arg, err := p.Getopt()
		i := 0
		for err == nil {
			newOpt := testOpt{opt, arg}
			if i >= len(test.opts) {
				t.Errorf("parsing %q: got unexpected option at position %v: %q", test.input, i, newOpt)
				goto nextTest
			}
			if newOpt != test.opts[i] {
				t.Errorf("parsing %q: at position %v: got %q, want %q", test.input, i, newOpt, test.opts[i])
				goto nextTest
			}

			opt, arg, err = p.Getopt()
			i++
		}
		if i != len(test.opts) {
			t.Errorf("parsing %q: got %v options, want %v", test.input, i, len(test.opts))
			goto nextTest
		}
		if err != End {
			t.Errorf("unexpected error with input %q: %q", test.input, err)
			goto nextTest
		}

		if len(p.Args()) != len(test.additionalArgs) {
			t.Errorf("parsing %q: got %v positional arguments, want %v", test.input, len(p.Args()), len(test.additionalArgs))
			goto nextTest
		}
		for i, arg := range p.Args() {
			if arg != test.additionalArgs[i] {
				t.Errorf("parsing %q: got positional argument %q at index %v, want %q", test.input, arg, i, test.additionalArgs[i])
				goto nextTest
			}
		}
	nextTest:
	}
}
