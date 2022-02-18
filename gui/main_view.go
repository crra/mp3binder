package gui

import (
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type bindServiceConfig struct {
	aBool      bool
	cover      string
	output     string
	mediaFiles string
	tags       map[string]string
}

type mainView struct {
	model      bindServiceConfig
	bs         bindService
	router     router
	theme      *material.Theme
	lineEditor *widget.Editor
	aBool      *widget.Bool
}

func newMainView(r router, bs bindService, theme *material.Theme) view {
	return &mainView{
		router: r,
		bs:     bs,
		theme:  theme,
		model:  bindServiceConfig{},
		aBool:  new(widget.Bool),
		lineEditor: &widget.Editor{
			SingleLine: true,
			Submit:     true,
		},
	}
}

func (v *mainView) Render(e system.FrameEvent) error {
	if v.aBool.Changed() {
		v.model.aBool = v.aBool.Value
	}

	ops := new(op.Ops)
	gtx := layout.NewContext(ops, e)

	layout.Flex{
		Axis:    layout.Vertical,
		Spacing: layout.SpaceStart,
	}.Layout(gtx,
		layout.Rigid(
			func(gtx C) D {
				e := material.Editor(v.theme, v.lineEditor, "Hint")
				e.Font.Style = text.Italic
				// border := widget.Border{Color: color.NRGBA{A: 0xff}, CornerRadius: unit.Dp(8), Width: unit.Px(2)}
				// return border.Layout(gtx, func(gtx C) D {
				return layout.UniformInset(unit.Dp(8)).Layout(gtx, e.Layout)
				//})
			}),
		layout.Rigid(
			func(gtx C) D {
				margins := layout.Inset{
					Top:    unit.Dp(25),
					Bottom: unit.Dp(25),
					Right:  unit.Dp(35),
					Left:   unit.Dp(35),
				}
				return margins.Layout(gtx,
					func(gtx C) D {
						return layout.Inset{Left: unit.Dp(16)}.Layout(gtx,
							material.Switch(v.theme, v.aBool, "Create chapters for every file").Layout,
						)
					})
			}),
	)
	e.Frame(gtx.Ops)

	return nil
}
