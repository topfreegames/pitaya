// Copyright (c) TFG Co. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package docgenerator

import (
	"encoding/json"
	"reflect"
	"strings"
	"unicode"

	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/route"
)

type docs struct {
	Handlers docMap `json:"handlers"`
	Remotes  docMap `json:"remotes"`
}

type docMap map[string]*doc

type doc struct {
	Input  interface{}   `json:"input"`
	Output []interface{} `json:"output"`
}

// HandlersDocs returns a map from route to input and output
func HandlersDocs(serverType string, services map[string]*component.Service, getPtrNames bool) (map[string]interface{}, error) {
	docs := &docs{
		Handlers: map[string]*doc{},
	}

	for serviceName, service := range services {
		for name, handler := range service.Handlers {
			routeName := route.NewRoute(serverType, serviceName, name)
			docs.Handlers[routeName.String()] = docForMethod(handler.Method, getPtrNames)
		}
	}

	return docs.Handlers.toMap()
}

// RemotesDocs returns a map from route to input and output
func RemotesDocs(serverType string, services map[string]*component.Service, getPtrNames bool) (map[string]interface{}, error) {
	docs := &docs{
		Remotes: map[string]*doc{},
	}

	for serviceName, service := range services {
		for name, remote := range service.Remotes {
			routeName := route.NewRoute(serverType, serviceName, name)
			docs.Remotes[routeName.String()] = docForMethod(remote.Method, getPtrNames)
		}
	}

	return docs.Remotes.toMap()
}

func (d docMap) toMap() (map[string]interface{}, error) {
	var m map[string]interface{}
	bts, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bts, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func docForMethod(method reflect.Method, getPtrNames bool) *doc {
	doc := &doc{
		Output: []interface{}{},
	}

	if method.Type.NumIn() > 2 {
		isOutput := false
		doc.Input = docForType(method.Type.In(2), isOutput, getPtrNames)
	}

	for i := 0; i < method.Type.NumOut(); i++ {
		isOutput := true
		doc.Output = append(doc.Output, docForType(method.Type.Out(i), isOutput, getPtrNames))
	}

	return doc
}

func parseStruct(typ reflect.Type) reflect.Type {
	switch typ.String() {
	case "time.Time":
		return nil
	default:
		return typ
	}
}

func docForType(typ reflect.Type, isOutput bool, getPtrNames bool) interface{} {
	if typ.Kind() == reflect.Ptr {
		fields := map[string]interface{}{}
		elm := typ.Elem()
		for i := 0; i < elm.NumField(); i++ {
			if name, valid := getName(elm.Field(i), isOutput); valid {
				fields[name] = parseType(elm.Field(i).Type, isOutput, getPtrNames)
			}
		}
		if getPtrNames {
			composite := map[string]interface{}{}
			composite[typ.String()] = fields
			return composite
		}
		return fields
	}

	return parseType(typ, isOutput, getPtrNames)
}

func validName(field reflect.StructField) bool {
	isProtoField := func(name string) bool {
		return strings.HasPrefix(name, "XXX_")
	}

	isPrivateField := func(name string) bool {
		for _, r := range name {
			return unicode.IsLower(r)
		}

		return true
	}

	isIgnored := func(field reflect.StructField) bool {
		return field.Tag.Get("json") == "-"
	}

	return !isProtoField(field.Name) && !isPrivateField(field.Name) && !isIgnored(field)
}

func firstLetterToLower(name string, isOutput bool) string {
	if isOutput {
		return name
	}

	return string(append([]byte{strings.ToLower(name)[0]}, name[1:]...))
}

func getName(field reflect.StructField, isOutput bool) (name string, valid bool) {
	if !validName(field) {
		return "", false
	}

	name, ok := field.Tag.Lookup("json")
	if !ok {
		return firstLetterToLower(field.Name, isOutput), true
	}

	return strings.Split(name, ",")[0], true
}

func parseType(typ reflect.Type, isOutput bool, getPtrNames bool) interface{} {
	var elm reflect.Type

	switch typ.Kind() {
	case reflect.Ptr:
		elm = typ.Elem()
	case reflect.Struct:
		elm = parseStruct(typ)
		if elm == nil {
			return typ.String()
		}
	case reflect.Slice:
		parsed := parseType(typ.Elem(), isOutput, getPtrNames)
		if parsed == "uint8" {
			return "[]byte"
		}
		return []interface{}{parsed}
	default:
		return typ.String()
	}

	fields := map[string]interface{}{}
	for i := 0; i < elm.NumField(); i++ {
		if name, valid := getName(elm.Field(i), isOutput); valid {
			fields[name] = parseType(elm.Field(i).Type, isOutput, getPtrNames)
		}
	}
	if getPtrNames {
		composite := map[string]interface{}{}
		composite[typ.String()] = fields
		return composite
	}
	return fields
}
