package openapi

// OpenAPISpec represents the OpenAPI 3.0 specification structure
type OpenAPISpec struct {
	OpenAPI    string                `yaml:"openapi" json:"openapi"`
	Info       Info                  `yaml:"info" json:"info"`
	Servers    []Server              `yaml:"servers,omitempty" json:"servers,omitempty"`
	Paths      map[string]PathItem   `yaml:"paths" json:"paths"`
	Components *Components           `yaml:"components,omitempty" json:"components,omitempty"`
	Security   []SecurityRequirement `yaml:"security,omitempty" json:"security,omitempty"`
}

// Info represents the info section of OpenAPI spec
type Info struct {
	Title       string `yaml:"title" json:"title"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Version     string `yaml:"version" json:"version"`
}

// Server represents a server in the OpenAPI spec
type Server struct {
	URL         string `yaml:"url" json:"url"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// PathItem represents a path item in OpenAPI spec
type PathItem struct {
	Get    *Operation `yaml:"get,omitempty" json:"get,omitempty"`
	Post   *Operation `yaml:"post,omitempty" json:"post,omitempty"`
	Put    *Operation `yaml:"put,omitempty" json:"put,omitempty"`
	Delete *Operation `yaml:"delete,omitempty" json:"delete,omitempty"`
}

// Operation represents an operation in OpenAPI spec
type Operation struct {
	Summary     string                `yaml:"summary,omitempty" json:"summary,omitempty"`
	Description string                `yaml:"description,omitempty" json:"description,omitempty"`
	OperationID string                `yaml:"operationId,omitempty" json:"operationId,omitempty"`
	Parameters  []Parameter           `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	RequestBody *RequestBody          `yaml:"requestBody,omitempty" json:"requestBody,omitempty"`
	Responses   map[string]Response   `yaml:"responses" json:"responses"`
	Security    []SecurityRequirement `yaml:"security,omitempty" json:"security,omitempty"`
	Tags        []string              `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// Parameter represents a parameter in OpenAPI spec
type Parameter struct {
	Name        string  `yaml:"name" json:"name"`
	In          string  `yaml:"in" json:"in"` // "query", "header", "path", "cookie"
	Description string  `yaml:"description,omitempty" json:"description,omitempty"`
	Required    bool    `yaml:"required,omitempty" json:"required,omitempty"`
	Schema      *Schema `yaml:"schema,omitempty" json:"schema,omitempty"`
}

// RequestBody represents a request body in OpenAPI spec
type RequestBody struct {
	Description string               `yaml:"description,omitempty" json:"description,omitempty"`
	Content     map[string]MediaType `yaml:"content" json:"content"`
	Required    bool                 `yaml:"required,omitempty" json:"required,omitempty"`
}

// Response represents a response in OpenAPI spec
type Response struct {
	Description string               `yaml:"description" json:"description"`
	Content     map[string]MediaType `yaml:"content,omitempty" json:"content,omitempty"`
}

// MediaType represents a media type in OpenAPI spec
type MediaType struct {
	Schema *Schema `yaml:"schema,omitempty" json:"schema,omitempty"`
}

// Schema represents a schema in OpenAPI spec
type Schema struct {
	Type       string            `yaml:"type,omitempty" json:"type,omitempty"`
	Properties map[string]Schema `yaml:"properties,omitempty" json:"properties,omitempty"`
	Items      *Schema           `yaml:"items,omitempty" json:"items,omitempty"`
	Required   []string          `yaml:"required,omitempty" json:"required,omitempty"`
	Ref        string            `yaml:"$ref,omitempty" json:"$ref,omitempty"`
}

// Components represents the components section of OpenAPI spec
type Components struct {
	Schemas         map[string]Schema         `yaml:"schemas,omitempty" json:"schemas,omitempty"`
	SecuritySchemes map[string]SecurityScheme `yaml:"securitySchemes,omitempty" json:"securitySchemes,omitempty"`
}

// SecurityScheme represents a security scheme in OpenAPI spec
type SecurityScheme struct {
	Type          string `yaml:"type" json:"type"`
	Scheme        string `yaml:"scheme,omitempty" json:"scheme,omitempty"`
	BearerFormat  string `yaml:"bearerFormat,omitempty" json:"bearerFormat,omitempty"`
	In            string `yaml:"in,omitempty" json:"in,omitempty"`
	Name          string `yaml:"name,omitempty" json:"name,omitempty"`
	OpenIDConnect string `yaml:"openIdConnectUrl,omitempty" json:"openIdConnectUrl,omitempty"`
}

// SecurityRequirement represents a security requirement in OpenAPI spec
type SecurityRequirement map[string][]string
