package openapi

// OpenAPISpec represents the OpenAPI 3.0 specification structure
type OpenAPISpec struct {
	OpenAPI    string                `yaml:"openapi"`
	Info       Info                  `yaml:"info"`
	Servers    []Server              `yaml:"servers,omitempty"`
	Paths      map[string]PathItem   `yaml:"paths"`
	Components *Components           `yaml:"components,omitempty"`
	Security   []SecurityRequirement `yaml:"security,omitempty"`
}

// Info represents the info section of OpenAPI spec
type Info struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description,omitempty"`
	Version     string `yaml:"version"`
}

// Server represents a server in the OpenAPI spec
type Server struct {
	URL         string `yaml:"url"`
	Description string `yaml:"description,omitempty"`
}

// PathItem represents a path item in OpenAPI spec
type PathItem struct {
	Get    *Operation `yaml:"get,omitempty"`
	Post   *Operation `yaml:"post,omitempty"`
	Put    *Operation `yaml:"put,omitempty"`
	Delete *Operation `yaml:"delete,omitempty"`
}

// Operation represents an operation in OpenAPI spec
type Operation struct {
	Summary     string                `yaml:"summary,omitempty"`
	Description string                `yaml:"description,omitempty"`
	OperationID string                `yaml:"operationId,omitempty"`
	Parameters  []Parameter           `yaml:"parameters,omitempty"`
	RequestBody *RequestBody          `yaml:"requestBody,omitempty"`
	Responses   map[string]Response   `yaml:"responses"`
	Security    []SecurityRequirement `yaml:"security,omitempty"`
	Tags        []string              `yaml:"tags,omitempty"`
}

// Parameter represents a parameter in OpenAPI spec
type Parameter struct {
	Name        string  `yaml:"name"`
	In          string  `yaml:"in"` // "query", "header", "path", "cookie"
	Description string  `yaml:"description,omitempty"`
	Required    bool    `yaml:"required,omitempty"`
	Schema      *Schema `yaml:"schema,omitempty"`
}

// RequestBody represents a request body in OpenAPI spec
type RequestBody struct {
	Description string               `yaml:"description,omitempty"`
	Content     map[string]MediaType `yaml:"content"`
	Required    bool                 `yaml:"required,omitempty"`
}

// Response represents a response in OpenAPI spec
type Response struct {
	Description string               `yaml:"description"`
	Content     map[string]MediaType `yaml:"content,omitempty"`
}

// MediaType represents a media type in OpenAPI spec
type MediaType struct {
	Schema *Schema `yaml:"schema,omitempty"`
}

// Schema represents a schema in OpenAPI spec
type Schema struct {
	Type       string            `yaml:"type,omitempty"`
	Properties map[string]Schema `yaml:"properties,omitempty"`
	Items      *Schema           `yaml:"items,omitempty"`
	Required   []string          `yaml:"required,omitempty"`
	Ref        string            `yaml:"$ref,omitempty"`
}

// Components represents the components section of OpenAPI spec
type Components struct {
	Schemas         map[string]Schema         `yaml:"schemas,omitempty"`
	SecuritySchemes map[string]SecurityScheme `yaml:"securitySchemes,omitempty"`
}

// SecurityScheme represents a security scheme in OpenAPI spec
type SecurityScheme struct {
	Type          string `yaml:"type"`
	Scheme        string `yaml:"scheme,omitempty"`
	BearerFormat  string `yaml:"bearerFormat,omitempty"`
	In            string `yaml:"in,omitempty"`
	Name          string `yaml:"name,omitempty"`
	OpenIDConnect string `yaml:"openIdConnectUrl,omitempty"`
}

// SecurityRequirement represents a security requirement in OpenAPI spec
type SecurityRequirement map[string][]string
