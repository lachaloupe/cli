package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"
	"unicode"

	"golang.org/x/tools/go/packages"
)

func Parse(filename string) (*Generator, error) {
	filename, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}

	cfg := &packages.Config{
		Mode: packages.NeedSyntax | packages.NeedName | packages.NeedFiles,
		Dir:  filepath.Dir(filename),
	}

	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, err
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no package found in %s", filepath.Dir(filename))
	}

	if len(pkgs[0].Errors) > 0 {
		return nil, pkgs[0].Errors[0]
	}

	gen := Generator{
		Imports: map[string]struct{}{
			"context":                   {},
			"github.com/lachaloupe/cli": {},
		},
	}

	for i, gofile := range pkgs[0].GoFiles {
		if f, err := filepath.Abs(gofile); err != nil || f != filename {
			continue
		}

		file := pkgs[0].Syntax[i]

		for _, decl := range file.Decls {
			if g, ok := decl.(*ast.GenDecl); ok && g.Tok == token.VAR {
				for _, s := range g.Specs {
					vs, ok := s.(*ast.ValueSpec)
					if !ok {
						continue
					}

					for i, v := range vs.Values {
						lit, ok := v.(*ast.CompositeLit)
						if ok {
							sel, ok := lit.Type.(*ast.SelectorExpr)
							if ok {
								pkg, ok := sel.X.(*ast.Ident)
								if ok && pkg.Name == "cli" && sel.Sel.Name == "Command" {
									cmd := &Command{
										ID:   vs.Names[i].Name,
										Path: "/",
									}

									if err := gen.parseCommand(cmd, lit); err != nil {
										return nil, err
									}

									if err := gen.parseFunctions(pkgs, cmd); err != nil {
										return nil, err
									}

									if err := gen.parseStructs(pkgs, cmd); err != nil {
										return nil, err
									}

									if err := gen.add(cmd); err != nil {
										return nil, err
									}

									break
								}
							}
						}
					}
				}
			}
		}
	}

	if len(gen.Cmds) == 0 {
		return nil, fmt.Errorf("no cli.Command variable found in %s", filename)
	}

	return &gen, nil
}

func (gen *Generator) parseCommand(cmd *Command, lit *ast.CompositeLit) error {
	seen := make(map[string]struct{})

	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			return fmt.Errorf("%s: expecting Name:Type", cmd.Path)
		}

		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			return fmt.Errorf("%s: expecting fields to be named", cmd.Path)
		}

		if _, ok := seen[key.Name]; ok {
			return fmt.Errorf("%s: duplicate field %q", cmd.Path, key.Name)
		}

		seen[key.Name] = struct{}{}

		switch key.Name {
		case "Commands":
			if lit, ok := kv.Value.(*ast.CompositeLit); ok {
				if arr, ok := lit.Type.(*ast.ArrayType); ok {
					if ptr, ok := arr.Elt.(*ast.StarExpr); ok {
						if sel, ok := ptr.X.(*ast.SelectorExpr); ok {
							if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "cli" && sel.Sel.Name == "Command" {
								for i, elt := range lit.Elts {
									p, ok := elt.(*ast.CompositeLit)
									if !ok {
										return fmt.Errorf("%s: expecting cli.Command literal", cmd.Path)
									}

									// set the index as the placeholder until the name is known
									sub := &Command{
										Path: filepath.Join(cmd.Path, fmt.Sprintf("%d", i)),
									}

									if err := gen.parseCommand(sub, p); err != nil {
										return err
									}

									cmd.Commands = append(cmd.Commands, sub)
								}

								continue
							}
						}
					}
				}
			}

			return fmt.Errorf("%s: expecting Commands: []*cli.Command{}", cmd.Path)
		case "Handler":
			switch p := kv.Value.(type) {
			case *ast.Ident:
				cmd.Handler = p.Name
			case *ast.SelectorExpr:
				if pkg, ok := p.X.(*ast.Ident); ok {
					cmd.Handler = fmt.Sprintf("%s.%s", pkg.Name, p.Sel.Name)
				}
			}

			if cmd.Handler == "" {
				return fmt.Errorf("%s: expecting Handler to be a function name", cmd.Path)
			}
		case "New":
			switch p := kv.Value.(type) {
			case *ast.Ident:
				cmd.New = p.Name
			case *ast.SelectorExpr:
				if pkg, ok := p.X.(*ast.Ident); ok {
					cmd.New = fmt.Sprintf("%s.%s", pkg.Name, p.Sel.Name)
				}
			}

			if cmd.New == "" {
				return fmt.Errorf("%s: expecting New to be a function name", cmd.Path)
			}
		case "Name":
			p, ok := kv.Value.(*ast.BasicLit)
			if !ok || p.Kind != token.STRING || len(p.Value) <= 2 {
				return fmt.Errorf("%s: expected Name: \"name\"", cmd.Path)
			}

			s := p.Value[1 : len(p.Value)-1]

			for _, c := range s {
				if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' && c != '-' && c != '.' {
					return fmt.Errorf("%s: invalid Name: %s", cmd.Path, p.Value)
				}
			}

			cmd.Path = filepath.Join(filepath.Dir(cmd.Path), s)
			cmd.Name = s
		case "Help":
			p, ok := kv.Value.(*ast.BasicLit)
			if !ok || p.Kind != token.STRING || len(p.Value) <= 2 {
				return fmt.Errorf("%s: expected Help: \"text\"", cmd.Path)
			}

			cmd.Help = p.Value[1 : len(p.Value)-1]
		default:
			return fmt.Errorf("%s: cli.Command do not have a field named %q", cmd.Path, key)
		}
	}

	return nil
}

func (gen *Generator) parseFunctions(pkgs []*packages.Package, root *Command) error {
	handlers := make(map[string]*Command)

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

			if valid[0] == false || valid[1] == false {
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

func (gen *Generator) parseStructs(pkgs []*packages.Package, root *Command) error {
	structs := make(map[string]*Command)

	for _, c := range root.CommandList() {
		if s := c.Struct; s != "" {
			structs[s] = c
		}
	}

	found := make(map[string]struct{})

	for _, file := range pkgs[0].Syntax {
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

								gen.Imports[pkg.Name] = struct{}{}
							}
						case *ast.ArrayType:
							switch item := p.Elt.(type) {
							case *ast.Ident:
								arg.Type = "[]" + item.Name
							case *ast.SelectorExpr:
								if pkg, ok := item.X.(*ast.Ident); ok {
									arg.Type = fmt.Sprintf("[]%s.%s", pkg.Name, item.Sel.Name)

									gen.Imports[pkg.Name] = struct{}{}
								}
							}
						}

						if arg.Name == "" || arg.Type == "" {
							return fmt.Errorf("%s: unsupported field: %q", cmd.Struct, arg.Name)
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

func (gen *Generator) add(cmd *Command) error {
	if err := cmd.Process(); err != nil {
		return err
	}

	gen.Cmds = append(gen.Cmds, cmd)
	return nil
}
