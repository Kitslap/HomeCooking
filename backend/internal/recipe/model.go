// Package recipe expose le CRUD des recettes, la recherche FTS5,
// et la gestion des ingrédients / étapes associés.
package recipe

import "time"

// ── Modèles de domaine ────────────────────────────────────────────────────

// Difficulty représente le niveau de difficulté d'une recette.
type Difficulty string

const (
	DifficultyEasy   Difficulty = "facile"
	DifficultyMedium Difficulty = "moyen"
	DifficultyHard   Difficulty = "difficile"
)

// Recipe est la représentation complète d'une recette en base.
type Recipe struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Servings    int        `json:"servings"`
	PrepTime    *int       `json:"prep_time,omitempty"` // minutes, nullable
	CookTime    *int       `json:"cook_time,omitempty"` // minutes, nullable
	Difficulty  Difficulty `json:"difficulty,omitempty"`
	Tags        []string   `json:"tags"`
	ImageURL    string     `json:"image_url,omitempty"`
	Ingredients []Ingredient `json:"ingredients,omitempty"`
	Steps       []Step       `json:"steps,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// RecipeSummary est la représentation allégée utilisée dans les listings.
// Les ingrédients et étapes ne sont pas chargés pour économiser les requêtes.
type RecipeSummary struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Servings    int        `json:"servings"`
	PrepTime    *int       `json:"prep_time,omitempty"`
	CookTime    *int       `json:"cook_time,omitempty"`
	Difficulty  Difficulty `json:"difficulty,omitempty"`
	Tags        []string   `json:"tags"`
	ImageURL    string     `json:"image_url,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Ingredient est un composant d'une recette.
type Ingredient struct {
	ID        int64   `json:"id"`
	RecipeID  int64   `json:"recipe_id"`
	Name      string  `json:"name"`
	Quantity  *float64 `json:"quantity,omitempty"`
	Unit      string  `json:"unit,omitempty"`
	SortOrder int     `json:"sort_order"`
}

// Step est une étape de préparation d'une recette.
type Step struct {
	ID        int64  `json:"id"`
	RecipeID  int64  `json:"recipe_id"`
	StepOrder int    `json:"step_order"`
	Content   string `json:"content"`
}

// ── Payloads d'entrée (validation via binding Gin) ────────────────────────

// IngredientInput est le payload pour créer/modifier un ingrédient.
type IngredientInput struct {
	Name      string   `json:"name"     binding:"required,min=1,max=120"`
	Quantity  *float64 `json:"quantity"`
	Unit      string   `json:"unit"     binding:"max=20"`
	SortOrder int      `json:"sort_order"`
}

// StepInput est le payload pour créer/modifier une étape.
type StepInput struct {
	StepOrder int    `json:"step_order" binding:"required,min=1"`
	Content   string `json:"content"    binding:"required,min=1,max=2000"`
}

// CreateRecipeInput est le payload pour POST /recipes.
type CreateRecipeInput struct {
	Name        string            `json:"name"        binding:"required,min=2,max=120"`
	Description string            `json:"description" binding:"max=2000"`
	Servings    int               `json:"servings"    binding:"required,min=1,max=50"`
	PrepTime    *int              `json:"prep_time"   binding:"omitempty,min=0,max=1440"`
	CookTime    *int              `json:"cook_time"   binding:"omitempty,min=0,max=1440"`
	Difficulty  Difficulty        `json:"difficulty"  binding:"omitempty,oneof=facile moyen difficile"`
	Tags        []string          `json:"tags"        binding:"omitempty,max=10,dive,max=30"`
	ImageURL    string            `json:"image_url"   binding:"omitempty,url"`
	Ingredients []IngredientInput `json:"ingredients" binding:"omitempty,max=100,dive"`
	Steps       []StepInput       `json:"steps"       binding:"omitempty,max=50,dive"`
}

// UpdateRecipeInput est le payload pour PATCH /recipes/:id.
// Tous les champs sont optionnels (patch partiel).
type UpdateRecipeInput struct {
	Name        *string           `json:"name"        binding:"omitempty,min=2,max=120"`
	Description *string           `json:"description" binding:"omitempty,max=2000"`
	Servings    *int              `json:"servings"    binding:"omitempty,min=1,max=50"`
	PrepTime    *int              `json:"prep_time"   binding:"omitempty,min=0,max=1440"`
	CookTime    *int              `json:"cook_time"   binding:"omitempty,min=0,max=1440"`
	Difficulty  *Difficulty       `json:"difficulty"  binding:"omitempty,oneof=facile moyen difficile"`
	Tags        []string          `json:"tags"        binding:"omitempty,max=10,dive,max=30"`
	ImageURL    *string           `json:"image_url"   binding:"omitempty,url"`
	// Remplacement complet des ingrédients/étapes si présents dans le payload
	Ingredients []IngredientInput `json:"ingredients" binding:"omitempty,max=100,dive"`
	Steps       []StepInput       `json:"steps"       binding:"omitempty,max=50,dive"`
}

// ListQuery regroupe les paramètres de pagination et filtrage de la liste.
type ListQuery struct {
	Limit  int    // Nombre max de résultats (défaut 20, max 100)
	Cursor string // Curseur opaque pour la pagination (ID de la dernière recette)
	Search string // Recherche FTS5 (optionnel)
	Tag    string // Filtre par tag (optionnel)
}

// ListResult est la réponse de la liste paginée.
type ListResult struct {
	Data       []RecipeSummary `json:"data"`
	NextCursor string          `json:"next_cursor,omitempty"` // vide = dernière page
	Total      int             `json:"total"`                 // total estimé (sans filtre search)
}
