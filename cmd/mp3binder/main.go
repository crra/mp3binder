package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/crra/mp3binder/cli"
	"github.com/crra/mp3binder/mp3binder"
	"github.com/crra/mp3binder/mp3binder/tags"
	"github.com/go-logr/stdr"
	"github.com/spf13/afero"
)

var (
	// externally set by the build system.
	version = "dev-build"
	name    = "mp3builder"
	realm   = "mp3builder"
)

// main is the entrypoint of the program.
// main is the only place where external dependencies (e.g. output stream, logger, filesystem)
// are resolved and where final errors are handled (e.g. writing to the console).
func main() {
	// use the built in logger
	log := stdr.New(log.New(os.Stdout, "", log.Lshortfile))

	// create a parent context that listens on os signals (e.g. CTRL-C)
	context, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	// cancel the parent context and all children if an os signal arrives
	go func() {
		<-context.Done()
		cancel()
	}()

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fs := afero.NewOsFs()

	// run the program and clean up
	resolver := tags.NewV24(cli.ErrTagNonStandard)
	binder := mp3binder.New(resolver)

	if err := cli.New(context, name, version, log, os.Stdout, fs, cwd, binder, resolver).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
