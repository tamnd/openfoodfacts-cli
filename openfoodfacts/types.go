package openfoodfacts

// Product is the output record for a single food product lookup by barcode.
// Nutriment values are formatted as strings with one decimal place
// so they render consistently in table and CSV output.
type Product struct {
	Barcode        string `kit:"id" json:"barcode"`
	Name           string `json:"name"`
	Brands         string `json:"brands"`
	Categories     string `json:"categories"`
	Quantity       string `json:"quantity"`
	NutritionGrade string `json:"nutrition_grade"`
	EcoScore       string `json:"eco_score"`
	EnergyKcal     string `json:"energy_kcal"`
	Fat            string `json:"fat_g"`
	Sugars         string `json:"sugars_g"`
	Proteins       string `json:"proteins_g"`
	Salt           string `json:"salt_g"`
}

// SearchResult is the output record for search and category listings.
type SearchResult struct {
	Barcode        string `kit:"id" json:"barcode"`
	Name           string `json:"name"`
	Brands         string `json:"brands"`
	NutritionGrade string `json:"nutrition_grade"`
	EcoScore       string `json:"eco_score"`
}
