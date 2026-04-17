package cosmosdb

import (
	"testing"
)

func TestBuildQueryValidation(t *testing.T) {
	tests := []struct {
		name    string
		query   map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid simple key",
			query: map[string]interface{}{
				"name": "john",
			},
			wantErr: false,
		},
		{
			name: "valid nested key",
			query: map[string]interface{}{
				"address.city": "Seattle",
			},
			wantErr: false,
		},
		{
			name: "valid key with numbers and underscores",
			query: map[string]interface{}{
				"user_123_age": 30,
			},
			wantErr: false,
		},
		{
			name: "invalid key with spaces",
			query: map[string]interface{}{
				"first name": "john",
			},
			wantErr: true,
		},
		{
			name: "invalid key with SQL injection attempt",
			query: map[string]interface{}{
				"name = 'a' OR 1=1; --": "john",
			},
			wantErr: true,
		},
		{
			name: "invalid key with operator",
			query: map[string]interface{}{
				"age >= 18": true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := buildQuery(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildQuery() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
