package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	gio "gioui.org/app"
	"github.com/cloudfoundry/jibber_jabber"
	"github.com/crra/mp3binder/gui"
	"github.com/crra/mp3binder/mp3binder"
	"github.com/crra/mp3binder/mp3binder/tags"
	"github.com/spf13/afero"
	"golang.org/x/text/language"
)

var (
	// externally set by the build system.
	version = "dev-build"
	name    = "mp3binder"
	realm   = "mp3binder"
	url     = "https://github.com/crra/mp3binder"
)

var defaultLocale = language.AmericanEnglish.String()

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

	resolver := tags.NewV24(gui.ErrTagNonStandard)
	binder := mp3binder.New(resolver)

	userLocale, err := jibber_jabber.DetectIETF()
	if err != nil {
		userLocale = defaultLocale
	}

	go func() {
		if err := gui.New(context, url, name, version, fs, cwd, binder, resolver, userLocale).Execute(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		os.Exit(0)
	}()
	gio.Main()
}
