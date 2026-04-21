package main

import (
	"fmt"
	"go/ast"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

func importsNames(file *ast.File) map[string]string {
	names := make(map[string]string, len(file.Imports))

	for _, spec := range file.Imports {
		if spec.Path == nil {
			continue
		}

		p, err := strconv.Unquote(spec.Path.Value)
		if err != nil || p == "" {
			continue
		}

		if spec.Name != nil {
			if spec.Name.Name == "_" || spec.Name.Name == "." {
				continue
			}

			names[spec.Name.Name] = p
			continue
		}

		names[path.Base(p)] = p
	}

	return names
}

func (gen *Generator) resolvePackageSymbol(pkgs []*packages.Package, ref, kind string) (*packages.Package, string, string, error) {
	alias, name, ok := strings.Cut(ref, ".")
	if !ok {
		return pkgs[0], ref, "", nil
	}

	if alias == "" || name == "" || strings.Contains(name, ".") {
		return nil, "", "", fmt.Errorf("missing %s %q", kind, ref)
	}

	path := gen.SourceImports[alias]
	if path == "" {
		return nil, "", "", fmt.Errorf("missing import for %q", alias)
	}

	cfg := &packages.Config{
		Mode: packages.NeedSyntax | packages.NeedName | packages.NeedFiles,
		Dir:  filepath.Dir(pkgs[0].GoFiles[0]),
	}

	loaded, err := packages.Load(cfg, path)
	if err != nil {
		return nil, "", "", err
	}

	if len(loaded) == 0 {
		return nil, "", "", fmt.Errorf("no package found for %q", path)
	}

	if len(loaded[0].Errors) > 0 {
		return nil, "", "", loaded[0].Errors[0]
	}

	return loaded[0], name, alias, nil
}
