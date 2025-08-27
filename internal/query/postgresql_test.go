package query

import (
	"reflect"
	"testing"
)

func TestPostgreSQLExecutor_convertSQLParameters(t *testing.T) {
	// Create a PostgreSQL executor for testing (without database connection)
	executor := &PostgreSQLExecutor{}

	tests := []struct {
		name         string
		sql          string
		params       map[string]interface{}
		expectedSQL  string
		expectedArgs []interface{}
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "no parameters",
			sql:          "SELECT * FROM users",
			params:       map[string]interface{}{},
			expectedSQL:  "SELECT * FROM users",
			expectedArgs: []interface{}{},
			expectError:  false,
		},
		{
			name:         "single parameter",
			sql:          "SELECT * FROM users WHERE id = :id",
			params:       map[string]interface{}{"id": 1},
			expectedSQL:  "SELECT * FROM users WHERE id = $1",
			expectedArgs: []interface{}{1},
			expectError:  false,
		},
		{
			name:         "multiple parameters",
			sql:          "SELECT * FROM users WHERE name = :name AND age = :age",
			params:       map[string]interface{}{"name": "John", "age": 30},
			expectedSQL:  "SELECT * FROM users WHERE name = $1 AND age = $2",
			expectedArgs: []interface{}{"John", 30},
			expectError:  false,
		},
		{
			name:         "duplicate parameters",
			sql:          "SELECT * FROM users WHERE id = :id OR parent_id = :id",
			params:       map[string]interface{}{"id": 1},
			expectedSQL:  "SELECT * FROM users WHERE id = $1 OR parent_id = $1",
			expectedArgs: []interface{}{1},
			expectError:  false,
		},
		{
			name:         "parameter order matters",
			sql:          "SELECT * FROM users WHERE created_at > :start_date AND created_at < :end_date AND status = :status",
			params:       map[string]interface{}{"end_date": "2023-12-31", "status": "active", "start_date": "2023-01-01"},
			expectedSQL:  "SELECT * FROM users WHERE created_at > $1 AND created_at < $2 AND status = $3",
			expectedArgs: []interface{}{"2023-01-01", "2023-12-31", "active"},
			expectError:  false,
		},
		{
			name:         "parameters with underscores",
			sql:          "SELECT * FROM users WHERE user_id = :user_id AND first_name = :first_name",
			params:       map[string]interface{}{"user_id": 123, "first_name": "Alice"},
			expectedSQL:  "SELECT * FROM users WHERE user_id = $1 AND first_name = $2",
			expectedArgs: []interface{}{123, "Alice"},
			expectError:  false,
		},
		{
			name:         "complex SQL with JOIN",
			sql:          "SELECT u.name, p.bio FROM users u JOIN profiles p ON u.id = p.user_id WHERE u.status = :status AND p.bio LIKE :search_term",
			params:       map[string]interface{}{"status": "active", "search_term": "%engineer%"},
			expectedSQL:  "SELECT u.name, p.bio FROM users u JOIN profiles p ON u.id = p.user_id WHERE u.status = $1 AND p.bio LIKE $2",
			expectedArgs: []interface{}{"active", "%engineer%"},
			expectError:  false,
		},
		{
			name:         "parameter in multiple contexts",
			sql:          "UPDATE users SET name = :name, updated_by = :user_id WHERE id = :user_id AND name != :name",
			params:       map[string]interface{}{"name": "John Doe", "user_id": 1},
			expectedSQL:  "UPDATE users SET name = $1, updated_by = $2 WHERE id = $2 AND name != $1",
			expectedArgs: []interface{}{"John Doe", 1},
			expectError:  false,
		},
		{
			name:         "missing required parameter",
			sql:          "SELECT * FROM users WHERE id = :id AND name = :name",
			params:       map[string]interface{}{"id": 1},
			expectedSQL:  "",
			expectedArgs: nil,
			expectError:  true,
			errorMsg:     "parameter 'name' referenced in SQL but not provided",
		},
		{
			name:         "missing multiple parameters",
			sql:          "SELECT * FROM users WHERE id = :id",
			params:       map[string]interface{}{},
			expectedSQL:  "",
			expectedArgs: nil,
			expectError:  true,
			errorMsg:     "parameter 'id' referenced in SQL but not provided",
		},
		{
			name:         "parameters with different types",
			sql:          "SELECT * FROM users WHERE id = :id AND active = :active AND score = :score AND name = :name",
			params:       map[string]interface{}{"id": 42, "active": true, "score": 95.5, "name": "Alice"},
			expectedSQL:  "SELECT * FROM users WHERE id = $1 AND active = $2 AND score = $3 AND name = $4",
			expectedArgs: []interface{}{42, true, 95.5, "Alice"},
			expectError:  false,
		},
		{
			name:         "parameter at end of query",
			sql:          "SELECT * FROM users ORDER BY created_at DESC LIMIT :limit",
			params:       map[string]interface{}{"limit": 10},
			expectedSQL:  "SELECT * FROM users ORDER BY created_at DESC LIMIT $1",
			expectedArgs: []interface{}{10},
			expectError:  false,
		},
		{
			name:         "parameter in parentheses",
			sql:          "SELECT * FROM users WHERE id IN (:id1, :id2, :id3)",
			params:       map[string]interface{}{"id1": 1, "id2": 2, "id3": 3},
			expectedSQL:  "SELECT * FROM users WHERE id IN ($1, $2, $3)",
			expectedArgs: []interface{}{1, 2, 3},
			expectError:  false,
		},
		{
			name:         "case sensitive parameters",
			sql:          "SELECT * FROM users WHERE name = :Name AND email = :EMAIL",
			params:       map[string]interface{}{"Name": "John", "EMAIL": "john@example.com"},
			expectedSQL:  "SELECT * FROM users WHERE name = $1 AND email = $2",
			expectedArgs: []interface{}{"John", "john@example.com"},
			expectError:  false,
		},
		{
			name:         "empty string and nil values",
			sql:          "SELECT * FROM users WHERE name = :name AND description = :description",
			params:       map[string]interface{}{"name": "", "description": nil},
			expectedSQL:  "SELECT * FROM users WHERE name = $1 AND description = $2",
			expectedArgs: []interface{}{"", nil},
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotArgs, err := executor.convertSQLParameters(tt.sql, tt.params)

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("expected error message %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			// Check for unexpected error
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check SQL conversion
			if gotSQL != tt.expectedSQL {
				t.Errorf("SQL mismatch:\nexpected: %q\ngot:      %q", tt.expectedSQL, gotSQL)
			}

			// Check arguments
			if !reflect.DeepEqual(gotArgs, tt.expectedArgs) {
				t.Errorf("Args mismatch:\nexpected: %+v\ngot:      %+v", tt.expectedArgs, gotArgs)
			}
		})
	}
}