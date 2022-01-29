package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/cloudfoundry/jibber_jabber"
	"github.com/crra/mp3binder/cli"
	"github.com/crra/mp3binder/mp3binder"
	"github.com/crra/mp3binder/mp3binder/tags"
	"github.com/spf13/afero"
	"golang.org/x/text/language"
)

var (
	// externally set by the build system.
	version = "dev-build"
	name    = "mp3builder"
	realm   = "mp3builder"
)

var defaultLocale = language.AmericanEnglish.String()

// main is the entrypoint of the program.
// main is the only place where external dependencies (e.g. output stream, logger, filesystem)
// are resolved and where final errors are handled (e.g. writing to the console).
func main() {
	// create a parent context that listens on os signals (e.g. CTRL-C)
	context, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	// cancel the parent context and all children if an os signal arrives
	go func() {
		<-context.Done()
		cancel()
	}()

	// dependencies
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fs := afero.NewOsFs()

	resolver := tags.NewV24(cli.ErrTagNonStandard)
	binder := mp3binder.New(resolver)

	userLocale, err := jibber_jabber.DetectIETF()
	if err != nil {
		userLocale = defaultLocale
	}

	// run the program and clean up
	if err := cli.New(context, name, version, os.Stdout, fs, cwd, binder, resolver, userLocale).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
