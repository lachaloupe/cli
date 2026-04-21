package main

import (
	"fmt"
	"go/ast"
	"strings"

	"golang.org/x/tools/go/packages"
)

func (gen *Generator) parseFunctions(pkgs []*packages.Package, root *Command) error {
	for _, cmd := range root.CommandList() {
		if cmd.Handler == "" {
			continue
		}

		pkg, name, alias, err := gen.resolvePackageSymbol(pkgs, cmd.Handler, "handler")
		if err != nil {
			return err
		}

		fn := func() *ast.FuncDecl {
			for _, file := range pkg.Syntax {
				for _, decl := range file.Decls {
					if candidate, ok := decl.(*ast.FuncDecl); ok && candidate.Name != nil && candidate.Recv == nil && candidate.Name.Name == name {
						return candidate
					}
				}
			}

			return nil
		}()

		if fn == nil {
			return fmt.Errorf("missing handler %q", cmd.Handler)
		}

		if cg := fn.Doc; cg != nil {
			cmd.Doc = strings.TrimSuffix(cg.Text(), "\n")

			for _, line := range cg.List {
				if d, ok := strings.CutPrefix(line.Text, "//cli:"); ok {
					cmd.Directives = append(cmd.Directives, d)
				}
			}
		}

		valid := false
		if fn.Type != nil && fn.Type.Params != nil {
			if len(fn.Type.Params.List) > 0 && len(fn.Type.Params.List) <= 2 {
				if ctx, ok := fn.Type.Params.List[0].Type.(*ast.SelectorExpr); ok {
					if p, ok := ctx.X.(*ast.Ident); ok && p.Name == "context" && ctx.Sel.Name == "Context" {
						if fn.Type.Results != nil && len(fn.Type.Results.List) == 1 {
							if result, ok := fn.Type.Results.List[0].Type.(*ast.Ident); ok && result.Name == "error" {
								valid = true

								if len(fn.Type.Params.List) == 2 {
									valid = false

									switch arg := fn.Type.Params.List[1].Type.(type) {
									case *ast.Ident:
										if alias == "" {
											cmd.Struct = arg.Name
										} else {
											cmd.Struct = alias + "." + arg.Name
										}
										valid = true
									case *ast.SelectorExpr:
										if alias != "" {
											return fmt.Errorf("%s: imported handlers must use an args struct from the same package", cmd.Handler)
										}

										if pkg, ok := arg.X.(*ast.Ident); ok {
											cmd.Struct = pkg.Name + "." + arg.Sel.Name
											valid = true
										}
									}
								}
							}
						}
					}
				}
			}
		}

		if !valid {
			return fmt.Errorf("%s: expecting Handler to be func(context.Context[, arg SomeArgs]) error", cmd.Handler)
		}
	}

	return nil
}
