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
	Context *StructSpec
	Name    string
}

type MethodSpec struct {
	Param     *FieldSpec
	Query     *StructSpec
	Body      *StructSpec
	Header    *StructSpec
	Doc       DocSpec
	Name      string
	HTTPName  string
	URLPath   string
	Format    string
	Responses []ResponseSpec
	Raw       bool
}

type ResponseSpec struct {
	Body       *FieldSpec
	BodyStruct *StructSpec
	Header     *StructSpec
	Doc        DocSpec
	BodyType   string
	Status     int
	FormData   bool
}

type StructSpec struct {
	Doc      DocSpec
	Name     string
	TypeName string
	Fields   []FieldSpec
	Inline   bool
}

type FieldSpec struct {
	Name        string
	SourceName  string
	Type        string
	JSONName    string
	Doc         DocSpec
	ValidateTag string
	Tag         string
	JSONTagged  bool
	Pointer     bool
	Slice       bool
}

type DocSpec struct {
	Summary     string
	Description string
}
