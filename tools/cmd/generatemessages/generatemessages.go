package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

var (
	tmpl = template.Must(template.New("asd").Funcs(template.FuncMap{
		"hasPreHandler":  hasPreHandler,
		"hasPostHandler": hasPostHandler,
	}).Parse(`// Auto-generated xd
package twitch

type EventHandler struct {
{{ range . }}
	on{{.}} []func(message {{.}})
{{end}}
}

func (e *Client) handleMessage(message Message) (err error) {
	switch msg := message.(type) {
{{ range . }}
	case *{{.}}:
		{{ if hasPreHandler . }}
		if !e.handle{{.}}(*msg) {
			return nil
		}
		{{ end }}
		for _, cb := range e.on{{.}} {
			cb(*msg)
		}

		{{ if hasPostHandler . }}
		err = e.handle{{.}}(*msg)
		{{ end }}
{{ end }}
	}

	return
}

{{ range . }}
// On{{ . }} test
func (e *EventHandler) On{{.}}(cb func({{ . }})) {
	e.on{{.}} = append(e.on{{.}}, cb)
}
{{end}}
	`))
)

func readMessageTypes(path string) []string {
	fs := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fs, path, nil, 0)
	if err != nil {
		log.Fatalf("warning: internal error: could not parse events.go: %s", err)
		return nil
	}

	names := []string{}
	for object, _ := range parsedFile.Scope.Objects {
		// fmt.Println(object, xd.Pos(), xd.Decl)
		names = append(names, object)
	}

	sort.Strings(names)

	return names
}

var preHandlers []string
var postHandlers []string

func readPreHandlers(path string) []string {
	fs := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fs, path, nil, 0)
	if err != nil {
		log.Fatalf("warning: internal error: could not parse events.go: %s", err)
		return nil
	}

	for _, decl := range parsedFile.Decls {
		if xd, ok := decl.(*ast.FuncDecl); ok {
			if strings.HasPrefix(xd.Name.String(), "handle") && strings.HasSuffix(xd.Name.String(), "Message") {
				if xd.Type.Results == nil {
					continue
				}
				if xd.Type.Params == nil {
					continue
				}
				fmt.Println("got function", xd.Name)
				if len(xd.Type.Params.List) != 1 {
					continue
				}

				if len(xd.Type.Results.List) != 1 {
					continue
				}

				parameter := xd.Type.Params.List[0]
				returnValue := xd.Type.Results.List[0]

				fmt.Println(returnValue)
				fmt.Println(returnValue.Tag)
				typeName := returnValue.Type.(*ast.Ident).Name
				parameterName := parameter.Type.(*ast.Ident).Name
				// fmt.Println(string(returnValue.Type))
				fmt.Println(returnValue.Names)
				fmt.Println(parameter.Names)
				switch typeName {
				case "bool":
					preHandlers = append(preHandlers, parameterName)
				case "error":
					postHandlers = append(postHandlers, parameterName)
				}

				fmt.Println(preHandlers)
				fmt.Println(postHandlers)
			}
		}
	}
	for name, object := range parsedFile.Scope.Objects {
		// fmt.Println(name, object.Kind.String())
		if name == "Client" {
			fmt.Println("Data:", object.Data)
			fmt.Println("Type:", object.Type)

			fmt.Println(object, object.Data, object.Type)
			xd := object.Decl.(*ast.TypeSpec)
			fmt.Println("found client", xd)
			xd2 := xd.Type.(*ast.StructType)
			fmt.Println(xd2.Incomplete)
			fmt.Println("found client", xd2)
			for _, field := range xd2.Fields.List {
				fmt.Println("Field:", field.Type)
			}
		}
	}

	return nil
}

func main() {
	var buf bytes.Buffer
	dir := filepath.Dir("../../../")

	names := readMessageTypes("../../../messages.go")
	preHandlers := readPreHandlers("../../../client.go")
	fmt.Println(preHandlers)

	tmpl.Execute(&buf, names)

	src, err := format.Source(buf.Bytes())
	if err != nil {
		log.Println("warning: internal error: invalid Go generated:", err)
		src = buf.Bytes()
	}

	err = ioutil.WriteFile(filepath.Join(dir, strings.ToLower("client_callbacks.go")), src, 0644)
	if err != nil {
		log.Fatal(buf, "writing output: %s", err)
	}
}

func hasPreHandler(name string) bool {
	for _, xd := range preHandlers {
		if xd == name {
			return true
		}
	}
	return false
}

func hasPostHandler(name string) bool {
	for _, xd := range postHandlers {
		if xd == name {
			return true
		}
	}
	return false
}
