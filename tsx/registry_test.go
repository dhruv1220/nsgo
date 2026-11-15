package tsx

import (
	"testing"

	"github.com/nsgo-dev/nsgo/base"
)

func init() {
	registerPart("nsgo/base", "HiddenPart", base.ResizablePanel{})
}

// A part sandboxes import is marked SandboxSafe, as ResizablePanel is; one without it is not.
func TestPartsAreHiddenUnlessSandboxSafe(t *testing.T) {
	if isSandboxSafe("nsgo/base", "HiddenPart") || !isSandboxSafe("nsgo/base", "ResizablePanel") {
		t.Fatalf("HiddenPart %v, ResizablePanel %v", isSandboxSafe("nsgo/base", "HiddenPart"), isSandboxSafe("nsgo/base", "ResizablePanel"))
	}
}

// A function's package is its name up to the first dot after the last slash.
func TestFuncPackage(t *testing.T) {
	for name, want := range map[string]string{
		"github.com/nsgo-dev/nsgo/tsx/ui.init.0":          "github.com/nsgo-dev/nsgo/tsx/ui",
		"github.com/nsgo-dev/nsgo/tsx.RegisterClass[...]": "github.com/nsgo-dev/nsgo/tsx",
		"example.com/app.(*T).M.func1":                    "example.com/app",
		"gopkg.in/yaml%2ev3.Unmarshal":                    "gopkg.in/yaml%2ev3",
		"main.init.0":                                     "main",
		"runtime.call32":                                  "runtime",
	} {
		if got := funcPackage(name); got != want {
			t.Errorf("funcPackage(%q) = %q, want %q", name, got, want)
		}
	}
}
