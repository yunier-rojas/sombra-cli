// Command check-usecases verifies that every interactor has a use case contract.
//
// Enforces adr/technical/2026-10-25-interactors-have-use-case-contracts.md.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	defaultDir = "internal/core/usecases"
	adrPath    = "adr/technical/2026-10-25-interactors-have-use-case-contracts.md"
)

type interfaceInfo struct {
	pos     token.Pos
	methods map[string]bool
}

type assertionInfo struct {
	caseName   string
	interactor string
}

type methodInfo struct {
	name string
	pos  token.Pos
}

type checker struct {
	path       string
	fset       *token.FileSet
	interfaces map[string]interfaceInfo
	structs    map[string]token.Pos
	assertions map[string]assertionInfo
	methods    map[string][]methodInfo
}

func main() {
	dir := defaultDir
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Printf("check-usecases: %s not found, skipping\n", dir)
		return
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		fail(err)
	}

	checked := 0
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		checked++
		checkFile(path)
	}

	if checked == 0 {
		fmt.Printf("check-usecases: no Go files in %s, skipping\n", dir)
		return
	}
	if len(allIssues) > 0 {
		sort.Strings(allIssues)
		for _, issue := range allIssues {
			fmt.Println(issue)
		}
		fmt.Fprintf(os.Stderr, "check-usecases: %d issue(s); see %s\n", len(allIssues), adrPath)
		os.Exit(1)
	}
}

var allIssues []string

func checkFile(path string) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		allIssues = append(allIssues, fmt.Sprintf("%s: parse: %v", path, err))
		return
	}

	c := &checker{
		path:       path,
		fset:       fset,
		interfaces: make(map[string]interfaceInfo),
		structs:    make(map[string]token.Pos),
		assertions: make(map[string]assertionInfo),
		methods:    make(map[string][]methodInfo),
	}

	for _, decl := range file.Decls {
		switch node := decl.(type) {
		case *ast.GenDecl:
			c.collectGenDecl(node)
		case *ast.FuncDecl:
			c.collectMethod(node)
		}
	}

	c.check()
}

func (c *checker) collectGenDecl(decl *ast.GenDecl) {
	switch decl.Tok {
	case token.TYPE:
		for _, spec := range decl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			name := typeSpec.Name.Name
			switch value := typeSpec.Type.(type) {
			case *ast.StructType:
				if strings.HasSuffix(name, "Interactor") {
					c.structs[name] = typeSpec.Pos()
				}
			case *ast.InterfaceType:
				methods := make(map[string]bool)
				for _, field := range value.Methods.List {
					for _, method := range field.Names {
						methods[method.Name] = true
					}
				}
				c.interfaces[name] = interfaceInfo{pos: typeSpec.Pos(), methods: methods}
			}
		}
	case token.VAR:
		for _, spec := range decl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok || len(valueSpec.Names) != 1 || valueSpec.Names[0].Name != "_" {
				continue
			}
			caseName, ok := valueSpec.Type.(*ast.Ident)
			if !ok || len(valueSpec.Values) != 1 {
				continue
			}
			interactor, ok := assertionTarget(valueSpec.Values[0])
			if !ok {
				continue
			}
			c.assertions[interactor] = assertionInfo{caseName: caseName.Name, interactor: interactor}
		}
	}
}

func (c *checker) collectMethod(decl *ast.FuncDecl) {
	receiver := receiverName(decl.Recv)
	if receiver == "" || !ast.IsExported(decl.Name.Name) {
		return
	}
	c.methods[receiver] = append(c.methods[receiver], methodInfo{name: decl.Name.Name, pos: decl.Pos()})
}

func (c *checker) check() {
	usedCases := make(map[string]bool)

	for interactor, pos := range c.structs {
		assertion, ok := c.assertions[interactor]
		if !ok {
			c.issue(pos, "%s has no var _ <Case> = (*%s)(nil) assertion", interactor, interactor)
			continue
		}

		iface, ok := c.interfaces[assertion.caseName]
		if !ok {
			c.issue(pos, "%s asserts %s, but that interface is not in the same file", interactor, assertion.caseName)
			continue
		}
		usedCases[assertion.caseName] = true

		if iface.pos >= pos {
			c.issue(pos, "%s interface %s must be declared before the interactor", interactor, assertion.caseName)
		}
		if len(iface.methods) == 0 {
			c.issue(iface.pos, "%s interface %s has no methods", interactor, assertion.caseName)
		}
		for method := range iface.methods {
			if !ast.IsExported(method) {
				c.issue(iface.pos, "%s interface %s declares unexported method %s", interactor, assertion.caseName, method)
			}
		}

		for _, method := range c.methods[interactor] {
			if !iface.methods[method.name] {
				c.issue(method.pos, "%s.%s is exported but not declared on %s", interactor, method.name, assertion.caseName)
			}
		}
	}

	for name, iface := range c.interfaces {
		if strings.HasSuffix(name, "Case") && !usedCases[name] {
			c.issue(iface.pos, "case interface %s has no interactor assertion", name)
		}
	}
}

func (c *checker) issue(pos token.Pos, format string, args ...interface{}) {
	allIssues = append(allIssues, fmt.Sprintf("%s:%d: %s", c.path, c.position(pos).Line, fmt.Sprintf(format, args...)))
}

func (c *checker) position(pos token.Pos) token.Position {
	return c.fset.Position(pos)
}

func assertionTarget(expr ast.Expr) (string, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return "", false
	}
	fun := call.Fun
	if paren, ok := fun.(*ast.ParenExpr); ok {
		fun = paren.X
	}
	star, ok := fun.(*ast.StarExpr)
	if !ok {
		return "", false
	}
	ident, ok := star.X.(*ast.Ident)
	if !ok {
		return "", false
	}
	return ident.Name, true
}

func receiverName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	typ := recv.List[0].Type
	if star, ok := typ.(*ast.StarExpr); ok {
		typ = star.X
	}
	ident, ok := typ.(*ast.Ident)
	if !ok {
		return ""
	}
	return ident.Name
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "check-usecases:", err)
	os.Exit(1)
}
