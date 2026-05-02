package generator

import (
	"fmt"
	"strings"
)

func serverText(apiDir string, routes []RouteSpec) string {
	packageName := rootPackage(routes)
	var b strings.Builder
	fmt.Fprintf(&b, "package %s\n\n", packageName)
	b.WriteString("import (\n")
	b.WriteString("\t\"context\"\n")
	b.WriteString("\t\"encoding/json\"\n")
	b.WriteString("\t\"fmt\"\n")
	b.WriteString("\t\"mime/multipart\"\n")
	b.WriteString("\t\"net/http\"\n")
	b.WriteString("\t\"reflect\"\n")
	b.WriteString("\t\"strconv\"\n")
	b.WriteString("\t\"strings\"\n")
	b.WriteString("\n")
	b.WriteString("\t\"github.com/go-playground/validator/v10\"\n")
	for _, route := range routes {
		if route.RelDir == "" {
			continue
		}
		fmt.Fprintf(&b, "\t%s %q\n", route.Alias, route.ImportPath)
	}
	b.WriteString(")\n\n")
	b.WriteString("var frourioValidate = validator.New()\n\n")
	b.WriteString("type Option func(*serverOptions)\n\n")
	b.WriteString("type serverOptions struct{}\n\n")
	b.WriteString("func Mount(mux *http.ServeMux, options ...Option) {\n")
	for _, route := range routes {
		for _, method := range route.Methods {
			if optionalPath, ok := optionalCatchAllPath(method); ok {
				fmt.Fprintf(&b, "\tmux.Handle(\"%s %s\", %s)\n", method.HTTPName, optionalPath, wrapperCall(route, method))
			}
			fmt.Fprintf(&b, "\tmux.Handle(\"%s %s\", %s)\n", method.HTTPName, method.URLPath, wrapperCall(route, method))
		}
	}
	b.WriteString("}\n\n")
	b.WriteString("func Handler(options ...Option) http.Handler {\n")
	b.WriteString("\tmux := http.NewServeMux()\n")
	b.WriteString("\tMount(mux, options...)\n")
	b.WriteString("\treturn mux\n")
	b.WriteString("}\n\n")

	for _, route := range routes {
		for _, method := range route.Methods {
			writeWrapper(&b, route, method)
		}
	}

	b.WriteString(runtimeText())
	return b.String()
}

func rootPackage(routes []RouteSpec) string {
	for _, route := range routes {
		if route.RelDir == "" {
			return route.PackageName
		}
	}
	return routes[0].PackageName
}

func routeQualifier(route RouteSpec) string {
	if route.RelDir == "" {
		return ""
	}
	return route.Alias + "."
}

func wrapperName(route RouteSpec, method MethodSpec) string {
	if route.RelDir == "" {
		return "wrap" + method.Name
	}
	return "wrap" + exportName(route.Alias) + method.Name
}

func decodeName(route RouteSpec, method MethodSpec, part string) string {
	if route.RelDir == "" {
		return "decode" + method.Name + part
	}
	return "decode" + exportName(route.Alias) + method.Name + part
}

func wrapperCall(route RouteSpec, method MethodSpec) string {
	return fmt.Sprintf("%s(%sRoute)", wrapperName(route, method), routeQualifier(route))
}

func writeWrapper(b *strings.Builder, route RouteSpec, method MethodSpec) {
	q := routeQualifier(route)
	fmt.Fprintf(b, "func %s(route %sRouteDefinition) http.Handler {\n", wrapperName(route, method), q)
	b.WriteString("\treturn http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {\n")
	fmt.Fprintf(b, "\t\thandlers := route.Handlers()\n")
	fmt.Fprintf(b, "\t\tif handlers.%s == nil {\n", method.Name)
	b.WriteString("\t\t\thttp.NotFound(w, r)\n")
	b.WriteString("\t\t\treturn\n")
	b.WriteString("\t\t}\n\n")
	fmt.Fprintf(b, "\t\treq := %s%sRequest{}\n", q, method.Name)
	if len(route.ParamAncestors) > 0 {
		fmt.Fprintf(b, "\t\tparams, paramErr := %s(r)\n", decodeName(route, method, "Params"))
		b.WriteString("\t\tif paramErr != nil {\n")
		b.WriteString("\t\t\twriteRequestError(w, []frourioIssue{{Path: []any{\"params\", paramErr.Field}, Message: paramErr.Message}})\n")
		b.WriteString("\t\t\treturn\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\treq.Params = params\n")
		hasValidate := false
		for _, ancestor := range route.ParamAncestors {
			if ancestor.Param != nil && ancestor.Param.ValidateTag != "" {
				hasValidate = true
				break
			}
		}
		if hasValidate {
			b.WriteString("\t\tif err := frourioValidate.Struct(req.Params); err != nil {\n")
			b.WriteString("\t\t\twriteValidationError(w, err, \"params\")\n")
			b.WriteString("\t\t\treturn\n")
			b.WriteString("\t\t}\n")
		}
		b.WriteString("\n")
	}
	if method.Query != nil {
		fmt.Fprintf(b, "\t\tquery, err := %s(r)\n", decodeName(route, method, "Query"))
		b.WriteString("\t\tif err != nil {\n")
		b.WriteString("\t\t\twriteRequestError(w, []frourioIssue{{Path: []any{\"query\"}, Message: err.Error()}})\n")
		b.WriteString("\t\t\treturn\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\treq.Query = query\n")
		b.WriteString("\t\tif err := frourioValidate.Struct(req.Query); err != nil {\n")
		b.WriteString("\t\t\twriteValidationError(w, err, \"query\")\n")
		b.WriteString("\t\t\treturn\n")
		b.WriteString("\t\t}\n\n")
	}
	if method.Body != nil {
		fmt.Fprintf(b, "\t\tbody, err := %s(r)\n", decodeName(route, method, "Body"))
		b.WriteString("\t\tif err != nil {\n")
		b.WriteString("\t\t\twriteRequestError(w, []frourioIssue{{Path: []any{\"body\"}, Message: err.Error()}})\n")
		b.WriteString("\t\t\treturn\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\treq.Body = body\n")
		b.WriteString("\t\tif err := frourioValidate.Struct(req.Body); err != nil {\n")
		b.WriteString("\t\t\twriteValidationError(w, err, \"body\")\n")
		b.WriteString("\t\t\treturn\n")
		b.WriteString("\t\t}\n\n")
	}
	if methodHasMiddleware(route, method) {
		writeMiddlewareCall(b, route, method)
	} else {
		fmt.Fprintf(b, "\t\tres, err := handlers.%s(r.Context(), req)\n", method.Name)
	}
	b.WriteString("\t\tif err != nil {\n")
	b.WriteString("\t\t\twriteInternalError(w)\n")
	b.WriteString("\t\t\treturn\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\tif res == nil {\n")
	b.WriteString("\t\t\twriteInternalError(w)\n")
	b.WriteString("\t\t\treturn\n")
	b.WriteString("\t\t}\n\n")
	if method.Raw {
		b.WriteString("\t\tif err := res.WriteHTTP(w, r); err != nil {\n")
		b.WriteString("\t\t\twriteInternalError(w)\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t})\n")
		b.WriteString("}\n\n")
		return
	}
	b.WriteString("\t\tswitch v := res.(type) {\n")
	for _, response := range method.Responses {
		fmt.Fprintf(b, "\t\tcase %s%sStatus%d:\n", q, method.Name, response.Status)
		if response.Header != nil {
			writeResponseHeaders(b, response)
		}
		if response.Body != nil {
			b.WriteString("\t\t\tif err := frourioValidate.Struct(v); err != nil {\n")
			b.WriteString("\t\t\t\twriteInternalError(w)\n")
			b.WriteString("\t\t\t\treturn\n")
			b.WriteString("\t\t\t}\n")
			writeResponse(b, response)
		} else {
			b.WriteString("\t\t\tw.WriteHeader(v.StatusCode())\n")
		}
	}
	b.WriteString("\t\tdefault:\n")
	b.WriteString("\t\t\twriteInternalError(w)\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t})\n")
	b.WriteString("}\n\n")

	if len(route.ParamAncestors) > 0 {
		paramsType := q + "Params"
		fmt.Fprintf(b, "func %s(r *http.Request) (%s, *frourioParamDecodeError) {\n", decodeName(route, method, "Params"), paramsType)
		fmt.Fprintf(b, "\tvar params %s\n", paramsType)
		writeParamsDecode(b, route)
		b.WriteString("\treturn params, nil\n")
		b.WriteString("}\n\n")
	}
	if method.Query != nil {
		queryType := requestDecodeType(q, method, "Query", method.Query)
		fmt.Fprintf(b, "func %s(r *http.Request) (%s, error) {\n", decodeName(route, method, "Query"), queryType)
		fmt.Fprintf(b, "\tvar query %s\n", queryType)
		b.WriteString("\tvalues := r.URL.Query()\n")
		for _, field := range method.Query.Fields {
			writeQueryDecode(b, field)
		}
		b.WriteString("\treturn query, nil\n")
		b.WriteString("}\n\n")
	}
	if method.Body != nil {
		bodyType := requestDecodeType(q, method, "Body", method.Body)
		fmt.Fprintf(b, "func %s(r *http.Request) (%s, error) {\n", decodeName(route, method, "Body"), bodyType)
		fmt.Fprintf(b, "\tvar body %s\n", bodyType)
		switch method.Format {
		case "urlencoded":
			b.WriteString("\tif err := r.ParseForm(); err != nil {\n")
			b.WriteString("\t\treturn body, err\n")
			b.WriteString("\t}\n")
			b.WriteString("\tvalues := r.PostForm\n")
			for _, field := range method.Body.Fields {
				writeValuesDecode(b, "body", field)
			}
		case "formData":
			b.WriteString("\tif err := r.ParseMultipartForm(32 << 20); err != nil {\n")
			b.WriteString("\t\treturn body, err\n")
			b.WriteString("\t}\n")
			b.WriteString("\tvalues := r.MultipartForm.Value\n")
			for _, field := range method.Body.Fields {
				writeValuesDecode(b, "body", field)
			}
		default:
			b.WriteString("\tif err := json.NewDecoder(r.Body).Decode(&body); err != nil {\n")
			b.WriteString("\t\treturn body, err\n")
			b.WriteString("\t}\n")
		}
		b.WriteString("\treturn body, nil\n")
		b.WriteString("}\n\n")
	}
}

func requestDecodeType(q string, method MethodSpec, part string, st *StructSpec) string {
	if st != nil && !st.Inline && st.TypeName != "" {
		return q + st.TypeName
	}
	return q + method.Name + part
}

func writeResponse(b *strings.Builder, response ResponseSpec) {
	autoContentType := "true"
	if hasContentTypeHeader(response) {
		autoContentType = "false"
	}
	if response.FormData {
		fmt.Fprintf(b, "\t\t\twriteMultipart(w, v.StatusCode(), v.Body, %s)\n", autoContentType)
		return
	}
	if response.BodyStruct != nil {
		fmt.Fprintf(b, "\t\t\twriteJSON(w, v.StatusCode(), v.Body, %s)\n", autoContentType)
		return
	}
	switch strings.TrimPrefix(response.Body.Type, "*") {
	case "string":
		fmt.Fprintf(b, "\t\t\twriteText(w, v.StatusCode(), v.Body, %s)\n", autoContentType)
	case "[]byte":
		fmt.Fprintf(b, "\t\t\twriteBytes(w, v.StatusCode(), v.Body, %s)\n", autoContentType)
	default:
		fmt.Fprintf(b, "\t\t\twriteJSON(w, v.StatusCode(), v.Body, %s)\n", autoContentType)
	}
}

func writeResponseHeaders(b *strings.Builder, response ResponseSpec) {
	for _, field := range response.Header.Fields {
		key := field.JSONName
		if key == "" {
			key = field.SourceName
		}
		if !field.JSONTagged && field.SourceName == "ContentType" {
			key = "content-type"
		}
		fmt.Fprintf(b, "\t\t\tw.Header().Set(%q, fmt.Sprint(v.Header.%s))\n", key, field.Name)
	}
}

func hasContentTypeHeader(response ResponseSpec) bool {
	if response.Header == nil {
		return false
	}
	for _, field := range response.Header.Fields {
		if strings.EqualFold(field.JSONName, "content-type") || strings.EqualFold(field.SourceName, "ContentType") {
			return true
		}
	}
	return false
}

func writeMiddlewareCall(b *strings.Builder, route RouteSpec, method MethodSpec) {
	q := routeQualifier(route)
	item := methodMiddleware(route, method)
	if hasMiddleware(route) {
		b.WriteString("\t\tmiddleware := handlers.Middleware\n")
	}
	for i, ancestor := range route.Ancestors {
		if ancestor.Middleware.All != nil {
			fmt.Fprintf(b, "\t\tancestorHandlers%d := %sRoute.Handlers()\n", i, routeQualifier(ancestor))
		}
	}
	fmt.Fprintf(b, "\t\trunHandler := func(ctx context.Context, methodState %s%sContext) (%s%sResponse, error) {\n", q, method.Name, q, method.Name)
	fmt.Fprintf(b, "\t\t\treturn handlers.%s(ctx, req, methodState)\n", method.Name)
	b.WriteString("\t\t}\n")
	fmt.Fprintf(b, "\t\trunMethod := func(ctx context.Context, mwState %sMiddlewareContext) (%s%sResponse, error) {\n", q, q, method.Name)
	if item != nil {
		fmt.Fprintf(b, "\t\t\tif middleware.%s != nil {\n", method.Name)
		if item.Context != nil {
			fmt.Fprintf(b, "\t\t\t\treturn middleware.%s(ctx, r, req, mwState, func(ctx context.Context, req %s%sRequest, methodCtx %s%sMiddlewareContext) (%s%sResponse, error) {\n", method.Name, q, method.Name, q, method.Name, q, method.Name)
			b.WriteString("\t\t\t\t\tif err := frourioValidate.Struct(methodCtx); err != nil {\n")
			b.WriteString("\t\t\t\t\t\treturn nil, err\n")
			b.WriteString("\t\t\t\t\t}\n")
			fmt.Fprintf(b, "\t\t\t\t\treturn runHandler(ctx, %s%sContext{MiddlewareContext: mwState, %sMiddlewareContext: methodCtx})\n", q, method.Name, method.Name)
			b.WriteString("\t\t\t\t})\n")
		} else {
			fmt.Fprintf(b, "\t\t\t\treturn middleware.%s(ctx, r, req, mwState, func(ctx context.Context, req %s%sRequest) (%s%sResponse, error) {\n", method.Name, q, method.Name, q, method.Name)
			fmt.Fprintf(b, "\t\t\t\t\treturn runHandler(ctx, %s%sContext{MiddlewareContext: mwState})\n", q, method.Name)
			b.WriteString("\t\t\t\t})\n")
		}
		b.WriteString("\t\t\t}\n")
	}
	fmt.Fprintf(b, "\t\t\treturn runHandler(ctx, %s%sContext{MiddlewareContext: mwState})\n", q, method.Name)
	b.WriteString("\t\t}\n")

	nextName := "runMethod"
	if route.Middleware.All != nil {
		fmt.Fprintf(b, "\t\trunCurrentAll := func(ctx context.Context, mwState %sMiddlewareContext) (%s%sResponse, error) {\n", q, q, method.Name)
		b.WriteString("\t\t\tif middleware.All != nil {\n")
		if route.Middleware.All.Context != nil {
			b.WriteString("\t\t\t\tres, err := middleware.All(ctx, r, func(ctx context.Context, allCtx " + q + "MiddlewareAllContext) (any, error) {\n")
			b.WriteString("\t\t\t\t\tif err := frourioValidate.Struct(allCtx); err != nil {\n")
			b.WriteString("\t\t\t\t\t\treturn nil, err\n")
			b.WriteString("\t\t\t\t\t}\n")
			b.WriteString("\t\t\t\t\tmwState.MiddlewareAllContext = allCtx\n")
			b.WriteString("\t\t\t\t\treturn runMethod(ctx, mwState)\n")
			b.WriteString("\t\t\t\t})\n")
		} else {
			b.WriteString("\t\t\t\tres, err := middleware.All(ctx, r, func(ctx context.Context) (any, error) {\n")
			b.WriteString("\t\t\t\t\treturn runMethod(ctx, mwState)\n")
			b.WriteString("\t\t\t\t})\n")
		}
		b.WriteString("\t\t\t\tif err != nil {\n")
		b.WriteString("\t\t\t\t\treturn nil, err\n")
		b.WriteString("\t\t\t\t}\n")
		fmt.Fprintf(b, "\t\t\t\ttyped, ok := res.(%s%sResponse)\n", q, method.Name)
		b.WriteString("\t\t\t\tif !ok {\n")
		b.WriteString("\t\t\t\t\treturn nil, fmt.Errorf(\"middleware returned invalid response\")\n")
		b.WriteString("\t\t\t\t}\n")
		b.WriteString("\t\t\t\treturn typed, nil\n")
		b.WriteString("\t\t\t}\n")
		b.WriteString("\t\t\treturn runMethod(ctx, mwState)\n")
		b.WriteString("\t\t}\n")
		nextName = "runCurrentAll"
	}
	for i := len(route.Ancestors) - 1; i >= 0; i-- {
		ancestor := route.Ancestors[i]
		if ancestor.Middleware.All == nil {
			continue
		}
		wrapperName := fmt.Sprintf("runAncestorAll%d", i)
		aq := routeQualifier(ancestor)
		fmt.Fprintf(b, "\t\t%s := func(ctx context.Context, mwState %sMiddlewareContext) (%s%sResponse, error) {\n", wrapperName, q, q, method.Name)
		fmt.Fprintf(b, "\t\t\tif ancestorHandlers%d.Middleware.All != nil {\n", i)
		if ancestor.Middleware.All.Context != nil {
			fmt.Fprintf(b, "\t\t\t\tres, err := ancestorHandlers%d.Middleware.All(ctx, r, func(ctx context.Context, allCtx %sMiddlewareAllContext) (any, error) {\n", i, aq)
			b.WriteString("\t\t\t\t\tif err := frourioValidate.Struct(allCtx); err != nil {\n")
			b.WriteString("\t\t\t\t\t\treturn nil, err\n")
			b.WriteString("\t\t\t\t\t}\n")
			writeCopyMiddlewareFields(b, ancestor.Middleware.All.Context.Fields)
			fmt.Fprintf(b, "\t\t\t\t\treturn %s(ctx, mwState)\n", nextName)
			b.WriteString("\t\t\t\t})\n")
		} else {
			fmt.Fprintf(b, "\t\t\t\tres, err := ancestorHandlers%d.Middleware.All(ctx, r, func(ctx context.Context) (any, error) {\n", i)
			fmt.Fprintf(b, "\t\t\t\t\treturn %s(ctx, mwState)\n", nextName)
			b.WriteString("\t\t\t\t})\n")
		}
		b.WriteString("\t\t\t\tif err != nil {\n")
		b.WriteString("\t\t\t\t\treturn nil, err\n")
		b.WriteString("\t\t\t\t}\n")
		fmt.Fprintf(b, "\t\t\t\ttyped, ok := res.(%s%sResponse)\n", q, method.Name)
		b.WriteString("\t\t\t\tif !ok {\n")
		b.WriteString("\t\t\t\t\treturn nil, fmt.Errorf(\"middleware returned invalid response\")\n")
		b.WriteString("\t\t\t\t}\n")
		b.WriteString("\t\t\t\treturn typed, nil\n")
		b.WriteString("\t\t\t}\n")
		fmt.Fprintf(b, "\t\t\treturn %s(ctx, mwState)\n", nextName)
		b.WriteString("\t\t}\n")
		nextName = wrapperName
	}
	fmt.Fprintf(b, "\t\tres, err := %s(r.Context(), %sMiddlewareContext{})\n", nextName, q)
}

func writeCopyMiddlewareFields(b *strings.Builder, fields []FieldSpec) {
	for _, field := range fields {
		fmt.Fprintf(b, "\t\t\t\t\tmwState.%s = allCtx.%s\n", field.Name, field.Name)
	}
}

func writeParamsDecode(b *strings.Builder, route RouteSpec) {
	for _, ancestor := range route.ParamAncestors {
		if ancestor.Param == nil {
			continue
		}
		writeParamFieldDecode(b, ancestor)
	}
}

func writeParamFieldDecode(b *strings.Builder, ancestor ParamAncestor) {
	field := *ancestor.Param
	key := ancestor.SlugName
	target := "params." + ancestor.FieldKey
	baseType := strings.TrimPrefix(field.Type, "*")
	baseType = strings.TrimPrefix(baseType, "[]")
	fmt.Fprintf(b, "\t{\n\t\tval := r.PathValue(%q)\n", key)
	if field.Pointer && field.Slice {
		b.WriteString("\t\tif val != \"\" {\n")
		b.WriteString("\t\t\tvals := strings.Split(val, \"/\")\n")
		switch baseType {
		case "string":
			fmt.Fprintf(b, "\t\t\t%s = &vals\n", target)
		default:
			fmt.Fprintf(b, "\t\t\treturn params, &frourioParamDecodeError{Field: %q, Message: \"unsupported catch-all param type %s\"}\n", key, field.Type)
		}
		b.WriteString("\t\t}\n")
		b.WriteString("\t}\n")
		return
	}
	fmt.Fprintf(b, "\t\tif val == \"\" { return params, &frourioParamDecodeError{Field: %q, Message: \"missing path parameter\"} }\n", key)
	if field.Slice {
		b.WriteString("\t\tvals := strings.Split(val, \"/\")\n")
		switch baseType {
		case "string":
			fmt.Fprintf(b, "\t\t%s = vals\n", target)
		default:
			fmt.Fprintf(b, "\t\treturn params, &frourioParamDecodeError{Field: %q, Message: \"unsupported catch-all param type %s\"}\n", key, field.Type)
		}
		b.WriteString("\t}\n")
		return
	}
	switch baseType {
	case "string":
		if field.Pointer {
			fmt.Fprintf(b, "\t\t%s = &val\n", target)
		} else {
			fmt.Fprintf(b, "\t\t%s = val\n", target)
		}
	case "int":
		b.WriteString("\t\tv, err := strconv.Atoi(val)\n")
		fmt.Fprintf(b, "\t\tif err != nil { return params, &frourioParamDecodeError{Field: %q, Message: \"invalid int\"} }\n", key)
		if field.Pointer {
			fmt.Fprintf(b, "\t\t%s = &v\n", target)
		} else {
			fmt.Fprintf(b, "\t\t%s = v\n", target)
		}
	default:
		fmt.Fprintf(b, "\t\treturn params, &frourioParamDecodeError{Field: %q, Message: \"unsupported param type %s\"}\n", key, field.Type)
	}
	b.WriteString("\t}\n")
}

func optionalCatchAllPath(method MethodSpec) (string, bool) {
	if method.Param == nil || !method.Param.Pointer || !method.Param.Slice {
		return "", false
	}
	suffixStart := strings.LastIndex(method.URLPath, "/{")
	if suffixStart < 0 || !strings.HasSuffix(method.URLPath, "...}") {
		return "", false
	}
	return method.URLPath[:suffixStart], true
}

func writeQueryDecode(b *strings.Builder, field FieldSpec) {
	writeValuesDecode(b, "query", field)
}

func writeValuesDecode(b *strings.Builder, receiver string, field FieldSpec) {
	key := field.JSONName
	if key == "" {
		key = lowerName(field.SourceName)
	}
	baseType := strings.TrimPrefix(strings.TrimPrefix(field.Type, "*"), "[]")
	target := receiver + "." + field.Name

	if field.Slice {
		fmt.Fprintf(b, "\tif vals, ok := values[%q]; ok {\n", key)
		switch baseType {
		case "string":
			fmt.Fprintf(b, "\t\t%s = vals\n", target)
		case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64", "bool":
			fmt.Fprintf(b, "\t\tparsed := make([]%s, 0, len(vals))\n", baseType)
			b.WriteString("\t\tfor _, val := range vals {\n")
			fmt.Fprintf(b, "\t\t\tv, err := decode%s(val)\n", exportName(baseType))
			fmt.Fprintf(b, "\t\t\tif err != nil { return %s, err }\n", receiver)
			b.WriteString("\t\t\tparsed = append(parsed, v)\n")
			b.WriteString("\t\t}\n")
			fmt.Fprintf(b, "\t\t%s = parsed\n", target)
		default:
			fmt.Fprintf(b, "\t\treturn %s, fmt.Errorf(\"unsupported values type %s\")\n", receiver, field.Type)
		}
		b.WriteString("\t}\n")
		return
	}

	fmt.Fprintf(b, "\tif vals, ok := values[%q]; ok && len(vals) > 0 {\n", key)
	b.WriteString("\t\tval := vals[0]\n")
	b.WriteString("\t\tif val != \"\" {\n")
	switch baseType {
	case "string":
		if field.Pointer {
			fmt.Fprintf(b, "\t\t\t%s = &val\n", target)
		} else {
			fmt.Fprintf(b, "\t\t\t%s = val\n", target)
		}
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64", "bool":
		fmt.Fprintf(b, "\t\t\tv, err := decode%s(val)\n", exportName(baseType))
		fmt.Fprintf(b, "\t\t\tif err != nil { return %s, err }\n", receiver)
		if field.Pointer {
			fmt.Fprintf(b, "\t\t\t%s = &v\n", target)
		} else {
			fmt.Fprintf(b, "\t\t\t%s = v\n", target)
		}
	default:
		fmt.Fprintf(b, "\t\t\treturn %s, fmt.Errorf(\"unsupported values type %s\")\n", receiver, field.Type)
	}
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
}

func runtimeText() string {
	return `type frourioError struct {
	Error  string         ` + "`json:\"error\"`" + `
	Issues []frourioIssue ` + "`json:\"issues,omitempty\"`" + `
	Status int            ` + "`json:\"status\"`" + `
}

type frourioIssue struct {
	Message string ` + "`json:\"message\"`" + `
	Path    []any  ` + "`json:\"path\"`" + `
}

type frourioParamDecodeError struct {
	Field   string
	Message string
}

func (e *frourioParamDecodeError) Error() string {
	return e.Message
}

func writeValidationError(w http.ResponseWriter, err error, root string) {
	issues := []frourioIssue{}
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, validationError := range validationErrors {
			issues = append(issues, frourioIssue{
				Path:    []any{root, validationError.Field()},
				Message: validationError.Tag(),
			})
		}
	} else {
		issues = append(issues, frourioIssue{Path: []any{root}, Message: err.Error()})
	}
	writeRequestError(w, issues)
}

func writeRequestError(w http.ResponseWriter, issues []frourioIssue) {
	writeJSON(w, http.StatusUnprocessableEntity, frourioError{
		Status: http.StatusUnprocessableEntity,
		Error:  "Unprocessable Entity",
		Issues: issues,
	}, true)
}

func writeInternalError(w http.ResponseWriter) {
	writeJSON(w, http.StatusInternalServerError, frourioError{
		Status: http.StatusInternalServerError,
		Error:  "Internal Server Error",
	}, true)
}

func writeJSON(w http.ResponseWriter, status int, body any, autoContentType bool) {
	if autoContentType {
		w.Header().Set("content-type", "application/json")
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeText(w http.ResponseWriter, status int, body string, autoContentType bool) {
	if autoContentType {
		w.Header().Set("content-type", "text/plain; charset=utf-8")
	}
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func writeBytes(w http.ResponseWriter, status int, body []byte, autoContentType bool) {
	if autoContentType {
		w.Header().Set("content-type", "application/octet-stream")
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func writeMultipart(w http.ResponseWriter, status int, body any, autoContentType bool) {
	writer := multipart.NewWriter(w)
	if autoContentType {
		w.Header().Set("content-type", writer.FormDataContentType())
	}
	w.WriteHeader(status)
	defer func() { _ = writer.Close() }()
	writeMultipartFields(writer, body)
}

func writeMultipartFields(writer *multipart.Writer, body any) {
	val := reflect.ValueOf(body)
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		_ = writer.WriteField("body", fmt.Sprint(body))
		return
	}
	typ := val.Type()
	for i := range val.NumField() {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}
		key := field.Name
		if tag := field.Tag.Get("json"); tag != "" {
			key = strings.Split(tag, ",")[0]
			if key == "-" {
				continue
			}
			if key == "" {
				key = field.Name
			}
		}
		value := val.Field(i)
		if value.Kind() == reflect.Slice && value.Type().Elem().Kind() != reflect.Uint8 {
			for j := range value.Len() {
				_ = writer.WriteField(key, fmt.Sprint(value.Index(j).Interface()))
			}
			continue
		}
		_ = writer.WriteField(key, fmt.Sprint(value.Interface()))
	}
}

func decodeInt(val string) (int, error) {
	v, err := strconv.ParseInt(val, 10, 0)
	return int(v), err
}

func decodeInt8(val string) (int8, error) {
	v, err := strconv.ParseInt(val, 10, 8)
	return int8(v), err
}

func decodeInt16(val string) (int16, error) {
	v, err := strconv.ParseInt(val, 10, 16)
	return int16(v), err
}

func decodeInt32(val string) (int32, error) {
	v, err := strconv.ParseInt(val, 10, 32)
	return int32(v), err
}

func decodeInt64(val string) (int64, error) {
	return strconv.ParseInt(val, 10, 64)
}

func decodeUint(val string) (uint, error) {
	v, err := strconv.ParseUint(val, 10, 0)
	return uint(v), err
}

func decodeUint8(val string) (uint8, error) {
	v, err := strconv.ParseUint(val, 10, 8)
	return uint8(v), err
}

func decodeUint16(val string) (uint16, error) {
	v, err := strconv.ParseUint(val, 10, 16)
	return uint16(v), err
}

func decodeUint32(val string) (uint32, error) {
	v, err := strconv.ParseUint(val, 10, 32)
	return uint32(v), err
}

func decodeUint64(val string) (uint64, error) {
	return strconv.ParseUint(val, 10, 64)
}

func decodeFloat32(val string) (float32, error) {
	v, err := strconv.ParseFloat(val, 32)
	return float32(v), err
}

func decodeFloat64(val string) (float64, error) {
	return strconv.ParseFloat(val, 64)
}

func decodeBool(val string) (bool, error) {
	return strconv.ParseBool(val)
}

var _ = context.Background
var _ = fmt.Sprint
var _ = multipart.NewWriter
var _ = reflect.ValueOf
var _ = strconv.Itoa
var _ = strings.Split
`
}
