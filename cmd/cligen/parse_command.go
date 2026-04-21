package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"unicode"
)

func (gen *Generator) parseCommand(cmd *Command, lit *ast.CompositeLit) error {
	seen := make(map[string]struct{})

	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			return fmt.Errorf("%s: expecting Name: Type", cmd.Path)
		}

		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			return fmt.Errorf("%s: expecting named fields", cmd.Path)
		}

		if _, ok := seen[key.Name]; ok {
			return fmt.Errorf("%s: duplicate field %q", cmd.Path, key.Name)
		}

		seen[key.Name] = struct{}{}

		switch key.Name {
		case "Commands":
			if err := gen.parseSubcommands(cmd, kv.Value); err != nil {
				return err
			}
		case "Handler", "New", "Lookup", "Open":
			value := ""
			if p, ok := kv.Value.(*ast.Ident); ok {
				value = p.Name
			} else if p, ok := kv.Value.(*ast.SelectorExpr); ok {
				if pkg, ok := p.X.(*ast.Ident); ok {
					if path := gen.SourceImports[pkg.Name]; path != "" {
						if pkg.Name == filepath.Base(path) {
							gen.Imports[path] = ""
						} else {
							gen.Imports[path] = pkg.Name
						}
					}

					value = fmt.Sprintf("%s.%s", pkg.Name, p.Sel.Name)
				}
			}

			if value == "" {
				return fmt.Errorf("%s: expecting %s to be a function name", cmd.Path, key.Name)
			}

			switch key.Name {
			case "Handler":
				cmd.Handler = value
			case "New":
				cmd.New = value
			case "Lookup":
				cmd.Lookup = value
			case "Open":
				cmd.Open = value
			}
		case "Name":
			value := ""
			if p, ok := kv.Value.(*ast.BasicLit); ok {
				if p.Kind == token.STRING && len(p.Value) > 2 {
					value = p.Value[1 : len(p.Value)-1]
				}
			}

			if value == "" {
				return fmt.Errorf("%s: expected Name: \"name\"", cmd.Path)
			}

			for _, c := range value {
				if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' && c != '-' && c != '.' {
					return fmt.Errorf("%s: invalid Name: %q", cmd.Path, value)
				}
			}

			cmd.Path = filepath.Join(filepath.Dir(cmd.Path), value)
			cmd.Name = value
		case "Help":
			value := ""
			if p, ok := kv.Value.(*ast.BasicLit); ok {
				if p.Kind == token.STRING && len(p.Value) > 2 {
					value = p.Value[1 : len(p.Value)-1]
				}
			}

			if value == "" {
				return fmt.Errorf("%s: expected Help: \"text\"", cmd.Path)
			}

			cmd.Help = value
		default:
			return fmt.Errorf("%s: cli.Command do not have a field named %q", cmd.Path, key.Name)
		}
	}

	return nil
}

func (gen *Generator) parseSubcommands(cmd *Command, value ast.Expr) error {
	if lit, ok := value.(*ast.CompositeLit); ok {
		if arr, ok := lit.Type.(*ast.ArrayType); ok {
			if ptr, ok := arr.Elt.(*ast.StarExpr); ok {
				if sel, ok := ptr.X.(*ast.SelectorExpr); ok {
					if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "cli" && sel.Sel.Name == "Command" {
						for i, elt := range lit.Elts {
							if lit, ok := elt.(*ast.CompositeLit); ok {
								sub := &Command{
									Path: filepath.Join(cmd.Path, fmt.Sprintf("%d", i)),
								}

								if err := gen.parseCommand(sub, lit); err != nil {
									return err
								}

								cmd.Commands = append(cmd.Commands, sub)
							}
						}
					}
				}
			}
		}
	}

	if len(cmd.Commands) == 0 {
		return fmt.Errorf("%s: expecting Commands: []*cli.Command{}", cmd.Path)
	}

	return nil
}
