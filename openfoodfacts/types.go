package openfoodfacts

// Nutrients holds per-100g nutritional values for a food product.
type Nutrients struct {
	EnergyKcal float64 `json:"energy_kcal_per_100g,omitempty"`
	Fat        float64 `json:"fat_per_100g,omitempty"`
	SatFat     float64 `json:"saturated_fat_per_100g,omitempty"`
	Carbs      float64 `json:"carbohydrates_per_100g,omitempty"`
	Sugars     float64 `json:"sugars_per_100g,omitempty"`
	Protein    float64 `json:"protein_per_100g,omitempty"`
	Salt       float64 `json:"salt_per_100g,omitempty"`
	Fiber      float64 `json:"fiber_per_100g,omitempty"`
}

// Product is the normalized output record for a food product.
type Product struct {
	Barcode     string     `json:"barcode"`
	Name        string     `json:"name"`
	Brands      string     `json:"brands,omitempty"`
	Categories  string     `json:"categories,omitempty"`
	Ingredients string     `json:"ingredients,omitempty"`
	NutriScore  string     `json:"nutri_score,omitempty"`
	EcoScore    string     `json:"eco_score,omitempty"`
	Nutrients   *Nutrients `json:"nutrients,omitempty"`
}

// unexported: wire types for JSON decode only

type wireNutriments struct {
	EnergyKcal float64 `json:"energy-kcal_100g"`
	Fat        float64 `json:"fat_100g"`
	SatFat     float64 `json:"saturated-fat_100g"`
	Carbs      float64 `json:"carbohydrates_100g"`
	Sugars     float64 `json:"sugars_100g"`
	Proteins   float64 `json:"proteins_100g"`
	Salt       float64 `json:"salt_100g"`
	Fiber      float64 `json:"fiber_100g"`
}

type wireProduct struct {
	Code            string         `json:"code"`
	ProductName     string         `json:"product_name"`
	Brands          string         `json:"brands"`
	Categories      string         `json:"categories"`
	IngredientsText string         `json:"ingredients_text"`
	NutriscoreGrade string         `json:"nutriscore_grade"`
	EcoscoreGrade   string         `json:"ecoscore_grade"`
	Nutriments      wireNutriments `json:"nutriments"`
}

type wireProductResp struct {
	Status        int         `json:"status"`
	StatusVerbose string      `json:"status_verbose"`
	Product       wireProduct `json:"product"`
}

type wireSearchResp struct {
	Count    int           `json:"count"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
	Products []wireProduct `json:"products"`
}
