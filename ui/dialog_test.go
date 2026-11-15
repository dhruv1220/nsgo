package ui

import (
	"slices"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
)

// TestDialog opens a dialog, centered on desktops and a sheet on phones,
// and closes it from its footer; an alert dialog keeps open on its
// backdrop and closes on an action.
func TestDialog(t *testing.T) {
	each(t, func(t *testing.T, e core.Environment) {
		var confirmed bool
		app, p := start(t, e, func(cx *core.Scope) core.Node {
			open, alert := core.NewSignal(cx, false), core.NewSignal(cx, false)
			return core.View{Style: core.Style{AlignItems: core.AlignStart, Gap: 8}, Children: []core.Node{
				Button{Text: "Rename", TestID: "open", OnPress: func() { open.Set(true) }},
				Button{Text: "Delete", TestID: "open-alert", OnPress: func() { alert.Set(true) }},
				Dialog{TestID: "dialog", Title: "Rename file", Description: "Choose a new name.",
					Open: open.Get, OnOpenChange: func(v bool, _ base.Reason) { open.Set(v) },
					Children: []core.Node{Input{Label: "Name", DefaultValue: "report.pdf"}},
					Footer: []core.Node{Button{Variant: ButtonSecondary, Text: "Cancel", TestID: "cancel", OnPress: func() { open.Set(false) }},
						Button{Variant: ButtonPrimary, Text: "Rename"}}},
				AlertDialog{TestID: "alert", Title: "Delete report.pdf?", Description: "This cannot be undone.",
					Open: alert.Get, OnOpenChange: func(v bool, _ base.Reason) { alert.Set(v) },
					Actions: []AlertAction{{Text: "Cancel", Variant: ButtonSecondary, TestID: "keep"},
						{Text: "Delete", Color: ColorsOf(cx).Danger, TestID: "confirm", OnPress: func() { confirmed = true }}}},
			}}
		})
		app.Press("open")
		d := app.Find("dialog")
		if e.Idiom == core.IdiomDesktop {
			if d.Style.CornerRadius != RadiusDialog {
				t.Fatal("the desktop dialog is not a rounded panel")
			}
			if _, _, _, a := d.Style.BorderColor.Components(); a != 1 || len(d.Style.Effects.Shadows) == 0 {
				t.Fatalf("the desktop dialog's rim %v is not solid over a shadow", d.Style.BorderColor)
			}
		} else if d.Style.Background != p.Surface || d.Style.Corners.BottomLeft != 0 || d.Style.Corners.TopLeft == 0 {
			t.Fatal("the phone dialog is not a sheet from the bottom")
		}
		app.Find("Rename file")
		app.Press("cancel")
		if e.Idiom == core.IdiomDesktop {
			app.Find("Rename file") // it fades out
			app.Advance(200 * time.Millisecond)
		}
		app.WaitForGone("Rename file", time.Second)
		app.Press("open-alert")
		app.Press("confirm")
		app.Advance(200 * time.Millisecond)
		app.WaitForGone("Delete report.pdf?", time.Second)
		if !confirmed {
			t.Fatal("the alert's action did not run")
		}
	})
}

func TestAlertDialogRoleAndActions(t *testing.T) {
	each(t, func(t *testing.T, e core.Environment) {
		for _, how := range []string{"return", "escape", "back", "press"} {
			t.Run(how, func(t *testing.T) {
				var calls []string
				var reason base.Reason
				var open *core.Signal[bool]
				closed := 0
				app, _ := start(t, e, func(cx *core.Scope) core.Node {
					open = core.NewSignal(cx, true)
					return AlertDialog{TestID: "alert", Title: "Delete the note?", Description: "This cannot be undone.",
						Open: open.Get, OnClose: func() { closed++ },
						OnOpenChange: func(v bool, r base.Reason) { calls = append(calls, "change"); reason = r; open.Set(v) },
						Actions: []AlertAction{
							{Text: "Cancel", Role: core.AlertCancel, TestID: "cancel", OnPress: func() { calls = append(calls, "cancel") }},
							{Text: "Delete", Role: core.AlertDestructive, Default: true, TestID: "delete", OnPress: func() { calls = append(calls, "delete") }},
						}}
				})
				alert := app.Find("role:alertdialog")
				if alert.Props.Label != "Delete the note?" || alert.Props.Description != "This cannot be undone." {
					t.Fatalf("alert accessibility = %+v", alert.Props)
				}
				if !app.Find("id:delete").Focused {
					t.Fatal("default action did not take initial focus")
				}
				wantAction, wantReason := "delete", base.ReasonSelect
				switch how {
				case "return":
					app.Key("Return")
				case "escape":
					app.Key("Escape")
					wantAction, wantReason = "cancel", base.ReasonCancel
				case "back":
					app.Back()
					wantAction, wantReason = "cancel", base.ReasonCancel
				case "press":
					app.Press("id:delete")
				}
				// A second close while the panel leaves cannot run an action twice.
				if how == "return" {
					app.Key("Return")
				}
				app.Key("Escape")
				if !slices.Equal(calls, []string{wantAction, "change"}) || reason != wantReason {
					t.Fatalf("calls %v, reason %v; want [%s change], %v", calls, reason, wantAction, wantReason)
				}
				app.Advance(time.Second)
				app.WaitForGone("id:alert", time.Second)
				if closed != 1 {
					t.Fatalf("OnClose ran %d times", closed)
				}
				app.Post(func() { open.Set(true) })
				app.WaitFor("id:delete", time.Second)
				if !app.Find("id:delete").Focused {
					t.Fatal("reopening did not focus the default action")
				}
			})
		}
	})
}

func TestAlertDialogWithoutActionMarkers(t *testing.T) {
	each(t, func(t *testing.T, e core.Environment) {
		pressed, changed := 0, 0
		var reason base.Reason
		app, _ := start(t, e, func(cx *core.Scope) core.Node {
			open := core.NewSignal(cx, true)
			return AlertDialog{TestID: "alert", Open: open.Get, OnOpenChange: func(v bool, r base.Reason) {
				changed++
				reason = r
				open.Set(v)
			}, Actions: []AlertAction{{Text: "Action", TestID: "action", OnPress: func() { pressed++ }}}}
		})
		if !app.Exists("role:alertdialog") {
			t.Fatal("unmarked alert lost its alert dialog role")
		}
		app.Key("Return")
		if pressed != 0 || changed != 0 {
			t.Fatalf("unmarked action ran on Return: pressed %d, changed %d", pressed, changed)
		}
		app.Key("Escape")
		if pressed != 0 || changed != 1 || reason != base.ReasonCancel {
			t.Fatalf("unmarked cancel: pressed %d, changed %d, reason %v", pressed, changed, reason)
		}
	})
}

func TestDialogPreservesAlertRoleOnTouch(t *testing.T) {
	app, _ := start(t, phoneEnv, func(cx *core.Scope) core.Node {
		return Dialog{Open: func() bool { return true }, Role: core.RoleAlertDialog, Title: "Confirm", Description: "Choose an action"}
	})
	if got := app.Find("role:alertdialog").Props.Description; got != "Choose an action" {
		t.Fatalf("sheet description %q", got)
	}
}

func TestAlertDefaultMayUpdateOpenItself(t *testing.T) {
	each(t, func(t *testing.T, e core.Environment) {
		var calls []string
		app, _ := start(t, e, func(cx *core.Scope) core.Node {
			open := core.NewSignal(cx, true)
			return AlertDialog{Open: open.Get, OnOpenChange: func(v bool, r base.Reason) {
				if r != base.ReasonSelect {
					t.Errorf("close reason %v", r)
				}
				calls = append(calls, "change")
				open.Set(v)
			}, Actions: []AlertAction{{Text: "Done", Default: true, OnPress: func() {
				calls = append(calls, "action")
				open.Set(false)
			}}}}
		})
		app.Key("Return")
		if !slices.Equal(calls, []string{"action", "change"}) {
			t.Fatalf("calls %v", calls)
		}
	})
}
