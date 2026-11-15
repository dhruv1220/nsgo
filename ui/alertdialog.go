package ui

import (
	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
)

// AlertAction is an action of an AlertDialog, a Button of its Variant,
// or filled with Color.
type AlertAction struct {
	Text    string
	Variant ButtonVariant
	Color   func() core.Color
	// Role identifies the cancel action Escape and back choose, or a
	// destructive action. Variant and Color still choose its look.
	Role core.AlertRole
	// Default takes keyboard focus on opening, so Return chooses it.
	Default bool
	OnPress func()
	TestID  string
}

// AlertDialog asks the user to confirm or acknowledge something: a Dialog
// whose actions close it, kept open by presses outside it.
type AlertDialog struct {
	Open         func() bool
	OnOpenChange func(open bool, reason base.Reason)
	// OnClose follows it being gone, once its motion ended.
	OnClose     func()
	Title       string
	Description string
	Actions     []AlertAction
	TestID      string
}

// Render implements core.Node.
func (a AlertDialog) Render(*core.Scope) core.Node {
	var footer []core.Node
	var cancel func()
	var confirm func()
	defaultSet, cancelSet := false, false
	for _, action := range a.Actions {
		if action.Role == core.AlertCancel && !cancelSet {
			cancel, cancelSet = action.OnPress, true
		}
		focus := action.Default && !defaultSet
		if focus {
			confirm = func() {
				if action.OnPress != nil {
					action.OnPress()
				}
			}
		}
		defaultSet = defaultSet || focus
		footer = append(footer, Button{Variant: action.Variant, Color: action.Color, Text: action.Text, TestID: action.TestID, AutoFocus: focus, OnPress: func() {
			if action.OnPress != nil {
				action.OnPress()
			}
			if a.OnOpenChange != nil {
				a.OnOpenChange(false, base.ReasonSelect)
			}
		}})
	}
	change := func(open bool, reason base.Reason) {
		if !open && reason == base.ReasonCancel && cancel != nil {
			cancel()
		}
		if a.OnOpenChange != nil {
			a.OnOpenChange(open, reason)
		}
	}
	return Dialog{Open: a.Open, OnOpenChange: change, OnClose: a.OnClose, OnConfirm: confirm, Title: a.Title, Description: a.Description, Footer: footer, Role: core.RoleAlertDialog,
		NoCloseOnBackdropPress: true, TestID: a.TestID}
}
