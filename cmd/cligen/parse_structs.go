package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/packages"
)

func (gen *Generator) parseStructs(pkgs []*packages.Package, root *Command) error {
	for _, cmd := range root.CommandList() {
		if cmd.Struct == "" {
			continue
		}

		pkg, err := gen.loadPackage(pkgs, cmd.StructPath)
		if err != nil {
			return err
		}

		found := false
		for i, candidate := range pkg.Syntax {
			for _, decl := range candidate.Decls {
				if g, ok := decl.(*ast.GenDecl); ok && g.Tok == token.TYPE {
					for _, item := range g.Specs {
						if ts, ok := item.(*ast.TypeSpec); ok && ts.Name != nil && ts.Name.Name == cmd.StructName {
							if s, ok := ts.Type.(*ast.StructType); ok {
								imports := importsNames(pkg.Syntax[i], pkg.TypesInfo)

								for _, field := range s.Fields.List {
									if len(field.Names) == 0 {
										return fmt.Errorf("%s: unsupported field: embedded fields are not supported", cmd.Struct)
									}
									if len(field.Names) != 1 {
										return fmt.Errorf("%s: unsupported field: multiple field names are not supported", cmd.Struct)
									}

									arg := &Arg{Name: field.Names[0].Name, Flag: field.Names[0].Name}

									switch p := field.Type.(type) {
									case *ast.Ident:
										arg.Type = p.Name
									case *ast.SelectorExpr:
										pkg, ok := p.X.(*ast.Ident)
										if ok {
											arg.Type = fmt.Sprintf("%s.%s", pkg.Name, p.Sel.Name)
											gen.addImport(imports[pkg.Name], pkg.Name)
										}
									case *ast.ArrayType:
										switch item := p.Elt.(type) {
										case *ast.Ident:
											arg.Type = "[]" + item.Name
										case *ast.SelectorExpr:
											pkg, ok := item.X.(*ast.Ident)
											if ok {
												arg.Type = fmt.Sprintf("[]%s.%s", pkg.Name, item.Sel.Name)
												gen.addImport(imports[pkg.Name], pkg.Name)
											}
										}
									}

									if arg.Name == "" || arg.Type == "" {
										return fmt.Errorf("%s: unsupported field: %q", cmd.Struct, arg.Name)
									}

									if !arg.Native() {
										gen.Imports["encoding"] = ""
									}

									if cg := field.Doc; cg != nil {
										arg.Doc = strings.TrimSuffix(cg.Text(), "\n")

										for _, line := range cg.List {
											if d, ok := strings.CutPrefix(line.Text, "//cli:"); ok {
												arg.Directives = append(arg.Directives, d)
											}
										}
									}

									cmd.Args = append(cmd.Args, arg)
								}

								found = true
								break
							}
						}
					}
				}
			}
		}

		if !found {
			return fmt.Errorf("%s: struct not found or not declared as a struct type literal: %q", cmd.Path, cmd.Struct)
		}
	}

	return nil
}
