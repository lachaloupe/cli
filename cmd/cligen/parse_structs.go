package main

import (
	"fmt"
	"go/ast"
	"go/token"
	pathpkg "path"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

func (gen *Generator) parseStructs(pkgs []*packages.Package, root *Command) error {
	structs := make(map[string]*Command)
	importsByName := func(file *ast.File) map[string]string {
		imports := make(map[string]string, len(file.Imports))

		for _, spec := range file.Imports {
			if spec.Path == nil {
				continue
			}

			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil || path == "" {
				continue
			}

			if spec.Name != nil {
				if spec.Name.Name == "_" || spec.Name.Name == "." {
					continue
				}

				imports[spec.Name.Name] = path
				continue
			}

			imports[pathpkg.Base(path)] = path
		}

		return imports
	}

	for _, c := range root.CommandList() {
		if s := c.Struct; s != "" {
			structs[s] = c
		}
	}

	found := make(map[string]struct{})

	for _, file := range pkgs[0].Syntax {
		imports := importsByName(file)

		for _, decl := range file.Decls {
			if g, ok := decl.(*ast.GenDecl); ok && g.Tok == token.TYPE {
				for _, s := range g.Specs {
					ts, ok := s.(*ast.TypeSpec)

					if !ok || ts.Name == nil {
						continue
					}

					cmd, ok := structs[ts.Name.Name]
					if !ok {
						continue
					}

					found[ts.Name.Name] = struct{}{}

					s, ok := ts.Type.(*ast.StructType)
					if !ok {
						return fmt.Errorf("%s: expecting struct", cmd.Struct)
					}

					for _, f := range s.Fields.List {
						arg, err := gen.parseStructField(imports, cmd, f)
						if err != nil {
							return err
						}

						cmd.Args = append(cmd.Args, arg)
					}
				}
			}
		}
	}

	for s := range structs {
		if _, ok := found[s]; !ok {
			return fmt.Errorf("missing struct %q", s)
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

			if path, ok := imports[pkg.Name]; ok {
				gen.Imports[path] = struct{}{}
			}
		}
	case *ast.ArrayType:
		switch item := p.Elt.(type) {
		case *ast.Ident:
			arg.Type = "[]" + item.Name
		case *ast.SelectorExpr:
			if pkg, ok := item.X.(*ast.Ident); ok {
				arg.Type = fmt.Sprintf("[]%s.%s", pkg.Name, item.Sel.Name)

				if path, ok := imports[pkg.Name]; ok {
					gen.Imports[path] = struct{}{}
				}
			}
		}
	}

	if arg.Name == "" || arg.Type == "" {
		return nil, fmt.Errorf("%s: unsupported field: %q", cmd.Struct, arg.Name)
	}

	if !arg.Native() {
		gen.Imports["encoding"] = struct{}{}
	}

	if cg := f.Doc; cg != nil {
		arg.Doc = strings.TrimSuffix(cg.Text(), "\n")

		for _, line := range cg.List {
			d, ok := strings.CutPrefix(line.Text, "//cli:")
			if !ok {
				continue
			}

			arg.Directives = append(arg.Directives, d)
		}
	}

	return arg, nil
}

func (gen *Generator) add(cmd *Command) error {
	if err := cmd.Process(); err != nil {
		return err
	}

	gen.Cmds = append(gen.Cmds, cmd)
	return nil
}
