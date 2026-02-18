package vault

import "testing"

func TestIsCanonicalField(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"identity.full_name", true},
		{"identity.email", true},
		{"payment.card_number", true},
		{"financial.ssn", true},
		{"identity.nickname", false},
		{"custom.field", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := IsCanonicalField(tt.id); got != tt.want {
			t.Errorf("IsCanonicalField(%q) = %v, want %v", tt.id, got, tt.want)
		}
	}
}

func TestGetSchemaField(t *testing.T) {
	f := GetSchemaField("identity.full_name")
	if f == nil {
		t.Fatal("expected non-nil for identity.full_name")
	}
	if f.Description == "" {
		t.Error("expected non-empty description")
	}

	if GetSchemaField("nonexistent.field") != nil {
		t.Error("expected nil for nonexistent field")
	}
}

func TestDefaultSensitivity(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"identity.full_name", "standard"},
		{"identity.phone", "sensitive"},
		{"financial.ssn", "critical"},
		{"payment.card_number", "critical"},
		{"preferences.timezone", "public"},
		{"custom.whatever", "standard"},
	}
	for _, tt := range tests {
		if got := DefaultSensitivity(tt.id); got != tt.want {
			t.Errorf("DefaultSensitivity(%q) = %q, want %q", tt.id, got, tt.want)
		}
	}
}

func TestSchemaIntegrity(t *testing.T) {
	seen := make(map[string]bool)
	for _, cat := range RecommendedSchema.Categories {
		if cat.Name == "" {
			t.Error("category with empty name")
		}
		for _, f := range cat.Fields {
			if seen[f.ID] {
				t.Errorf("duplicate field ID: %s", f.ID)
			}
			seen[f.ID] = true

			if err := ValidateFieldID(f.ID); err != nil {
				t.Errorf("invalid field ID in schema: %s: %v", f.ID, err)
			}
			if f.Description == "" {
				t.Errorf("field %s has empty description", f.ID)
			}
			if !validTiers[f.Sensitivity] {
				t.Errorf("field %s has invalid sensitivity %q", f.ID, f.Sensitivity)
			}
		}
	}
}

func TestSchemaIndex_MatchesData(t *testing.T) {
	count := 0
	for _, cat := range RecommendedSchema.Categories {
		count += len(cat.Fields)
	}
	if len(schemaIndex) != count {
		t.Errorf("schemaIndex has %d entries, but schema has %d fields", len(schemaIndex), count)
	}
}
