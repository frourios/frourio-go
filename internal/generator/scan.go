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
	docs := newDocResolver(fset, file.Comments)

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
	namedStructs := collectNamedStructs(file, docs)

	for _, field := range spec.Fields.List {
		for _, name := range field.Names {
			if name.Name == "Middleware" {
				middleware, err := parseMiddleware(field.Type, namedStructs, docs)
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

			method, err := parseMethod(name.Name, httpName, methodStruct, namedStructs, docs)
			if err != nil {
				return RouteSpec{}, fmt.Errorf("%s: %w", filePath, err)
			}
			method.Doc = docForNode(docs, field, field.Doc)
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

type namedStructDef struct {
	Struct *ast.StructType
	Doc    DocSpec
}

func collectNamedStructs(file *ast.File, docs *docResolver) map[string]namedStructDef {
	named := map[string]namedStructDef{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name == "FrourioSpec" {
				continue
			}
			if st, ok := typeSpec.Type.(*ast.StructType); ok {
				named[typeSpec.Name.Name] = namedStructDef{Struct: st, Doc: docForNode(docs, typeSpec, docGroup(gen.Doc, typeSpec.Doc))}
			}
		}
	}
	return named
}

func parseMiddleware(expr ast.Expr, namedStructs map[string]namedStructDef, docs *docResolver) (MiddlewareSpec, error) {
	st, ok := expr.(*ast.StructType)
	if !ok {
		return MiddlewareSpec{}, fmt.Errorf("middleware must be a struct")
	}
	mw := MiddlewareSpec{Methods: map[string]*MiddlewareItem{}}
	for _, field := range st.Fields.List {
		for _, name := range field.Names {
			item, err := parseMiddlewareItem(name.Name, field.Type, namedStructs, docs)
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

func parseMiddlewareItem(name string, expr ast.Expr, namedStructs map[string]namedStructDef, docs *docResolver) (*MiddlewareItem, error) {
	if ident, ok := expr.(*ast.Ident); ok && ident.Name == "bool" {
		return &MiddlewareItem{Name: name}, nil
	}
	st, ok := expr.(*ast.StructType)
	if !ok {
		return nil, fmt.Errorf("middleware.%s must be bool or a struct", name)
	}
	item := &MiddlewareItem{Name: name}
	for _, field := range st.Fields.List {
		for _, fieldName := range field.Names {
			if fieldName.Name != "Context" {
				continue
			}
			ctx, err := parseStruct(name+"MiddlewareContext", field.Type, namedStructs, docs)
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

func parseMethod(name, httpName string, st *ast.StructType, namedStructs map[string]namedStructDef, docs *docResolver) (MethodSpec, error) {
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
				param.Doc = docForNode(docs, field, field.Doc)
				method.Param = &param
			case "Query":
				query, err := parseStruct("Query", field.Type, namedStructs, docs)
				if err != nil {
					return MethodSpec{}, err
				}
				query.Doc = docForNode(docs, field, field.Doc)
				method.Query = query
			case "Body":
				body, err := parseStruct("Body", field.Type, namedStructs, docs)
				if err != nil {
					return MethodSpec{}, err
				}
				body.Doc = docForNode(docs, field, field.Doc)
				method.Body = body
			case "Header", "Headers":
				header, err := parseStruct("Header", field.Type, namedStructs, docs)
				if err != nil {
					return MethodSpec{}, err
				}
				header.Doc = docForNode(docs, field, field.Doc)
				method.Header = header
			case "Res":
				resStruct, ok := field.Type.(*ast.StructType)
				if !ok {
					return MethodSpec{}, fmt.Errorf("res must be a struct")
				}
				responses, err := parseResponses(resStruct, namedStructs, docs)
				if err != nil {
					return MethodSpec{}, err
				}
				method.Responses = responses
			}
		}
	}
	method.Raw = len(method.Responses) == 0
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

func parseResponses(st *ast.StructType, namedStructs map[string]namedStructDef, docs *docResolver) ([]ResponseSpec, error) {
	var responses []ResponseSpec
	for _, field := range st.Fields.List {
		for _, name := range field.Names {
			status, ok := parseStatusName(name.Name)
			if !ok {
				continue
			}
			resStruct, ok := field.Type.(*ast.StructType)
			if !ok {
				return nil, fmt.Errorf("res.%s must be a struct", name.Name)
			}
			res := ResponseSpec{Status: status, Doc: docForNode(docs, field, field.Doc)}
			for _, resField := range resStruct.Fields.List {
				for _, resFieldName := range resField.Names {
					switch strings.ToLower(resFieldName.Name) {
					case "formdata":
						if !isBoolField(resField.Type) {
							return nil, fmt.Errorf("formData must be bool")
						}
						res.FormData = true
					case "body":
						if isStructLike(resField.Type, namedStructs) {
							body, err := parseStruct("Body", resField.Type, namedStructs, docs)
							if err != nil {
								return nil, err
							}
							res.BodyStruct = body
						}
						fs := parseField(resFieldName.Name, resField.Type, resField.Tag)
						fs.Name = "Body"
						fs.Doc = docForNode(docs, resField, resField.Doc)
						if fs.JSONName == "" || fs.JSONName == "-" {
							fs.JSONName = "body"
						}
						res.Body = &fs
					case "header", "headers":
						header, err := parseStruct("Header", resField.Type, namedStructs, docs)
						if err != nil {
							return nil, err
						}
						header.Doc = docForNode(docs, resField, resField.Doc)
						res.Header = header
					}
				}
			}
			responses = append(responses, res)
		}
	}
	return responses, nil
}

func parseStruct(name string, expr ast.Expr, namedStructs map[string]namedStructDef, docs *docResolver) (*StructSpec, error) {
	st, ok := expr.(*ast.StructType)
	typeName := name
	inline := true
	doc := DocSpec{}
	if !ok {
		ident, identOK := expr.(*ast.Ident)
		if !identOK {
			return nil, fmt.Errorf("%s must be a struct", name)
		}
		def, ok := namedStructs[ident.Name]
		if !ok {
			return nil, fmt.Errorf("%s named struct %s not found", name, ident.Name)
		}
		st = def.Struct
		doc = def.Doc
		typeName = ident.Name
		inline = false
	}
	spec := &StructSpec{Name: name, TypeName: typeName, Inline: inline, Doc: doc}
	for _, field := range st.Fields.List {
		for _, fieldName := range field.Names {
			fs := parseField(fieldName.Name, field.Type, field.Tag)
			fs.Doc = docForNode(docs, field, field.Doc)
			spec.Fields = append(spec.Fields, fs)
		}
	}
	return spec, nil
}

func isStructLike(expr ast.Expr, namedStructs map[string]namedStructDef) bool {
	if _, ok := expr.(*ast.StructType); ok {
		return true
	}
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	_, ok = namedStructs[ident.Name]
	return ok
}

func parseField(name string, expr ast.Expr, tag *ast.BasicLit) FieldSpec {
	field := FieldSpec{
		Name:       exportName(name),
		SourceName: name,
		Type:       exprString(expr),
		JSONName:   name,
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
				field.JSONTagged = true
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

type docResolver struct {
	fset     *token.FileSet
	comments []*ast.CommentGroup
}

func newDocResolver(fset *token.FileSet, comments []*ast.CommentGroup) *docResolver {
	return &docResolver{fset: fset, comments: comments}
}

func (r *docResolver) docFor(node ast.Node, group *ast.CommentGroup) DocSpec {
	if r == nil {
		return parseDoc(group)
	}
	groups := r.commentGroupsFor(node, group)
	return docFromGroups(groups...)
}

func docForNode(resolver *docResolver, node ast.Node, group *ast.CommentGroup) DocSpec {
	if resolver == nil {
		return parseDoc(group)
	}
	return resolver.docFor(node, group)
}

func (r *docResolver) commentGroupsFor(node ast.Node, group *ast.CommentGroup) []*ast.CommentGroup {
	nodeLine := r.fset.Position(node.Pos()).Line
	index := -1
	for i, candidate := range r.comments {
		if r.fset.Position(candidate.End()).Line == nodeLine-1 {
			index = i
		}
	}
	if index < 0 {
		if group == nil {
			group = r.directCommentGroup(node.Pos())
		}
		if group == nil {
			return nil
		}
		for i, candidate := range r.comments {
			if candidate == group || (candidate.Pos() == group.Pos() && candidate.End() == group.End()) {
				index = i
				break
			}
		}
		if index < 0 {
			return []*ast.CommentGroup{group}
		}
	}
	start := index
	for start > 0 && r.adjacent(r.comments[start-1], r.comments[start]) {
		start--
	}
	return r.comments[start : index+1]
}

func (r *docResolver) directCommentGroup(pos token.Pos) *ast.CommentGroup {
	nodeLine := r.fset.Position(pos).Line
	for _, group := range r.comments {
		if r.fset.Position(group.End()).Line == nodeLine-1 {
			return group
		}
	}
	return nil
}

func (r *docResolver) adjacent(prev, next *ast.CommentGroup) bool {
	return r.fset.Position(prev.End()).Line+1 == r.fset.Position(next.Pos()).Line
}

func docGroup(groups ...*ast.CommentGroup) *ast.CommentGroup {
	for i := len(groups) - 1; i >= 0; i-- {
		if groups[i] != nil {
			return groups[i]
		}
	}
	return nil
}

func docFromGroups(groups ...*ast.CommentGroup) DocSpec {
	doc := DocSpec{}
	for _, group := range groups {
		next := parseDoc(group)
		if doc.Summary == "" {
			doc.Summary = next.Summary
		} else if next.Summary != "" {
			doc.Summary += " " + next.Summary
		}
		if doc.Description == "" {
			doc.Description = next.Description
		} else if next.Description != "" {
			doc.Description += "\n\n" + next.Description
		}
	}
	return doc
}

func parseDoc(group *ast.CommentGroup) DocSpec {
	if group == nil {
		return DocSpec{}
	}
	summary := []string{}
	description := []string{}
	for _, comment := range group.List {
		text := comment.Text
		if strings.HasPrefix(text, "//") {
			line := strings.TrimSpace(strings.TrimPrefix(text, "//"))
			if line != "" {
				summary = append(summary, line)
			}
			continue
		}
		if strings.HasPrefix(text, "/*") {
			block := strings.TrimSuffix(strings.TrimPrefix(text, "/*"), "*/")
			block = trimBlockCommentIndent(block)
			if block != "" {
				description = append(description, block)
			}
		}
	}
	return DocSpec{
		Summary:     strings.Join(summary, " "),
		Description: strings.Join(description, "\n\n"),
	}
}

func trimBlockCommentIndent(text string) string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := leadingIndentWidth(line)
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent > 0 {
		for i, line := range lines {
			lines[i] = trimIndentWidth(line, minIndent)
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func leadingIndentWidth(line string) int {
	width := 0
	for _, r := range line {
		switch r {
		case '\t':
			width += 8
		case ' ':
			width++
		default:
			return width
		}
	}
	return width
}

func trimIndentWidth(line string, width int) string {
	removed := 0
	for i, r := range line {
		next := removed
		switch r {
		case '\t':
			next += 8
		case ' ':
			next++
		default:
			return line[i:]
		}
		if next > width {
			return line[i:]
		}
		removed = next
		if removed == width {
			return line[i+len(string(r)):]
		}
	}
	return ""
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
			if param.Slice {
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
