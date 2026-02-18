package vault

import "testing"

func TestScopeAllows(t *testing.T) {
	tests := []struct {
		scope   string
		fieldID string
		want    bool
	}{
		// Wildcard
		{"*", "identity.full_name", true},
		{"*", "financial.income", true},

		// Category wildcard
		{"identity.*", "identity.full_name", true},
		{"identity.*", "identity.dob", true},
		{"identity.*", "financial.income", false},

		// Exact field
		{"identity.full_name", "identity.full_name", true},
		{"identity.full_name", "identity.dob", false},
		{"identity.full_name", "financial.income", false},

		// Multiple patterns
		{"identity.*,financial.*", "identity.full_name", true},
		{"identity.*,financial.*", "financial.income", true},
		{"identity.*,financial.*", "addresses.home_city", false},

		// Mixed patterns
		{"identity.*,financial.income", "financial.income", true},
		{"identity.*,financial.income", "financial.ssn", false},

		// Whitespace in patterns
		{"identity.* , financial.*", "financial.income", true},

		// Empty/edge cases
		{"", "identity.full_name", false},
		{"identity.*", "", false},
	}

	for _, tt := range tests {
		got := ScopeAllows(tt.scope, tt.fieldID)
		if got != tt.want {
			t.Errorf("ScopeAllows(%q, %q) = %v, want %v", tt.scope, tt.fieldID, got, tt.want)
		}
	}
}

func TestValidateFieldID(t *testing.T) {
	valid := []string{
		"identity.full_name",
		"financial.income",
		"addresses.home_city",
		"identity.t-shirt_size",
		"my_category.my_field",
		"A.B",
	}
	for _, id := range valid {
		if err := ValidateFieldID(id); err != nil {
			t.Errorf("ValidateFieldID(%q) = %v, want nil", id, err)
		}
	}

	invalid := []string{
		"",
		"noDot",
		".leading_dot",
		"trailing_dot.",
		"identity.full name",       // space
		"identity.full/name",       // slash
		"../etc/passwd.evil",       // path traversal
		"identity.name\x00extra",   // null byte
		"identity.name;DROP TABLE", // SQL injection attempt
		"cat.field.extra",          // three parts still valid as SplitN(2) keeps "field.extra"
	}
	for _, id := range invalid {
		if err := ValidateFieldID(id); err == nil {
			t.Errorf("ValidateFieldID(%q) = nil, want error", id)
		}
	}
}

func TestScopeAllowsCategory(t *testing.T) {
	tests := []struct {
		scope    string
		category string
		want     bool
	}{
		{"*", "identity", true},
		{"identity.*", "identity", true},
		{"identity.*", "financial", false},
		{"identity.*,financial.*", "financial", true},
		{"identity.*,financial.*", "addresses", false},
		{"identity.full_name", "identity", true},
		{"identity.full_name", "financial", false},
		{"", "identity", false},
	}

	for _, tt := range tests {
		got := ScopeAllowsCategory(tt.scope, tt.category)
		if got != tt.want {
			t.Errorf("ScopeAllowsCategory(%q, %q) = %v, want %v", tt.scope, tt.category, got, tt.want)
		}
	}
}
