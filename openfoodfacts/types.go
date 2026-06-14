package openfoodfacts

// Nutriments holds per-100g nutritional values for a food product.
type Nutriments struct {
	EnergyKcal100g    float64 `json:"energy-kcal_100g"`
	Fat100g           float64 `json:"fat_100g"`
	Carbohydrates100g float64 `json:"carbohydrates_100g"`
	Proteins100g      float64 `json:"proteins_100g"`
	Salt100g          float64 `json:"salt_100g"`
	Sugars100g        float64 `json:"sugars_100g"`
}

// Product is the normalized output record for a food product.
type Product struct {
	Barcode    string     `kit:"id" json:"code"`
	Name       string     `json:"product_name"`
	Brands     string     `json:"brands"`
	Categories string     `json:"categories"`
	ImageURL   string     `json:"image_url"`
	NutriScore string     `json:"nutriscore_grade"`
	NovaGroup  int        `json:"nova_group"`
	Nutriments Nutriments `json:"nutriments"`
}
