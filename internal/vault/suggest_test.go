package vault

import "testing"

func TestSuggestCanonical_Synonyms(t *testing.T) {
	tests := []struct {
		id            string
		wantCanonical string
	}{
		{"identity.name", "identity.full_name"},
		{"identity.dob", "identity.date_of_birth"},
		{"identity.firstname", "identity.first_name"},
		{"identity.lastname", "identity.last_name"},
		{"identity.mail", "identity.email"},
		{"addresses.zip", "addresses.home_zip"},
		{"addresses.street", "addresses.home_street"},
		{"addresses.country", "addresses.home_country"},
		{"payment.cc_number", "payment.card_number"},
		{"employment.company", "employment.employer"},
		{"preferences.tz", "preferences.timezone"},
	}
	for _, tt := range tests {
		s := SuggestCanonical(tt.id)
		if s == nil {
			t.Errorf("SuggestCanonical(%q) = nil, want %q", tt.id, tt.wantCanonical)
			continue
		}
		if s.Canonical != tt.wantCanonical {
			t.Errorf("SuggestCanonical(%q).Canonical = %q, want %q", tt.id, s.Canonical, tt.wantCanonical)
		}
		if s.Reason != "synonym" {
			t.Errorf("SuggestCanonical(%q).Reason = %q, want 'synonym'", tt.id, s.Reason)
		}
	}
}

func TestSuggestCanonical_Typos(t *testing.T) {
	tests := []struct {
		id            string
		wantCanonical string
	}{
		{"identity.fist_name", "identity.first_name"},   // missing 'r'
		{"identity.ful_name", "identity.full_name"},     // missing 'l'
		{"addresses.home_cty", "addresses.home_city"},   // missing 'i'
		{"payment.card_bran", "payment.card_brand"},     // missing 'd'
	}
	for _, tt := range tests {
		s := SuggestCanonical(tt.id)
		if s == nil {
			t.Errorf("SuggestCanonical(%q) = nil, want %q", tt.id, tt.wantCanonical)
			continue
		}
		if s.Canonical != tt.wantCanonical {
			t.Errorf("SuggestCanonical(%q).Canonical = %q, want %q", tt.id, s.Canonical, tt.wantCanonical)
		}
		if s.Reason != "similar" {
			t.Errorf("SuggestCanonical(%q).Reason = %q, want 'similar'", tt.id, s.Reason)
		}
	}
}

func TestSuggestCanonical_NoSuggestion(t *testing.T) {
	tests := []string{
		"identity.full_name",   // already canonical
		"identity.email",       // already canonical
		"custom.whatever",      // no schema for custom category
		"identity.zodiac_sign", // too different from any canonical field
	}
	for _, id := range tests {
		if s := SuggestCanonical(id); s != nil {
			t.Errorf("SuggestCanonical(%q) = %+v, want nil", id, s)
		}
	}
}

func TestSuggestCanonical_SynonymWrongCategory(t *testing.T) {
	// "name" is a synonym for "full_name", but only in identity category
	s := SuggestCanonical("financial.name")
	if s != nil {
		t.Errorf("SuggestCanonical(financial.name) = %+v, want nil (wrong category)", s)
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
	}
	for _, tt := range tests {
		if got := levenshtein(tt.a, tt.b); got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
