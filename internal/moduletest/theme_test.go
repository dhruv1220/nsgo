package moduletest

import (
	"bytes"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/types"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// themeReaders are what reads the app's theme, which no base component may (docs/architecture/themes.md): the scope's methods and core's functions, by name.
var themeReaders = []string{"Scope.Palette", "Scope.Theme", "DefaultTheme"}

// appearanceAware are the base packages that follow the appearance their subtree shows, cx.Dark(), as themes.md documents: the editor's, the diff's and highlighting's colors change between light and dark.
var appearanceAware = []string{"base/editor/", "base/diff/", "base/textview/highlight/"}

// TestBaseComponentsDoNotReadTheme fails when a package under base/ reads the theme, or reads cx.Dark() outside appearanceAware. It resolves calls by their types, so neither a scope named otherwise nor a method value escapes it.
func TestBaseComponentsDoNotReadTheme(t *testing.T) {
	if testing.Short() {
		t.Skip("type-checks every base package")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("needs the go command")
	}
	files, imp := typesImporter(t)
	corePkg, err := imp.Import(modulePath + "/core")
	if err != nil {
		t.Fatal(err)
	}
	scope := types.NewPointer(corePkg.Scope().Lookup("Scope").Type())
	method := func(name string) types.Object {
		sel := types.NewMethodSet(scope).Lookup(corePkg, name)
		if sel == nil {
			t.Fatalf("core.Scope has no method %s", name)
		}
		return sel.Obj()
	}
	forbidden := map[types.Object]string{}
	for _, name := range themeReaders {
		if m, ok := strings.CutPrefix(name, "Scope."); ok {
			forbidden[method(m)] = "cx." + m
		} else if obj := corePkg.Scope().Lookup(name); obj != nil {
			forbidden[obj] = "core." + name
		} else {
			t.Fatalf("core has no %s", name)
		}
	}
	dark := method("Dark")

	for _, p := range listBase(t) {
		rel, _ := inModule(p.ImportPath)
		// A file this platform does not build cannot be type-checked here; it may not reach core at all.
		for _, name := range p.IgnoredGoFiles {
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			f, err := parser.ParseFile(files, filepath.Join(p.Dir, name), nil, parser.ImportsOnly)
			if err != nil {
				t.Fatal(err)
			}
			for _, spec := range f.Imports {
				if path, _ := strconv.Unquote(spec.Path.Value); path == modulePath+"/core" {
					t.Errorf("%s/%s imports core but is not built on this platform, so its theme reads go unchecked", rel, name)
				}
			}
		}
		var parsed []*ast.File
		for _, name := range append(p.GoFiles, p.CgoFiles...) {
			f, err := parser.ParseFile(files, filepath.Join(p.Dir, name), nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			parsed = append(parsed, f)
		}
		info := &types.Info{Uses: map[*ast.Ident]types.Object{}}
		if _, err := (&types.Config{Importer: imp}).Check(p.ImportPath, files, parsed, info); err != nil {
			t.Fatalf("type-check %s: %v", p.ImportPath, err)
		}
		mayDark := matchesAny(rel, appearanceAware)
		for id, obj := range info.Uses {
			if what, ok := forbidden[obj]; ok {
				t.Errorf("%s: base component reads %s; base has no look, so the themed look belongs in ui", files.Position(id.Pos()), what)
			} else if obj == dark && !mayDark {
				t.Errorf("%s: base component reads cx.Dark outside the appearance-aware packages themes.md documents (%v)", files.Position(id.Pos()), appearanceAware)
			}
		}
	}
}

type basePackage struct {
	ImportPath, Dir                   string
	GoFiles, CgoFiles, IgnoredGoFiles []string
	Error                             *struct{ Err string }
}

// listBase lists the packages under base/ as this platform builds them.
func listBase(t *testing.T) []basePackage {
	t.Helper()
	cmd := exec.Command("go", "list", "-e", "-json=ImportPath,Dir,GoFiles,CgoFiles,IgnoredGoFiles,Error", "./base/...")
	cmd.Dir = moduleRoot
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v\n%s", err, stderr.Bytes())
	}
	var pkgs []basePackage
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var p basePackage
		if err := dec.Decode(&p); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if p.Error != nil && !strings.Contains(p.Error.Err, "build constraints exclude all Go files") {
			t.Errorf("%s: %s", p.ImportPath, p.Error.Err)
		}
		pkgs = append(pkgs, p)
	}
	if len(pkgs) < 10 {
		t.Fatalf("go list found only %d packages under base/", len(pkgs))
	}
	return pkgs
}
