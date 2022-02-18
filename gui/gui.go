package gui

import (
	"context"
	"errors"
	"fmt"
	"io"

	gio "gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/carolynvs/aferox"
	"github.com/spf13/afero"
)

type view interface {
	Render(system.FrameEvent) error
}

type Application interface {
	Execute() error
}

var ErrTagNonStandard = errors.New("non-standard tag")

type app struct {
	context context.Context
	url     string
	name    string
	version string

	fs  aferox.Aferox
	cwd string

	binder      binder
	tagResolver tagResolver

	theme *material.Theme

	// router
	currentView view
	mainView    view
}

type router interface {
	Switch(string) error
}

type bindService interface {
	Bind() error
}

type binder interface {
	Bind(context.Context, io.WriteSeeker, io.ReadWriteSeeker, []io.Reader, ...any) error
}

type tagResolver interface {
	DescriptionFor(string) (string, error)
}

func New(parent context.Context, url, name, version string, fs afero.Fs, cwd string, binder binder, tagResolver tagResolver, userLocale string) Application {
	app := &app{
		context:     parent,
		url:         url,
		name:        name,
		version:     version,
		theme:       material.NewTheme(gofont.Collection()),
		binder:      binder,
		tagResolver: tagResolver,
		fs:          aferox.NewAferox(cwd, fs),
		cwd:         cwd,
	}

	app.mainView = newMainView(app, app, app.theme)
	app.currentView = app.mainView

	return app
}

type (
	C = layout.Context
	D = layout.Dimensions
)

var errNoSuchView = errors.New("view not found")

func (a *app) Switch(id string) error {
	return errNoSuchView
}

func (a *app) Bind() error {
	return nil
}

func (a *app) Execute() error {
	w := gio.NewWindow(
		gio.Title(fmt.Sprintf("%s-%s", a.name, a.version)),
		gio.MinSize(unit.Dp(400), unit.Dp(600)),
	)

	for {
		select {
		case <-a.context.Done():
			a.shutdown()
			return a.context.Err()
		case e := <-w.Events():
			switch e := e.(type) {
			case system.DestroyEvent:
				a.shutdown()
				return e.Err
			case system.FrameEvent:
				if err := a.Render(e); err != nil {
					return err
				}
			}
		}
	}
}

func (a *app) shutdown() {
}

// application router
func (a *app) Render(e system.FrameEvent) error {
	return a.currentView.Render(e)
}
