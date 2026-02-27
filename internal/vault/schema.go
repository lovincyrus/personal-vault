package vault

// SchemaField describes a recommended field in the vault schema.
type SchemaField struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Sensitivity string `json:"sensitivity"`
}

// SchemaCategory groups recommended fields under a category.
type SchemaCategory struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Fields      []SchemaField `json:"fields"`
}

// Schema is the full recommended schema for the vault.
type Schema struct {
	Version    string           `json:"version"`
	Categories []SchemaCategory `json:"categories"`
}

// RecommendedSchema is the canonical schema that agents can discover.
var RecommendedSchema = Schema{
	Version: "1",
	Categories: []SchemaCategory{
		{
			Name:        "identity",
			Description: "Personal identity information",
			Fields: []SchemaField{
				{ID: "identity.first_name", Description: "First/given name", Sensitivity: "standard"},
				{ID: "identity.last_name", Description: "Last/family name", Sensitivity: "standard"},
				{ID: "identity.full_name", Description: "Full display name", Sensitivity: "standard"},
				{ID: "identity.email", Description: "Primary email address", Sensitivity: "standard"},
				{ID: "identity.phone", Description: "Phone number", Sensitivity: "sensitive"},
				{ID: "identity.date_of_birth", Description: "Date of birth", Sensitivity: "sensitive"},
				},
		},
		{
			Name:        "addresses",
			Description: "Physical addresses",
			Fields: []SchemaField{
				{ID: "addresses.home_street", Description: "Home street address", Sensitivity: "sensitive"},
				{ID: "addresses.home_city", Description: "Home city", Sensitivity: "standard"},
				{ID: "addresses.home_state", Description: "Home state or province", Sensitivity: "standard"},
				{ID: "addresses.home_zip", Description: "Home ZIP or postal code", Sensitivity: "standard"},
				{ID: "addresses.home_country", Description: "Home country code (e.g. US)", Sensitivity: "standard"},
			},
		},
		{
			Name:        "financial",
			Description: "Financial and tax information",
			Fields: []SchemaField{
				{ID: "financial.filing_status", Description: "Tax filing status", Sensitivity: "sensitive"},
				{ID: "financial.ssn", Description: "Social Security Number", Sensitivity: "critical"},
			},
		},
		{
			Name:        "payment",
			Description: "Payment card details",
			Fields: []SchemaField{
				{ID: "payment.card_number", Description: "Payment card number", Sensitivity: "critical"},
				{ID: "payment.card_expiry", Description: "Card expiration date", Sensitivity: "critical"},
				{ID: "payment.cardholder_name", Description: "Name on payment card", Sensitivity: "critical"},
				{ID: "payment.card_brand", Description: "Card brand (e.g. Visa, Mastercard)", Sensitivity: "standard"},
			},
		},
		{
			Name:        "preferences",
			Description: "User preferences",
			Fields: []SchemaField{
				{ID: "preferences.timezone", Description: "Preferred timezone (e.g. America/New_York)", Sensitivity: "public"},
				{ID: "preferences.language", Description: "Preferred language (e.g. en)", Sensitivity: "public"},
			},
		},
		{
			Name:        "employment",
			Description: "Employment information",
			Fields: []SchemaField{
				{ID: "employment.employer", Description: "Current employer name", Sensitivity: "standard"},
				{ID: "employment.title", Description: "Job title", Sensitivity: "standard"},
			},
		},
		{
			Name:        "medical",
			Description: "Medical information (user-defined fields)",
			Fields:      []SchemaField{},
		},
		{
			Name:        "documents",
			Description: "Document references (user-defined fields)",
			Fields:      []SchemaField{},
		},
	},
}

var schemaIndex map[string]*SchemaField

func init() {
	schemaIndex = make(map[string]*SchemaField)
	for i := range RecommendedSchema.Categories {
		for j := range RecommendedSchema.Categories[i].Fields {
			f := &RecommendedSchema.Categories[i].Fields[j]
			schemaIndex[f.ID] = f
		}
	}
}

// IsCanonicalField returns true if the field ID is in the recommended schema.
func IsCanonicalField(id string) bool {
	_, ok := schemaIndex[id]
	return ok
}

// GetSchemaField returns the schema field for a canonical ID, or nil.
func GetSchemaField(id string) *SchemaField {
	return schemaIndex[id]
}

// DefaultSensitivity returns the schema default sensitivity for a field ID,
// or "standard" if the field is not in the schema.
func DefaultSensitivity(id string) string {
	if f, ok := schemaIndex[id]; ok {
		return f.Sensitivity
	}
	return "standard"
}
