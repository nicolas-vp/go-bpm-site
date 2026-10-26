package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// fenceRe - блок кода со вставкой из файла: ```go file=examples/orders/x.go decl=TestA,fakeMailer.Send
// Содержимое блока в исходнике игнорируется: код берётся из файла, чтобы страница не расходилась с проверенным примером.
var fenceRe = regexp.MustCompile("(?m)^```(\\w+) file=(\\S+)(?: decl=(\\S+))?\\n(?:(?:.*\\n)*?)```$")

// includeCode подставляет в блоки ```lang file=... содержимое файлов. decl - имена функций, типов и методов
// (Тип.Метод) через запятую: из Go-файла берутся только они, с комментариями, в порядке перечисления.
// Путь - от корня репозитория сайта; go-bpm/... - от корня репозитория go-bpm (каталог над -docs).
func includeCode(src []byte, goBpmRoot string) ([]byte, error) {
	var firstErr error
	out := fenceRe.ReplaceAllFunc(src, func(m []byte) []byte {
		g := fenceRe.FindSubmatch(m)
		lang, file, decls := string(g[1]), string(g[2]), string(g[3])
		path := file
		if rest, ok := strings.CutPrefix(file, "go-bpm/"); ok {
			path = filepath.Join(goBpmRoot, rest)
		}
		code, err := os.ReadFile(path)
		if err == nil && decls != "" {
			code, err = goDecls(file, code, strings.Split(decls, ","))
		}
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return m
		}
		header := ""
		if lang == "go" {
			header = "// " + strings.TrimPrefix(file, "examples/") + "\n"
		}
		return []byte("```" + lang + "\n" + header + strings.TrimRight(string(code), "\n") + "\n```")
	})
	return out, firstErr
}

func goDecls(file string, code []byte, names []string) ([]byte, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, code, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	found := map[string][2]token.Pos{}
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			name := d.Name.Name
			if d.Recv != nil && len(d.Recv.List) == 1 {
				t := d.Recv.List[0].Type
				if star, ok := t.(*ast.StarExpr); ok {
					t = star.X
				}
				if id, ok := t.(*ast.Ident); ok {
					name = id.Name + "." + name
				}
			}
			found[name] = [2]token.Pos{declStart(d.Doc, d.Pos()), d.End()}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					found[ts.Name.Name] = [2]token.Pos{declStart(d.Doc, d.Pos()), d.End()}
				}
			}
		}
	}
	var b bytes.Buffer
	for i, name := range names {
		pos, ok := found[name]
		if !ok {
			return nil, fmt.Errorf("%s: нет объявления %s", file, name)
		}
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.Write(code[fset.Position(pos[0]).Offset:fset.Position(pos[1]).Offset])
	}
	return b.Bytes(), nil
}

func declStart(doc *ast.CommentGroup, pos token.Pos) token.Pos {
	if doc != nil {
		return doc.Pos()
	}
	return pos
}
