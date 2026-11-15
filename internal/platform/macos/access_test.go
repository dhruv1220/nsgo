//go:build darwin && !ios

package macos

import (
	"testing"

	"github.com/nsgo-dev/nsgo/internal/driver"
)

func TestAXRoleEditableCombobox(t *testing.T) {
	got := (&view{kind: driver.KindInput, props: driver.ViewProps{Role: driver.RoleCombobox}}).axRole()
	if want := [2]string{"AXComboBox", ""}; got != want {
		t.Fatalf("editable combobox role = %#v, want %#v", got, want)
	}
}

func TestAXRoleSelectCombobox(t *testing.T) {
	got := (&view{kind: driver.KindView, props: driver.ViewProps{Role: driver.RoleCombobox}}).axRole()
	if want := [2]string{"AXPopUpButton", ""}; got != want {
		t.Fatalf("select combobox role = %#v, want %#v", got, want)
	}
}
