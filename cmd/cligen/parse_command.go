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
	selectorName := func(expr ast.Expr) (string, error) {
		switch p := expr.(type) {
		case *ast.Ident:
			return p.Name, nil
		case *ast.SelectorExpr:
			if pkg, ok := p.X.(*ast.Ident); ok {
				return fmt.Sprintf("%s.%s", pkg.Name, p.Sel.Name), nil
			}
		}

		return "", fmt.Errorf("expected selector")
	}
	stringLiteral := func(expr ast.Expr) (string, error) {
		p, ok := expr.(*ast.BasicLit)
		if !ok || p.Kind != token.STRING || len(p.Value) <= 2 {
			return "", fmt.Errorf("expected string")
		}

		return p.Value[1 : len(p.Value)-1], nil
	}

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
			if err := gen.parseSubcommands(cmd, kv.Value); err != nil {
				return err
			}
		case "Handler":
			value, err := selectorName(kv.Value)
			if err != nil {
				return fmt.Errorf("%s: expecting Handler to be a function name", cmd.Path)
			}
			cmd.Handler = value
		case "New":
			value, err := selectorName(kv.Value)
			if err != nil {
				return fmt.Errorf("%s: expecting New to be a function name", cmd.Path)
			}
			cmd.New = value
		case "Lookup":
			value, err := selectorName(kv.Value)
			if err != nil {
				return fmt.Errorf("%s: expecting Lookup to be a function name", cmd.Path)
			}
			cmd.Lookup = value
		case "Open":
			value, err := selectorName(kv.Value)
			if err != nil {
				return fmt.Errorf("%s: expecting Open to be a function name", cmd.Path)
			}
			cmd.Open = value
		case "Name":
			value, err := stringLiteral(kv.Value)
			if err != nil {
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
			value, err := stringLiteral(kv.Value)
			if err != nil {
				return fmt.Errorf("%s: expected Help: \"text\"", cmd.Path)
			}
			cmd.Help = value
		default:
			return fmt.Errorf("%s: cli.Command do not have a field named %q", cmd.Path, key)
		}
	}

	return nil
}

func (gen *Generator) parseSubcommands(cmd *Command, value ast.Expr) error {
	lit, ok := value.(*ast.CompositeLit)
	if !ok {
		return fmt.Errorf("%s: expecting Commands: []*cli.Command{}", cmd.Path)
	}

	arr, ok := lit.Type.(*ast.ArrayType)
	if !ok {
		return fmt.Errorf("%s: expecting Commands: []*cli.Command{}", cmd.Path)
	}

	ptr, ok := arr.Elt.(*ast.StarExpr)
	if !ok {
		return fmt.Errorf("%s: expecting Commands: []*cli.Command{}", cmd.Path)
	}

	sel, ok := ptr.X.(*ast.SelectorExpr)
	if !ok {
		return fmt.Errorf("%s: expecting Commands: []*cli.Command{}", cmd.Path)
	}

	pkg, ok := sel.X.(*ast.Ident)
	if !ok || pkg.Name != "cli" || sel.Sel.Name != "Command" {
		return fmt.Errorf("%s: expecting Commands: []*cli.Command{}", cmd.Path)
	}

	for i, elt := range lit.Elts {
		p, ok := elt.(*ast.CompositeLit)
		if !ok {
			return fmt.Errorf("%s: expecting cli.Command literal", cmd.Path)
		}

		sub := &Command{
			Path: filepath.Join(cmd.Path, fmt.Sprintf("%d", i)),
		}

		if err := gen.parseCommand(sub, p); err != nil {
			return err
		}

		cmd.Commands = append(cmd.Commands, sub)
	}

	return nil
}
