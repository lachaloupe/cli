package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"strconv"

	"golang.org/x/tools/go/packages"
)

func Parse(filename string, providers []string) (*Generator, error) {
	filename, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}

	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedSyntax | packages.NeedName | packages.NeedFiles | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedDeps,
		Dir:  filepath.Dir(filename),
	}, ".")
	if err != nil {
		return nil, err
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no package found in %s", filepath.Dir(filename))
	}

	if len(pkgs[0].Errors) > 0 {
		errs := make([]error, len(pkgs[0].Errors))
		for i, err := range pkgs[0].Errors {
			errs[i] = err
		}

		return nil, errors.Join(errs...)
	}

	gen := &Generator{
		Fset: pkgs[0].Fset,
		Imports: map[string]string{
			"context":                   "",
			"github.com/lachaloupe/cli": "",
		},
		Package:       pkgs[0].Name,
		PackagePath:   pkgs[0].PkgPath,
		SourceImports: map[string]string{},
		TypesInfo:     pkgs[0].TypesInfo,
		Providers:     map[string]struct{}{},
	}

	for _, provider := range providers {
		switch provider {
		case "aws":
		default:
			return nil, fmt.Errorf("unsupported provider %q", provider)
		}

		gen.Providers[provider] = struct{}{}
	}

	if _, ok := gen.Providers["aws"]; ok {
		gen.Imports["fmt"] = ""
		gen.Imports["io"] = ""
		gen.Imports["net/url"] = ""
		gen.Imports["strings"] = ""
		gen.Imports["github.com/aws/aws-sdk-go-v2/aws"] = ""
		gen.Imports["github.com/aws/aws-sdk-go-v2/config"] = ""
		gen.Imports["github.com/aws/aws-sdk-go-v2/service/s3"] = ""
		gen.Imports["github.com/aws/aws-sdk-go-v2/service/secretsmanager"] = ""
		gen.Imports["github.com/aws/aws-sdk-go-v2/service/ssm"] = ""
	}

	for i, file := range pkgs[0].GoFiles {
		path, err := filepath.Abs(file)
		if err != nil || path != filename {
			continue
		}

		f := pkgs[0].Syntax[i]
		gen.SourceImports = importsNames(f, pkgs[0].TypesInfo)

		for _, decl := range f.Decls {
			g, ok := decl.(*ast.GenDecl)
			if !ok || g.Tok != token.VAR {
				continue
			}

			for _, spec := range g.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}

				for i, value := range vs.Values {
					lit, ok := value.(*ast.CompositeLit)
					if !ok {
						continue
					}

					sel, ok := lit.Type.(*ast.SelectorExpr)
					if !ok {
						continue
					}

					pkg, ok := sel.X.(*ast.Ident)
					if !ok || pkg.Name != "cli" || sel.Sel.Name != "Command" {
						continue
					}

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

					if err := cmd.Process(); err != nil {
						return nil, err
					}

					gen.Cmds = append(gen.Cmds, cmd)
					break
				}
			}
		}
	}

	if len(gen.Cmds) == 0 {
		return nil, fmt.Errorf("no cli.Command variable found in %s", filename)
	}

	return gen, nil
}

func (gen *Generator) addImport(path, name string) string {
	if path == "" {
		return ""
	}

	if p := gen.Imports[path]; p != "" {
		return p
	}

	if name == "" || name == filepath.Base(path) {
		gen.Imports[path] = ""
		return filepath.Base(path)
	}

	gen.Imports[path] = name
	return name
}

func (gen *Generator) loadPackage(pkgs []*packages.Package, path string) (*packages.Package, error) {
	if path == "" || path == gen.PackagePath {
		return pkgs[0], nil
	}

	loaded, err := packages.Load(&packages.Config{
		Mode: packages.NeedSyntax | packages.NeedName | packages.NeedFiles | packages.NeedTypes | packages.NeedTypesInfo,
		Dir:  filepath.Dir(pkgs[0].GoFiles[0]),
	}, path)

	if err != nil {
		return nil, err
	}

	if len(loaded) == 0 {
		return nil, fmt.Errorf("no package found for %q", path)
	}

	if len(loaded[0].Errors) > 0 {
		errs := make([]error, len(loaded[0].Errors))
		for i, err := range loaded[0].Errors {
			errs[i] = err
		}

		return nil, errors.Join(errs...)
	}

	return loaded[0], nil
}

func importsNames(file *ast.File, info *types.Info) map[string]string {
	names := make(map[string]string, len(file.Imports))

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

			names[spec.Name.Name] = path
			continue
		}

		if info != nil {
			if name, ok := info.Implicits[spec].(*types.PkgName); ok {
				names[name.Name()] = path
				continue
			}
		}

		names[filepath.Base(path)] = path
	}

	return names
}
