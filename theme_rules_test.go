package nsgo

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// TestBaseComponentsDoNotReadTheme keeps unstyled components independent of
// the application's look. A base component may follow appearance only where
// its content, such as syntax highlighting, genuinely changes with light or
// dark rendering.
func TestBaseComponentsDoNotReadTheme(t *testing.T) {
	files := token.NewFileSet()
	err := filepath.WalkDir("base", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(files, path, nil, 0)
		if err != nil {
			return err
		}
		allowedDark := strings.HasPrefix(filepath.ToSlash(path), "base/editor/") ||
			strings.HasPrefix(filepath.ToSlash(path), "base/diff/") ||
			strings.HasPrefix(filepath.ToSlash(path), "base/textview/highlight/")
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			switch selector.Sel.Name {
			case "Palette", "Theme":
				if scopeSelector(selector) {
					pos := files.Position(selector.Pos())
					t.Errorf("%s:%d: base component reads cx.%s; move the themed look to ui", path, pos.Line, selector.Sel.Name)
				}
			case "Dark":
				if !allowedDark && scopeSelector(selector) {
					pos := files.Position(selector.Pos())
					t.Errorf("%s:%d: base component reads cx.Dark outside the documented appearance-aware packages", path, pos.Line)
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// scopeSelector recognizes cx.Dark(), including a receiver field such as
// v.cx.Dark(), while leaving ordinary Dark methods and fields alone.
func scopeSelector(selector *ast.SelectorExpr) bool {
	switch receiver := selector.X.(type) {
	case *ast.Ident:
		return receiver.Name == "cx"
	case *ast.SelectorExpr:
		return receiver.Sel.Name == "cx"
	case *ast.ParenExpr:
		inner, ok := receiver.X.(*ast.Ident)
		return ok && inner.Name == "cx"
	default:
		return false
	}
}
