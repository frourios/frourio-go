package generator

import (
	"fmt"
	"strings"
)

func openAPIDocument(routes []RouteSpec) map[string]any {
	paths := map[string]any{}
	components := map[string]any{
		"schemas": map[string]any{
			"FrourioError": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"status": map[string]any{"type": "integer"},
					"error":  map[string]any{"type": "string"},
					"issues": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"path":    map[string]any{"type": "array", "items": map[string]any{}},
								"message": map[string]any{"type": "string"},
							},
						},
					},
				},
			},
		},
	}

	for _, route := range routes {
		pathItem := map[string]any{}
		for _, method := range route.Methods {
			pathItem[strings.ToLower(method.HTTPName)] = operationObject(route, method, components["schemas"].(map[string]any), method.URLPath, true)
		}
		paths[openAPIPath(firstMethodPath(route))] = pathItem
		for _, method := range route.Methods {
			if optionalPath, ok := optionalCatchAllPath(method); ok {
				item := map[string]any{}
				item[strings.ToLower(method.HTTPName)] = operationObject(route, method, components["schemas"].(map[string]any), optionalPath, false)
				paths[optionalPath] = item
			}
		}
	}

	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":   "frourio-go API",
			"version": "0.1.0",
		},
		"paths":      paths,
		"components": components,
	}
}

func operationObject(route RouteSpec, method MethodSpec, schemas map[string]any, path string, includePathParam bool) map[string]any {
	op := map[string]any{
		"operationId": operationID(method.HTTPName, path),
		"responses":   responsesObject(route, method, schemas),
	}
	params := []any{}
	if method.Param != nil && includePathParam {
		param := map[string]any{
			"name":     pathParamName(route.RelDir),
			"in":       "path",
			"required": true,
			"schema":   schemaForField(*method.Param),
		}
		if method.Param.Slice {
			param["style"] = "simple"
			param["explode"] = false
			param["x-frourio-catch-all"] = true
		}
		params = append(params, param)
	}
	if method.Query != nil {
		for _, field := range method.Query.Fields {
			params = append(params, map[string]any{
				"name":     field.JSONName,
				"in":       "query",
				"required": !field.Pointer,
				"schema":   schemaForField(field),
			})
		}
	}
	if len(params) > 0 {
		op["parameters"] = params
	}
	if method.Body != nil {
		schemaName := schemaName(route, method, "Body")
		schemas[schemaName] = schemaForStruct(method.Body)
		op["requestBody"] = map[string]any{
			"required": true,
			"content": map[string]any{
				"application/json": map[string]any{
					"schema": map[string]any{"$ref": "#/components/schemas/" + schemaName},
				},
			},
		}
	}
	return op
}

func responsesObject(route RouteSpec, method MethodSpec, schemas map[string]any) map[string]any {
	responses := map[string]any{
		"422": map[string]any{
			"description": "Unprocessable Entity",
			"content": map[string]any{
				"application/json": map[string]any{
					"schema": map[string]any{"$ref": "#/components/schemas/FrourioError"},
				},
			},
		},
	}
	for _, res := range method.Responses {
		response := map[string]any{"description": httpStatusDescription(res.Status)}
		if res.Body != nil {
			schemaName := schemaName(route, method, fmt.Sprintf("Status%dBody", res.Status))
			schemas[schemaName] = schemaForField(*res.Body)
			response["content"] = map[string]any{
				"application/json": map[string]any{
					"schema": map[string]any{"$ref": "#/components/schemas/" + schemaName},
				},
			}
		}
		responses[fmt.Sprint(res.Status)] = response
	}
	return responses
}

func schemaForField(field FieldSpec) map[string]any {
	typ := strings.TrimPrefix(field.Type, "*")
	if strings.HasPrefix(typ, "[]") {
		item := field
		item.Type = strings.TrimPrefix(typ, "[]")
		return map[string]any{"type": "array", "items": schemaForField(item)}
	}

	schema := map[string]any{}
	switch typ {
	case "string":
		schema["type"] = "string"
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		schema["type"] = "integer"
	case "float32", "float64":
		schema["type"] = "number"
	case "bool":
		schema["type"] = "boolean"
	default:
		schema["type"] = "object"
		schema["x-go-type"] = typ
	}
	if field.ValidateTag != "" {
		schema["x-go-validate"] = field.ValidateTag
	}
	return schema
}

func schemaForStruct(st *StructSpec) map[string]any {
	props := map[string]any{}
	required := []string{}
	for _, field := range st.Fields {
		name := field.JSONName
		if name == "" {
			name = lowerName(field.Name)
		}
		props[name] = schemaForField(field)
		if !field.Pointer && strings.Contains(field.ValidateTag, "required") {
			required = append(required, name)
		}
	}
	schema := map[string]any{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func operationID(method, path string) string {
	parts := []string{strings.ToLower(method)}
	for _, part := range strings.Split(strings.Trim(path, "/"), "/") {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			param := strings.TrimSuffix(strings.TrimPrefix(part, "{"), "}")
			param = strings.TrimSuffix(param, "...")
			parts = append(parts, "By"+exportName(param))
			continue
		}
		for _, token := range strings.FieldsFunc(part, func(r rune) bool {
			return r == '-' || r == '_' || r == '{' || r == '}' || r == '.'
		}) {
			token = safeIdentToken(token)
			if token == "" {
				continue
			}
			parts = append(parts, exportName(token))
		}
	}
	return strings.Join(parts, "")
}

func schemaName(route RouteSpec, method MethodSpec, part string) string {
	base := strings.Trim(method.URLPath, "/")
	if base == "" {
		base = "root"
	}
	chunks := []string{}
	for _, token := range strings.FieldsFunc(base, func(r rune) bool {
		return r == '/' || r == '-' || r == '_' || r == '{' || r == '}' || r == '.'
	}) {
		token = strings.TrimSuffix(token, "...")
		token = safeIdentToken(token)
		if token == "" {
			continue
		}
		chunks = append(chunks, exportName(token))
	}
	chunks = append(chunks, method.Name, part)
	return strings.Join(chunks, "")
}

func safeIdentToken(token string) string {
	var b strings.Builder
	for _, r := range token {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func firstMethodPath(route RouteSpec) string {
	if len(route.Methods) == 0 {
		return route.URLPath
	}
	return route.Methods[0].URLPath
}

func openAPIPath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "...}") {
			parts[i] = strings.TrimSuffix(part, "...}") + "}"
		}
	}
	return strings.Join(parts, "/")
}

func httpStatusDescription(status int) string {
	switch status {
	case 200:
		return "OK"
	case 201:
		return "Created"
	case 204:
		return "No Content"
	case 400:
		return "Bad Request"
	case 404:
		return "Not Found"
	default:
		return fmt.Sprintf("Status %d", status)
	}
}
