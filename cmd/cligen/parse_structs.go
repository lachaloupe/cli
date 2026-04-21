package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

func (gen *Generator) parseStructs(pkgs []*packages.Package, root *Command) error {
	for _, cmd := range root.CommandList() {
		if cmd.Struct == "" {
			continue
		}

		pkg, name, _, err := gen.resolvePackageSymbol(pkgs, cmd.Struct, "struct")
		if err != nil {
			return err
		}

		for i, candidate := range pkg.Syntax {
			for _, decl := range candidate.Decls {
				if g, ok := decl.(*ast.GenDecl); ok && g.Tok == token.TYPE {
					for _, item := range g.Specs {
						if ts, ok := item.(*ast.TypeSpec); ok && ts.Name != nil && ts.Name.Name == name {
							if s, ok := ts.Type.(*ast.StructType); ok {
								imports := importsNames(pkg.Syntax[i])

								for _, field := range s.Fields.List {
									arg, err := gen.parseStructField(imports, cmd, field)
									if err != nil {
										return err
									}

									cmd.Args = append(cmd.Args, arg)
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}

func (gen *Generator) parseStructField(imports map[string]string, cmd *Command, f *ast.Field) (*Arg, error) {
	arg := &Arg{
		Name: f.Names[0].Name,
		Flag: f.Names[0].Name,
	}

	switch p := f.Type.(type) {
	case *ast.Ident:
		arg.Type = p.Name
	case *ast.SelectorExpr:
		if pkg, ok := p.X.(*ast.Ident); ok {
			arg.Type = fmt.Sprintf("%s.%s", pkg.Name, p.Sel.Name)

			if path := imports[pkg.Name]; path != "" {
				if pkg.Name == filepath.Base(path) {
					gen.Imports[path] = ""
				} else {
					gen.Imports[path] = pkg.Name
				}
			}
		}
	case *ast.ArrayType:
		switch item := p.Elt.(type) {
		case *ast.Ident:
			arg.Type = "[]" + item.Name
		case *ast.SelectorExpr:
			if pkg, ok := item.X.(*ast.Ident); ok {
				arg.Type = fmt.Sprintf("[]%s.%s", pkg.Name, item.Sel.Name)

				if path := imports[pkg.Name]; path != "" {
					if pkg.Name == filepath.Base(path) {
						gen.Imports[path] = ""
					} else {
						gen.Imports[path] = pkg.Name
					}
				}
			}
		}
	}

	if arg.Name == "" || arg.Type == "" {
		return nil, fmt.Errorf("%s: unsupported field: %q", cmd.Struct, arg.Name)
	}

	if !arg.Native() {
		gen.Imports["encoding"] = ""
	}

	if cg := f.Doc; cg != nil {
		arg.Doc = strings.TrimSuffix(cg.Text(), "\n")

		for _, line := range cg.List {
			if d, ok := strings.CutPrefix(line.Text, "//cli:"); ok {
				arg.Directives = append(arg.Directives, d)
			}
		}
	}

	return arg, nil
}
