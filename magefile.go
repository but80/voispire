// +build mage

package main

import (
	"fmt"
	"os"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/mattn/go-shellwords"
	"github.com/mattn/go-zglob"
)

func init() {
	os.Setenv("GOFLAGS", "-mod=vendor")
	os.Setenv("GO111MODULE", "on")
}

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
		if ok, _ := zglob.Match("vendor/**/*", file); ok {
			continue
		}
		if err := sh.RunV("goimports", "-w", file); err != nil {
			return err
		}
	}
	return nil
}

// Check coding style
func Lint() error {
	return sh.RunV("golangci-lint", "run")
}

// Run test
func Test() error {
	return sh.RunV("go", "test", "./...")
}

func exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

var dirStack = []string{}

func pushDir(dir string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := os.Chdir(dir); err != nil {
		return err
	}
	dirStack = append(dirStack, wd)
	return nil
}

func popDir() {
	n := len(dirStack)
	if n == 0 {
		return
	}
	_ = os.Chdir(dirStack[n-1])
	dirStack = dirStack[:n-1]
}

func isNewer(this, that string) bool {
	sThis, err := os.Stat(this)
	if err != nil {
		return false
	}
	sThat, err := os.Stat(that)
	if err != nil {
		return false
	}
	return 0 < sThis.Size() && sThis.ModTime().After(sThat.ModTime())
}

// サブモジュールのチェックアウト
func Submodules() error {
	if exists("cmodules/world/makefile") {
		return nil
	}
	if err := sh.RunV("git", "submodule", "init"); err != nil {
		return err
	}
	if err := sh.RunV("git", "submodule", "update"); err != nil {
		return err
	}
	return nil
}

func BuildWorld() error {
	mg.SerialDeps(Submodules)
	if isNewer("cmodules/world/build/libworld.a", "cmodules/world/libsent/include/sent/speech.h") {
		fmt.Println("libworld.a は最新です")
		return nil
	}
	fmt.Println("world をビルド中...")
	if err := pushDir("cmodules/world"); err != nil {
		return err
	}
	defer popDir()
	_ = sh.RunV("make")
	return nil
}

// Build program
func Build() error {
	mg.SerialDeps(BuildWorld)
	return runVWithArgs("go", "build", "./cmd/voispire")
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
