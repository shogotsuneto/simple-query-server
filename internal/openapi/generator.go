package openapi

import (
	"fmt"
	"strings"

	"github.com/shogotsuneto/simple-query-server/internal/config"
	"gopkg.in/yaml.v3"
)

// Generator generates OpenAPI specifications from server configurations
type Generator struct {
	dbConfig      *config.DatabaseConfig
	queriesConfig *config.QueriesConfig
	serverConfig  *config.ServerConfig
	baseURL       string
}

// NewGenerator creates a new OpenAPI spec generator
func NewGenerator(dbConfig *config.DatabaseConfig, queriesConfig *config.QueriesConfig, serverConfig *config.ServerConfig, baseURL string) *Generator {
	return &Generator{
		dbConfig:      dbConfig,
		queriesConfig: queriesConfig,
		serverConfig:  serverConfig,
		baseURL:       baseURL,
	}
}

// GenerateSpec generates the complete OpenAPI specification
func (g *Generator) GenerateSpec() (*OpenAPISpec, error) {
	spec := &OpenAPISpec{
		OpenAPI: "3.0.3",
		Info: Info{
			Title:       "Simple Query Server API",
			Description: "REST API for executing database queries defined in YAML configuration files",
			Version:     "1.0.0",
		},
		Servers: []Server{
			{
				URL:         g.baseURL,
				Description: "Simple Query Server",
			},
		},
		Paths: make(map[string]PathItem),
		Components: &Components{
			Schemas:         make(map[string]Schema),
			SecuritySchemes: make(map[string]SecurityScheme),
		},
	}

	// Add standard endpoints
	g.addStandardEndpoints(spec)

	// Add query endpoints
	g.addQueryEndpoints(spec)

	// Add security schemes from middleware configuration
	g.addSecuritySchemes(spec)

	// Add global security if middleware requires it
	g.addGlobalSecurity(spec)

	return spec, nil
}

// GenerateYAML generates the OpenAPI spec as YAML string
func (g *Generator) GenerateYAML() (string, error) {
	spec, err := g.GenerateSpec()
	if err != nil {
		return "", err
	}

	yamlData, err := yaml.Marshal(spec)
	if err != nil {
		return "", fmt.Errorf("failed to marshal OpenAPI spec to YAML: %w", err)
	}

	return string(yamlData), nil
}

// addStandardEndpoints adds the standard server endpoints
func (g *Generator) addStandardEndpoints(spec *OpenAPISpec) {
	// Root endpoint
	spec.Paths["/"] = PathItem{
		Get: &Operation{
			Summary:     "Get server information",
			Description: "Returns basic information about the server and available endpoints",
			OperationID: "getServerInfo",
			Tags:        []string{"server"},
			Responses: map[string]Response{
				"200": {
					Description: "Server information",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Ref: "#/components/schemas/ServerInfo",
							},
						},
					},
				},
			},
		},
	}

	// Health endpoint
	spec.Paths["/health"] = PathItem{
		Get: &Operation{
			Summary:     "Health check",
			Description: "Returns the health status of the server and its dependencies",
			OperationID: "getHealth",
			Tags:        []string{"server"},
			Responses: map[string]Response{
				"200": {
					Description: "Service is healthy",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Ref: "#/components/schemas/HealthResponse",
							},
						},
					},
				},
				"503": {
					Description: "Service is unhealthy",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Ref: "#/components/schemas/HealthResponse",
							},
						},
					},
				},
			},
		},
	}

	// Queries list endpoint
	spec.Paths["/queries"] = PathItem{
		Get: &Operation{
			Summary:     "List available queries",
			Description: "Returns all configured queries with their parameters",
			OperationID: "listQueries",
			Tags:        []string{"queries"},
			Responses: map[string]Response{
				"200": {
					Description: "List of available queries",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Ref: "#/components/schemas/QueriesResponse",
							},
						},
					},
				},
			},
		},
	}

	// Add standard response schemas
	g.addStandardSchemas(spec)
}

// addQueryEndpoints adds endpoints for each configured query
func (g *Generator) addQueryEndpoints(spec *OpenAPISpec) {
	for queryName, query := range g.queriesConfig.Queries {
		path := fmt.Sprintf("/query/%s", queryName)

		operation := &Operation{
			Summary:     fmt.Sprintf("Execute %s query", queryName),
			Description: fmt.Sprintf("Executes the %s query with the provided parameters", queryName),
			OperationID: fmt.Sprintf("execute_%s", queryName),
			Tags:        []string{"queries"},
			Responses: map[string]Response{
				"200": {
					Description: "Query executed successfully",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Ref: "#/components/schemas/QueryResponse",
							},
						},
					},
				},
				"400": {
					Description: "Bad request - invalid parameters",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Ref: "#/components/schemas/ErrorResponse",
							},
						},
					},
				},
				"404": {
					Description: "Query not found",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Ref: "#/components/schemas/ErrorResponse",
							},
						},
					},
				},
			},
		}

		// Add request body if the query has parameters
		if len(query.Params) > 0 {
			requestSchema := g.createRequestSchema(query)
			operation.RequestBody = &RequestBody{
				Description: fmt.Sprintf("Parameters for %s query", queryName),
				Required:    true,
				Content: map[string]MediaType{
					"application/json": {
						Schema: requestSchema,
					},
				},
			}
		} else {
			// Even for queries without parameters, we accept empty JSON
			operation.RequestBody = &RequestBody{
				Description: "Empty JSON object (no parameters required)",
				Required:    false,
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{
							Type: "object",
						},
					},
				},
			}
		}

		// Add security requirements based on middleware
		if g.hasSecurityMiddleware() {
			operation.Security = g.getSecurityRequirements()
		}

		spec.Paths[path] = PathItem{
			Post: operation,
		}
	}
}

// createRequestSchema creates a schema for query request parameters
func (g *Generator) createRequestSchema(query config.Query) *Schema {
	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]Schema),
		Required:   make([]string, 0),
	}

	// Add body parameters
	for _, param := range query.Params {
		paramSchema := Schema{
			Type: g.convertParamType(param.Type),
		}
		schema.Properties[param.Name] = paramSchema
		schema.Required = append(schema.Required, param.Name)
	}

	return schema
}

// convertParamType converts query parameter types to OpenAPI types
func (g *Generator) convertParamType(paramType string) string {
	switch strings.ToLower(paramType) {
	case "int", "integer":
		return "integer"
	case "float", "double", "number":
		return "number"
	case "bool", "boolean":
		return "boolean"
	case "string", "text":
		return "string"
	default:
		return "string"
	}
}

// addSecuritySchemes adds security schemes based on middleware configuration
func (g *Generator) addSecuritySchemes(spec *OpenAPISpec) {
	if g.serverConfig == nil {
		return
	}

	for _, middleware := range g.serverConfig.Middleware {
		switch middleware.Type {
		case "http-header":
			if config, ok := middleware.Config["header"]; ok {
				if headerName, ok := config.(string); ok {
					schemeName := fmt.Sprintf("ApiKey_%s", strings.ReplaceAll(headerName, "-", "_"))
					spec.Components.SecuritySchemes[schemeName] = SecurityScheme{
						Type: "apiKey",
						In:   "header",
						Name: headerName,
					}
				}
			}
		case "bearer-jwks":
			spec.Components.SecuritySchemes["BearerAuth"] = SecurityScheme{
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
			}
		}
	}
}

// hasSecurityMiddleware checks if there's any security middleware configured
func (g *Generator) hasSecurityMiddleware() bool {
	if g.serverConfig == nil {
		return false
	}

	for _, middleware := range g.serverConfig.Middleware {
		if middleware.Type == "http-header" || middleware.Type == "bearer-jwks" {
			return true
		}
	}
	return false
}

// getSecurityRequirements returns security requirements based on middleware
func (g *Generator) getSecurityRequirements() []SecurityRequirement {
	if g.serverConfig == nil {
		return nil
	}

	var requirements []SecurityRequirement

	for _, middleware := range g.serverConfig.Middleware {
		switch middleware.Type {
		case "http-header":
			if config, ok := middleware.Config["header"]; ok {
				if headerName, ok := config.(string); ok {
					schemeName := fmt.Sprintf("ApiKey_%s", strings.ReplaceAll(headerName, "-", "_"))
					requirements = append(requirements, SecurityRequirement{
						schemeName: []string{},
					})
				}
			}
		case "bearer-jwks":
			requirements = append(requirements, SecurityRequirement{
				"BearerAuth": []string{},
			})
		}
	}

	return requirements
}

// addGlobalSecurity adds global security requirements
func (g *Generator) addGlobalSecurity(spec *OpenAPISpec) {
	if g.hasSecurityMiddleware() {
		spec.Security = g.getSecurityRequirements()
	}
}

// addStandardSchemas adds standard response schemas
func (g *Generator) addStandardSchemas(spec *OpenAPISpec) {
	// Server info schema
	spec.Components.Schemas["ServerInfo"] = Schema{
		Type: "object",
		Properties: map[string]Schema{
			"service": {Type: "string"},
			"status":  {Type: "string"},
			"endpoints": {
				Type: "object",
				Properties: map[string]Schema{
					"/health":       {Type: "string"},
					"/queries":      {Type: "string"},
					"/query/{name}": {Type: "string"},
				},
			},
		},
	}

	// Health response schema
	spec.Components.Schemas["HealthResponse"] = Schema{
		Type: "object",
		Properties: map[string]Schema{
			"status": {Type: "string"},
			"database": {
				Type: "object",
				Properties: map[string]Schema{
					"connected": {Type: "boolean"},
				},
			},
			"middleware": {
				Type: "object",
				Properties: map[string]Schema{
					"healthy": {Type: "boolean"},
				},
			},
		},
		Required: []string{"status", "database"},
	}

	// Queries response schema
	spec.Components.Schemas["QueriesResponse"] = Schema{
		Type: "object",
		Properties: map[string]Schema{
			"queries": {
				Type: "object",
			},
		},
	}

	// Query response schema
	spec.Components.Schemas["QueryResponse"] = Schema{
		Type: "object",
		Properties: map[string]Schema{
			"rows": {
				Type: "array",
				Items: &Schema{
					Type: "object",
				},
			},
		},
	}

	// Error response schema
	spec.Components.Schemas["ErrorResponse"] = Schema{
		Type: "object",
		Properties: map[string]Schema{
			"error": {Type: "string"},
		},
		Required: []string{"error"},
	}
}
