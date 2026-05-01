package generator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

func scan(apiDir string) ([]RouteSpec, error) {
	var routes []RouteSpec
	err := filepath.WalkDir(apiDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "frourio.go" {
			return nil
		}

		route, err := parseFrourioFile(apiDir, path)
		if err != nil {
			return err
		}
		routes = append(routes, route)
		return nil
	})
	return routes, err
}

func parseFrourioFile(apiDir, filePath string) (RouteSpec, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return RouteSpec{}, err
	}

	dir := filepath.Dir(filePath)
	rel, err := filepath.Rel(apiDir, dir)
	if err != nil {
		return RouteSpec{}, err
	}
	if rel == "." {
		rel = ""
	}

	route := RouteSpec{
		Dir:          dir,
		RelDir:       filepath.ToSlash(rel),
		PackageName:  file.Name.Name,
		PathOverride: findFrourioPath(file),
	}
	if err := validateRouteDir(route.RelDir, filePath); err != nil {
		return RouteSpec{}, err
	}
	if route.PathOverride != "" {
		if strings.Contains(route.PathOverride, "/") {
			return RouteSpec{}, fmt.Errorf("%s: FrourioPath must not contain /", filePath)
		}
	}

	spec, err := findFrourioSpec(file)
	if err != nil {
		return RouteSpec{}, err
	}

	for _, field := range spec.Fields.List {
		for _, name := range field.Names {
			if name.Name == "Middleware" {
				middleware, err := parseMiddleware(field.Type)
				if err != nil {
					return RouteSpec{}, fmt.Errorf("%s: %w", filePath, err)
				}
				route.Middleware = middleware
				continue
			}
			httpName, ok := httpMethodName(name.Name)
			if !ok {
				continue
			}

			methodStruct, ok := field.Type.(*ast.StructType)
			if !ok {
				return RouteSpec{}, fmt.Errorf("%s: FrourioSpec.%s must be a struct", filePath, name.Name)
			}

			method, err := parseMethod(name.Name, httpName, methodStruct)
			if err != nil {
				return RouteSpec{}, fmt.Errorf("%s: %w", filePath, err)
			}
			route.Methods = append(route.Methods, method)
		}
	}
	hasParam := false
	hasNoParam := false
	methodNames := map[string]bool{}
	for _, method := range route.Methods {
		methodNames[method.Name] = true
		if method.Param == nil {
			hasNoParam = true
		} else {
			hasParam = true
		}
	}
	if hasParam && hasNoParam {
		return RouteSpec{}, fmt.Errorf("%s: all methods in a route directory must either define Param or omit Param", filePath)
	}
	if hasParam {
		if route.RelDir == "" {
			return RouteSpec{}, fmt.Errorf("%s: root route cannot define Param", filePath)
		}
		if route.PathOverride != "" {
			return RouteSpec{}, fmt.Errorf("%s: FrourioPath cannot be used with Param", filePath)
		}
	}
	for name := range route.Middleware.Methods {
		if !methodNames[name] {
			return RouteSpec{}, fmt.Errorf("%s: Middleware.%s requires FrourioSpec.%s", filePath, name, name)
		}
	}

	return route, nil
}

func parseMiddleware(expr ast.Expr) (MiddlewareSpec, error) {
	st, ok := expr.(*ast.StructType)
	if !ok {
		return MiddlewareSpec{}, fmt.Errorf("Middleware must be a struct")
	}
	mw := MiddlewareSpec{Methods: map[string]*MiddlewareItem{}}
	for _, field := range st.Fields.List {
		for _, name := range field.Names {
			item, err := parseMiddlewareItem(name.Name, field.Type)
			if err != nil {
				return MiddlewareSpec{}, err
			}
			if name.Name == "All" {
				mw.All = item
				continue
			}
			if _, ok := httpMethodName(name.Name); ok {
				mw.Methods[name.Name] = item
			}
		}
	}
	return mw, nil
}

func parseMiddlewareItem(name string, expr ast.Expr) (*MiddlewareItem, error) {
	if ident, ok := expr.(*ast.Ident); ok && ident.Name == "bool" {
		return &MiddlewareItem{Name: name}, nil
	}
	st, ok := expr.(*ast.StructType)
	if !ok {
		return nil, fmt.Errorf("Middleware.%s must be bool or a struct", name)
	}
	item := &MiddlewareItem{Name: name}
	for _, field := range st.Fields.List {
		for _, fieldName := range field.Names {
			if fieldName.Name != "Context" {
				continue
			}
			ctx, err := parseStruct(name+"MiddlewareContext", field.Type)
			if err != nil {
				return nil, err
			}
			item.Context = ctx
		}
	}
	return item, nil
}

func validateRouteDir(rel, filePath string) error {
	for _, part := range strings.Split(rel, "/") {
		switch {
		case strings.HasPrefix(part, "~"):
			return fmt.Errorf("%s: ~ path parameter directories are no longer supported; use a regular directory name and FrourioSpec.Param", filePath)
		case strings.HasPrefix(part, "[") || strings.HasSuffix(part, "]"):
			return fmt.Errorf("%s: bracket path parameter directories are not supported; use a regular directory name and FrourioSpec.Param", filePath)
		case strings.HasPrefix(part, "(") && strings.HasSuffix(part, ")"):
			return fmt.Errorf("%s: route group directories are not supported", filePath)
		}
	}
	return nil
}

func findFrourioSpec(file *ast.File) (*ast.StructType, error) {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != "FrourioSpec" {
				continue
			}
			st, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				return nil, fmt.Errorf("FrourioSpec must be a struct")
			}
			return st, nil
		}
	}
	return nil, fmt.Errorf("FrourioSpec not found")
}

func parseMethod(name, httpName string, st *ast.StructType) (MethodSpec, error) {
	method := MethodSpec{Name: name, HTTPName: httpName}
	for _, field := range st.Fields.List {
		for _, fieldName := range field.Names {
			switch fieldName.Name {
			case "URLEncoded":
				if !isBoolField(field.Type) {
					return MethodSpec{}, fmt.Errorf("URLEncoded must be bool")
				}
				if method.Format != "" {
					return MethodSpec{}, fmt.Errorf("URLEncoded and FormData cannot be used together")
				}
				method.Format = "urlencoded"
			case "FormData":
				if !isBoolField(field.Type) {
					return MethodSpec{}, fmt.Errorf("FormData must be bool")
				}
				if method.Format != "" {
					return MethodSpec{}, fmt.Errorf("URLEncoded and FormData cannot be used together")
				}
				method.Format = "formData"
			case "Param":
				param := parseField("Param", field.Type, field.Tag)
				method.Param = &param
			case "Query":
				query, err := parseStruct("Query", field.Type)
				if err != nil {
					return MethodSpec{}, err
				}
				method.Query = query
			case "Body":
				body, err := parseStruct("Body", field.Type)
				if err != nil {
					return MethodSpec{}, err
				}
				method.Body = body
			case "Header", "Headers":
				header, err := parseStruct("Header", field.Type)
				if err != nil {
					return MethodSpec{}, err
				}
				method.Header = header
			case "Res":
				resStruct, ok := field.Type.(*ast.StructType)
				if !ok {
					return MethodSpec{}, fmt.Errorf("Res must be a struct")
				}
				responses, err := parseResponses(resStruct)
				if err != nil {
					return MethodSpec{}, err
				}
				method.Responses = responses
			}
		}
	}
	return method, nil
}

func isBoolField(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "bool"
}

func findFrourioPath(file *ast.File) string {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range valueSpec.Names {
				if name.Name != "FrourioPath" || i >= len(valueSpec.Values) {
					continue
				}
				lit, ok := valueSpec.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				value, err := strconv.Unquote(lit.Value)
				if err == nil {
					return value
				}
			}
		}
	}
	return ""
}

func parseResponses(st *ast.StructType) ([]ResponseSpec, error) {
	var responses []ResponseSpec
	for _, field := range st.Fields.List {
		for _, name := range field.Names {
			status, ok := parseStatusName(name.Name)
			if !ok {
				continue
			}
			resStruct, ok := field.Type.(*ast.StructType)
			if !ok {
				return nil, fmt.Errorf("Res.%s must be a struct", name.Name)
			}
			res := ResponseSpec{Status: status}
			for _, resField := range resStruct.Fields.List {
				for _, resFieldName := range resField.Names {
					switch strings.ToLower(resFieldName.Name) {
					case "body":
						fs := parseField(resFieldName.Name, resField.Type, resField.Tag)
						fs.Name = "Body"
						if fs.JSONName == "" || fs.JSONName == "-" {
							fs.JSONName = "body"
						}
						res.Body = &fs
					case "header", "headers":
						header, err := parseStruct("Header", resField.Type)
						if err != nil {
							return nil, err
						}
						res.Header = header
					}
				}
			}
			responses = append(responses, res)
		}
	}
	return responses, nil
}

func parseStruct(name string, expr ast.Expr) (*StructSpec, error) {
	st, ok := expr.(*ast.StructType)
	if !ok {
		return nil, fmt.Errorf("%s must be an inline struct in the initial implementation", name)
	}
	spec := &StructSpec{Name: name}
	for _, field := range st.Fields.List {
		for _, fieldName := range field.Names {
			spec.Fields = append(spec.Fields, parseField(fieldName.Name, field.Type, field.Tag))
		}
	}
	return spec, nil
}

func parseField(name string, expr ast.Expr, tag *ast.BasicLit) FieldSpec {
	field := FieldSpec{
		Name:       exportName(name),
		SourceName: name,
		Type:       exprString(expr),
		JSONName:   lowerName(name),
	}

	if strings.HasPrefix(field.Type, "*") {
		field.Pointer = true
	}
	if strings.HasPrefix(strings.TrimPrefix(field.Type, "*"), "[]") {
		field.Slice = true
	}

	if tag != nil {
		raw, err := strconv.Unquote(tag.Value)
		if err == nil {
			field.Tag = raw
			st := reflect.StructTag(raw)
			if jsonName := tagName(st.Get("json")); jsonName != "" {
				field.JSONName = jsonName
			}
			field.ValidateTag = st.Get("validate")
		}
	}
	return field
}

func tagName(tag string) string {
	if tag == "" {
		return ""
	}
	return strings.Split(tag, ",")[0]
}

func exprString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprString(t.X)
	case *ast.ArrayType:
		return "[]" + exprString(t.Elt)
	case *ast.SelectorExpr:
		return exprString(t.X) + "." + t.Sel.Name
	default:
		return "any"
	}
}

func httpMethodName(name string) (string, bool) {
	switch strings.ToLower(name) {
	case "get":
		return "GET", true
	case "post":
		return "POST", true
	case "put":
		return "PUT", true
	case "patch":
		return "PATCH", true
	case "delete":
		return "DELETE", true
	case "head":
		return "HEAD", true
	case "options":
		return "OPTIONS", true
	default:
		return "", false
	}
}

var statusRE = regexp.MustCompile(`^Status([1-5][0-9][0-9])$`)

func parseStatusName(name string) (int, bool) {
	matches := statusRE.FindStringSubmatch(name)
	if matches == nil {
		return 0, false
	}
	status, err := strconv.Atoi(matches[1])
	return status, err == nil
}

func routePath(rel string, overrides map[string]string, param *FieldSpec) string {
	if rel == "" {
		return "/api"
	}

	parts := []string{"api"}
	relParts := strings.Split(rel, "/")
	for i, part := range relParts {
		if part == "" {
			continue
		}
		currentRel := strings.Join(relParts[:i+1], "/")
		if param != nil && i == len(relParts)-1 {
			if param != nil && param.Slice {
				parts = append(parts, "{"+part+"...}")
			} else {
				parts = append(parts, "{"+part+"}")
			}
			continue
		}
		if override := overrides[currentRel]; override != "" {
			parts = append(parts, override)
			continue
		}
		parts = append(parts, part)
	}
	return "/" + strings.Join(parts, "/")
}

func pathParamName(rel string) string {
	part := filepath.Base(filepath.FromSlash(rel))
	if part != "." && part != string(filepath.Separator) && part != "" {
		return part
	}
	return "param"
}
