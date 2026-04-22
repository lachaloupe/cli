package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

func (gen *Generator) parseFunctions(pkgs []*packages.Package, root *Command) error {
	errorType := types.Universe.Lookup("error").Type()

	for _, cmd := range root.CommandList() {
		if cmd.HandlerExpr == nil {
			continue
		}

		if cmd.HandlerRef != "" {
			pkg, name := pkgs[0], cmd.HandlerRef

			if alias, item, ok := strings.Cut(cmd.HandlerRef, "."); ok {
				if alias == "" || item == "" || strings.Contains(item, ".") {
					return fmt.Errorf("%s: missing handler %q", cmd.Path, cmd.HandlerRef)
				}

				path := gen.SourceImports[alias]
				if path == "" {
					return fmt.Errorf("%s: missing import for %q", cmd.Path, alias)
				}

				loaded, err := gen.loadPackage(pkgs, path)
				if err != nil {
					return err
				}

				pkg, name = loaded, item
			}

			func() {
				for _, file := range pkg.Syntax {
					for _, decl := range file.Decls {
						fn, ok := decl.(*ast.FuncDecl)
						if !ok || fn.Name == nil || fn.Recv != nil || fn.Name.Name != name {
							continue
						}

						if doc := fn.Doc; doc != nil {
							cmd.Doc = strings.TrimSuffix(doc.Text(), "\n")
							for _, line := range doc.List {
								if d, ok := strings.CutPrefix(line.Text, "//cli:"); ok {
									cmd.Directives = append(cmd.Directives, d)
								}
							}
						}

						return
					}
				}
			}()
		}

		arg, ok := func() (*types.TypeName, bool) {
			if gen.TypesInfo == nil {
				return nil, false
			}

			t := gen.TypesInfo.TypeOf(cmd.HandlerExpr)
			if t == nil {
				return nil, false
			}

			sig, ok := types.Unalias(t).Underlying().(*types.Signature)
			if !ok || sig.Params() == nil || sig.Results() == nil {
				return nil, false
			}

			if sig.Params().Len() == 0 || sig.Params().Len() > 2 {
				return nil, false
			}

			if sig.Results().Len() != 1 || !types.Identical(sig.Results().At(0).Type(), errorType) {
				return nil, false
			}

			ctx, ok := types.Unalias(sig.Params().At(0).Type()).(*types.Named)
			if !ok {
				return nil, false
			}

			if obj := ctx.Obj(); obj == nil || obj.Name() != "Context" || obj.Pkg() == nil || obj.Pkg().Path() != "context" {
				return nil, false
			}

			if sig.Params().Len() == 1 {
				return nil, true
			}

			arg, ok := types.Unalias(sig.Params().At(1).Type()).(*types.Named)
			if !ok {
				return nil, false
			}

			if _, ok := arg.Underlying().(*types.Struct); !ok {
				return nil, false
			}

			obj := arg.Obj()
			return obj, obj != nil
		}()

		if !ok {
			return fmt.Errorf("%s: expecting Handler to be func(context.Context[, arg SomeArgs]) error", cmd.Path)
		}

		if arg == nil {
			continue
		}

		cmd.StructName = arg.Name()

		pkg := arg.Pkg()
		if pkg == nil || pkg.Path() == "" || pkg.Path() == gen.PackagePath {
			cmd.Struct = cmd.StructName
			cmd.StructPath = ""
			continue
		}

		cmd.StructPath = pkg.Path()

		alias := pkg.Name()
		for name, imported := range gen.SourceImports {
			if imported == cmd.StructPath {
				alias = gen.addImport(cmd.StructPath, name)
				break
			}
		}

		if alias == pkg.Name() {
			alias = gen.addImport(cmd.StructPath, alias)
		}

		cmd.Struct = alias + "." + cmd.StructName
	}

	return nil
}
