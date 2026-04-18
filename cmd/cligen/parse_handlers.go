package main

import (
	"fmt"
	"go/ast"
	"strings"

	"golang.org/x/tools/go/packages"
)

func (gen *Generator) parseFunctions(pkgs []*packages.Package, root *Command) error {
	handlers := make(map[string]*Command)
	validHandlerSignature := func(f *ast.FuncDecl, cmd *Command) bool {
		valid := make([]bool, 2)

		if f.Type != nil && f.Type.Params != nil && len(f.Type.Params.List) <= 2 {
			for i, lst := range f.Type.Params.List {
				switch i {
				case 0:
					if s, ok := lst.Type.(*ast.SelectorExpr); ok {
						if pkg, ok := s.X.(*ast.Ident); ok && pkg.Name == "context" && s.Sel.Name == "Context" {
							valid[0] = true
						}
					}
				case 1:
					switch p := lst.Type.(type) {
					case *ast.Ident:
						cmd.Struct = p.Name
					case *ast.SelectorExpr:
						if pkg, ok := p.X.(*ast.Ident); ok {
							cmd.Struct = fmt.Sprintf("%s.%s", pkg.Name, p.Sel.Name)
						}
					}

					valid[0] = valid[0] && cmd.Struct != ""
				}
			}
		}

		if f.Type != nil && f.Type.Results != nil && len(f.Type.Results.List) == 1 {
			r := f.Type.Results.List[0]
			if s, ok := r.Type.(*ast.Ident); ok && s.Name == "error" {
				valid[1] = true
			}
		}

		return valid[0] && valid[1]
	}

	for _, c := range root.CommandList() {
		if h := c.Handler; h != "" {
			handlers[h] = c
		}
	}

	found := make(map[string]struct{})

	for _, file := range pkgs[0].Syntax {
		for _, decl := range file.Decls {
			f, ok := decl.(*ast.FuncDecl)

			if !ok || f.Name == nil || f.Recv != nil {
				continue
			}

			cmd, ok := handlers[f.Name.Name]
			if !ok {
				continue
			}

			found[f.Name.Name] = struct{}{}

			if cg := f.Doc; cg != nil {
				cmd.Doc = strings.TrimSuffix(cg.Text(), "\n")

				for _, line := range cg.List {
					d, ok := strings.CutPrefix(line.Text, "//cli:")
					if !ok {
						continue
					}

					cmd.Directives = append(cmd.Directives, d)
				}
			}

			if !validHandlerSignature(f, cmd) {
				return fmt.Errorf("%s: expecting Handler to be func(context.Context[, arg SomeArgs]) error", cmd.Handler)
			}
		}
	}

	for h := range handlers {
		if _, ok := found[h]; !ok {
			return fmt.Errorf("missing handler %q", h)
		}
	}

	return nil
}
