package generator

import (
	"fmt"
	"strings"
)

func relayText(route RouteSpec) string {
	var b strings.Builder
	fmt.Fprintf(&b, "package %s\n\n", route.PackageName)
	if routeHasRaw(route) {
		b.WriteString("import (\n")
		b.WriteString("\t\"context\"\n")
		b.WriteString("\t\"net/http\"\n")
		b.WriteString(")\n\n")
	} else {
		b.WriteString("import \"context\"\n\n")
	}
	b.WriteString("type RouteDefinition struct {\n")
	b.WriteString("\tspec routeMetadata\n")
	b.WriteString("\thandlers RouteHandlers\n")
	b.WriteString("}\n\n")
	b.WriteString("type routeMetadata struct{}\n\n")
	b.WriteString("var routeSpec = routeMetadata{}\n\n")

	for _, method := range route.Methods {
		writeMethodTypes(&b, method)
	}
	if hasMiddleware(route) || len(route.Ancestors) > 0 {
		writeMiddlewareTypes(&b, route)
	}

	b.WriteString("type RouteHandlers struct {\n")
	if hasMiddleware(route) || len(route.Ancestors) > 0 {
		b.WriteString("\tMiddleware RouteMiddleware\n")
	}
	for _, method := range route.Methods {
		if methodHasMiddleware(route, method) {
			fmt.Fprintf(&b, "\t%s func(context.Context, %sRequest, %sContext) (%sResponse, error)\n", method.Name, method.Name, method.Name, method.Name)
		} else {
			fmt.Fprintf(&b, "\t%s func(context.Context, %sRequest) (%sResponse, error)\n", method.Name, method.Name, method.Name)
		}
	}
	b.WriteString("}\n\n")
	b.WriteString("func DefineRoute(handlers RouteHandlers) RouteDefinition {\n")
	b.WriteString("\treturn RouteDefinition{spec: routeSpec, handlers: handlers}\n")
	b.WriteString("}\n")
	b.WriteString("\nfunc (route RouteDefinition) Handlers() RouteHandlers {\n")
	b.WriteString("\treturn route.handlers\n")
	b.WriteString("}\n")
	return b.String()
}

func routeHasRaw(route RouteSpec) bool {
	for _, method := range route.Methods {
		if method.Raw {
			return true
		}
	}
	return false
}

func hasMiddleware(route RouteSpec) bool {
	return route.Middleware.All != nil || len(route.Middleware.Methods) > 0
}

func methodMiddleware(route RouteSpec, method MethodSpec) *MiddlewareItem {
	if route.Middleware.Methods == nil {
		return nil
	}
	return route.Middleware.Methods[method.Name]
}

func methodHasMiddleware(route RouteSpec, method MethodSpec) bool {
	return route.Middleware.All != nil || len(route.Ancestors) > 0 || methodMiddleware(route, method) != nil
}

func writeMiddlewareTypes(b *strings.Builder, route RouteSpec) {
	if route.Middleware.All != nil && route.Middleware.All.Context != nil {
		writeStruct(b, "MiddlewareAllContext", route.Middleware.All.Context.Fields)
	}
	b.WriteString("type MiddlewareContext struct {\n")
	for _, ancestor := range route.Ancestors {
		if ancestor.Middleware.All != nil && ancestor.Middleware.All.Context != nil {
			for _, field := range ancestor.Middleware.All.Context.Fields {
				fmt.Fprintf(b, "\t%s %s %s\n", field.Name, field.Type, goTag(&field, field.JSONName))
			}
		}
	}
	if route.Middleware.All != nil && route.Middleware.All.Context != nil {
		b.WriteString("\tMiddlewareAllContext\n")
	}
	b.WriteString("}\n\n")

	if route.Middleware.All != nil {
		if route.Middleware.All.Context != nil {
			b.WriteString("type MiddlewareNext func(context.Context, MiddlewareAllContext) (any, error)\n")
		} else {
			b.WriteString("type MiddlewareNext func(context.Context) (any, error)\n")
		}
		b.WriteString("type MiddlewareAll func(context.Context, MiddlewareNext) (any, error)\n\n")
	}

	for _, method := range route.Methods {
		item := methodMiddleware(route, method)
		if item != nil && item.Context != nil {
			writeStruct(b, method.Name+"MiddlewareContext", item.Context.Fields)
		}
		if methodHasMiddleware(route, method) {
			fmt.Fprintf(b, "type %sContext struct {\n", method.Name)
			b.WriteString("\tMiddlewareContext\n")
			if item != nil && item.Context != nil {
				fmt.Fprintf(b, "\t%sMiddlewareContext\n", method.Name)
			}
			b.WriteString("}\n\n")
		}
		if item != nil {
			if item.Context != nil {
				fmt.Fprintf(b, "type %sNext func(context.Context, %sRequest, %sMiddlewareContext) (%sResponse, error)\n", method.Name, method.Name, method.Name, method.Name)
			} else {
				fmt.Fprintf(b, "type %sNext func(context.Context, %sRequest) (%sResponse, error)\n", method.Name, method.Name, method.Name)
			}
			fmt.Fprintf(b, "type %sMiddleware func(context.Context, %sRequest, MiddlewareContext, %sNext) (%sResponse, error)\n\n", method.Name, method.Name, method.Name, method.Name)
		}
	}

	b.WriteString("type RouteMiddleware struct {\n")
	if route.Middleware.All != nil {
		b.WriteString("\tAll MiddlewareAll\n")
	}
	for _, method := range route.Methods {
		if methodMiddleware(route, method) != nil {
			fmt.Fprintf(b, "\t%s %sMiddleware\n", method.Name, method.Name)
		}
	}
	b.WriteString("}\n\n")
}

func writeMethodTypes(b *strings.Builder, method MethodSpec) {
	fmt.Fprintf(b, "type %sRequest struct {\n", method.Name)
	if method.Param != nil {
		fmt.Fprintf(b, "\tParam %s %s\n", method.Param.Type, goTag(method.Param, ""))
	}
	if method.Query != nil {
		fmt.Fprintf(b, "\tQuery %s\n", requestPartType(method, "Query", method.Query))
	}
	if method.Header != nil {
		fmt.Fprintf(b, "\tHeader %s\n", requestPartType(method, "Header", method.Header))
	}
	if method.Body != nil {
		fmt.Fprintf(b, "\tBody %s\n", requestPartType(method, "Body", method.Body))
	}
	b.WriteString("}\n\n")

	if method.Query != nil && method.Query.Inline {
		writeStruct(b, method.Name+"Query", method.Query.Fields)
	}
	if method.Header != nil && method.Header.Inline {
		writeStruct(b, method.Name+"Header", method.Header.Fields)
	}
	if method.Body != nil && method.Body.Inline {
		writeStruct(b, method.Name+"Body", method.Body.Fields)
	}

	if method.Raw {
		b.WriteString("type RawResponse interface {\n")
		b.WriteString("\tWriteHTTP(http.ResponseWriter, *http.Request) error\n")
		b.WriteString("}\n\n")
		b.WriteString("type RawResponseFunc func(http.ResponseWriter, *http.Request) error\n\n")
		b.WriteString("func (f RawResponseFunc) WriteHTTP(w http.ResponseWriter, r *http.Request) error { return f(w, r) }\n\n")
		fmt.Fprintf(b, "type %sResponse = RawResponse\n\n", method.Name)
		return
	}

	fmt.Fprintf(b, "type %sResponse interface {\n", method.Name)
	fmt.Fprintf(b, "\tis%sResponse()\n", method.Name)
	b.WriteString("\tStatusCode() int\n")
	b.WriteString("}\n\n")

	for _, res := range method.Responses {
		typeName := fmt.Sprintf("%sStatus%d", method.Name, res.Status)
		if res.Header != nil && res.Header.Inline {
			writeStruct(b, typeName+"Header", res.Header.Fields)
		}
		if res.BodyStruct != nil && res.BodyStruct.Inline {
			writeStruct(b, typeName+"Body", res.BodyStruct.Fields)
		}
		fmt.Fprintf(b, "type %s struct {\n", typeName)
		if res.Header != nil {
			fmt.Fprintf(b, "\tHeader %s\n", responsePartType(typeName, "Header", res.Header))
		}
		if res.Body != nil {
			bodyType := res.Body.Type
			if res.BodyStruct != nil {
				bodyType = responsePartType(typeName, "Body", res.BodyStruct)
			}
			fmt.Fprintf(b, "\tBody %s %s\n", bodyType, goTag(res.Body, "body"))
		}
		b.WriteString("}\n\n")
		fmt.Fprintf(b, "func (%s) is%sResponse() {}\n", typeName, method.Name)
		fmt.Fprintf(b, "func (%s) StatusCode() int { return %d }\n\n", typeName, res.Status)
	}
}

func requestPartType(method MethodSpec, part string, st *StructSpec) string {
	if st != nil && !st.Inline && st.TypeName != "" {
		return st.TypeName
	}
	return method.Name + part
}

func responsePartType(responseType, part string, st *StructSpec) string {
	if st != nil && !st.Inline && st.TypeName != "" {
		return st.TypeName
	}
	return responseType + part
}

func writeStruct(b *strings.Builder, name string, fields []FieldSpec) {
	fmt.Fprintf(b, "type %s struct {\n", name)
	for _, field := range fields {
		fmt.Fprintf(b, "\t%s %s %s\n", field.Name, field.Type, goTag(&field, field.JSONName))
	}
	b.WriteString("}\n\n")
}

func goTag(field *FieldSpec, fallbackJSON string) string {
	parts := []string{}
	if field.JSONTagged {
		jsonName := field.JSONName
		if jsonName == "" {
			jsonName = fallbackJSON
		}
		if jsonName != "" {
			parts = append(parts, fmt.Sprintf(`json:"%s"`, jsonName))
		}
	}
	if field.ValidateTag != "" {
		parts = append(parts, fmt.Sprintf(`validate:"%s"`, field.ValidateTag))
	}
	if len(parts) == 0 {
		return ""
	}
	return "`" + strings.Join(parts, " ") + "`"
}
