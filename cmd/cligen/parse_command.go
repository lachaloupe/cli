package main

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"path/filepath"
	"strconv"
	"unicode"
)

func (gen *Generator) parseCommand(cmd *Command, lit *ast.CompositeLit) error {
	seen := make(map[string]struct{})

	stringValue := func(value ast.Expr) (string, bool) {
		p, ok := value.(*ast.BasicLit)
		if !ok || p.Kind != token.STRING {
			if gen.TypesInfo == nil {
				return "", false
			}

			tv, ok := gen.TypesInfo.Types[value]
			if !ok || tv.Value == nil || tv.Value.Kind() != constant.String {
				return "", false
			}

			return constant.StringVal(tv.Value), true
		}

		s, err := strconv.Unquote(p.Value)
		if err != nil {
			return "", false
		}

		return s, true
	}

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
		case "Handler", "Renderer", "New", "Lookup", "Open", "Context":
			if gen.TypesInfo != nil {
				ast.Inspect(kv.Value, func(node ast.Node) bool {
					id, ok := node.(*ast.Ident)
					if !ok {
						return true
					}

					pkg, ok := gen.TypesInfo.Uses[id].(*types.PkgName)
					if !ok || pkg.Imported() == nil {
						return true
					}

					gen.addImport(pkg.Imported().Path(), id.Name)
					return true
				})
			}

			switch key.Name {
			case "Handler":
				cmd.HandlerExpr = kv.Value

				switch expr := kv.Value.(type) {
				case *ast.Ident:
					if gen.TypesInfo != nil {
						if _, ok := gen.TypesInfo.Uses[expr].(*types.Func); ok {
							cmd.HandlerRef = expr.Name
						}
					}
				case *ast.SelectorExpr:
					pkg, ok := expr.X.(*ast.Ident)
					if !ok || gen.TypesInfo == nil {
						break
					}

					if _, ok := gen.TypesInfo.Uses[pkg].(*types.PkgName); !ok {
						break
					}

					if _, ok := gen.TypesInfo.Uses[expr.Sel].(*types.Func); ok {
						cmd.HandlerRef = fmt.Sprintf("%s.%s", pkg.Name, expr.Sel.Name)
					}
				}
			case "Renderer":
				cmd.RendererExpr = kv.Value
			case "New":
				cmd.NewExpr = kv.Value
			case "Lookup":
				cmd.LookupExpr = kv.Value
			case "Open":
				cmd.OpenExpr = kv.Value
			}
		case "Name":
			value, ok := stringValue(kv.Value)
			if !ok || value == "" {
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
			value, ok := stringValue(kv.Value)
			if !ok || value == "" {
				return fmt.Errorf("%s: expected Help: \"text\"", cmd.Path)
			}

			cmd.Help = value
		case "Template":
			value, ok := stringValue(kv.Value)
			if !ok {
				return fmt.Errorf("%s: expected Template: \"text\"", cmd.Path)
			}

			cmd.Template = value
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
