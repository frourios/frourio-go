package generator

type Options struct {
	APIDir      string
	OpenAPIPath string
	OnlyOpenAPI bool
}

type RouteSpec struct {
	Dir          string
	RelDir       string
	PackageName  string
	ImportPath   string
	Alias        string
	PathOverride string
	URLPath      string
	Middleware   MiddlewareSpec
	Ancestors    []RouteSpec
	Methods      []MethodSpec
}

type MiddlewareSpec struct {
	All     *MiddlewareItem
	Methods map[string]*MiddlewareItem
}

type MiddlewareItem struct {
	Name    string
	Context *StructSpec
}

type MethodSpec struct {
	Name      string
	HTTPName  string
	URLPath   string
	Format    string
	Param     *FieldSpec
	Query     *StructSpec
	Body      *StructSpec
	Header    *StructSpec
	Raw       bool
	Responses []ResponseSpec
}

type ResponseSpec struct {
	Status     int
	Body       *FieldSpec
	BodyType   string
	BodyStruct *StructSpec
	Header     *StructSpec
	FormData   bool
}

type StructSpec struct {
	Name   string
	Fields []FieldSpec
}

type FieldSpec struct {
	Name        string
	SourceName  string
	Type        string
	JSONName    string
	JSONTagged  bool
	ValidateTag string
	Tag         string
	Pointer     bool
	Slice       bool
}
