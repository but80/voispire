// +build mage

package main

import (
	"os"

	"github.com/magefile/mage/sh"
	"github.com/mattn/go-shellwords"
	"github.com/mattn/go-zglob"
)

func runVWithArgs(cmd string, args ...string) error {
	envArgs, err := shellwords.Parse(os.Getenv("ARGS"))
	if err != nil {
		return err
	}
	return sh.RunV(cmd, append(args, envArgs...)...)
}

// Format code
func Fmt() error {
	files, err := zglob.Glob("./**/*.go")
	if err != nil {
		return err
	}
	for _, file := range files {
		if err := sh.RunV("goimports", "-w", file); err != nil {
			return err
		}
	}
	return nil
}

// Check coding style
func Lint() error {
	return sh.RunV("gometalinter", "--config=.gometalinter.json", "./...")
}

// Run test
func Test() error {
	return sh.RunV("go", "test", "./...")
}

// Run program
func Run() error {
	return runVWithArgs("go", "run", "cmd/voispire/main.go")
}

// Run program with profiling
func Prof() error {
	return runVWithArgs("go", "run", "cmd/voispire/*.go")
}

// Run program with profiling without inlining optimization
func Prof2() error {
	return runVWithArgs("go", "run", "-gcflags", "-N -l", "cmd/voispire/*.go")
}