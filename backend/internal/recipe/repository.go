package recipe

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ErrNotFound est retourné quand une recette est introuvable ou n'appartient pas à l'utilisateur.
var ErrNotFound = errors.New("recette introuvable")

// Repository encapsule toutes les requêtes SQL liées aux recettes.
// L'injection de *sql.DB permet un remplacement facile par un mock en test.
type Repository struct {
	db *sql.DB
}

// NewRepository crée un Repository à partir d'une connexion SQL existante.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ── Lecture ───────────────────────────────────────────────────────────────

// List retourne une page de recettes résumées pour un utilisateur.
// La pagination est cursor-based (ID décroissant) pour être stable sous insertions.
// Si q.Search est non-vide, une recherche FTS5 est effectuée à la place.
func (r *Repository) List(ctx context.Context, userID int64, q ListQuery) (ListResult, error) {
	if q.Limit <= 0 || q.Limit > 100 {
		q.Limit = 20
	}

	// Recherche FTS5
	if q.Search != "" {
		return r.search(ctx, userID, q)
	}

	// ── Requête de liste standard avec curseur ────────────────────────────
	// Curseur = ID de la dernière recette reçue (décodé depuis la chaîne opaque)
	var args []any
	args = append(args, userID)

	where := "WHERE user_id = ?"
	if q.Tag != "" {
		// Filtre par tag : json_each déplie le tableau JSON stocké
		where += " AND EXISTS (SELECT 1 FROM json_each(tags) WHERE value = ?)"
		args = append(args, q.Tag)
	}
	if q.Cursor != "" {
		cursorID, err := decodeCursor(q.Cursor)
		if err == nil {
			where += " AND id < ?"
			args = append(args, cursorID)
		}
	}

	// Comptage total (sans curseur ni tag pour la perf)
	var total int
	_ = r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM recipes WHERE user_id = ?`, userID,
	).Scan(&total)

	// Récupération de limit+1 pour détecter s'il y a une page suivante
	fetchLimit := q.Limit + 1
	args = append(args, fetchLimit)

	query := fmt.Sprintf(`
		SELECT id, name, description, servings, prep_time, cook_time,
		       difficulty, tags, image_url, created_at, updated_at
		FROM recipes
		%s
		ORDER BY id DESC
		LIMIT ?`, where)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return ListResult{}, fmt.Errorf("list recipes: %w", err)
	}
	defer rows.Close()

	var summaries []RecipeSummary
	for rows.Next() {
		s, err := scanSummary(rows)
		if err != nil {
			return ListResult{}, fmt.Errorf("list scan: %w", err)
		}
		summaries = append(summaries, s)
	}
	if err := rows.Err(); err != nil {
		return ListResult{}, fmt.Errorf("list rows: %w", err)
	}

	// Détermination du curseur suivant
	var nextCursor string
	if len(summaries) > q.Limit {
		summaries = summaries[:q.Limit]
		nextCursor = encodeCursor(summaries[len(summaries)-1].ID)
	}

	return ListResult{Data: summaries, NextCursor: nextCursor, Total: total}, nil
}

// search effectue une recherche full-text via FTS5 sur name, description, tags.
func (r *Repository) search(ctx context.Context, userID int64, q ListQuery) (ListResult, error) {
	// Sanitisation de la query FTS5 (évite les requêtes invalides)
	ftsQuery := sanitizeFTSQuery(q.Search)

	// La jointure entre recipes et recipes_fts permet de filtrer par user_id
	rows, err := r.db.QueryContext(ctx, `
		SELECT r.id, r.name, r.description, r.servings, r.prep_time, r.cook_time,
		       r.difficulty, r.tags, r.image_url, r.created_at, r.updated_at
		FROM recipes r
		JOIN recipes_fts fts ON fts.rowid = r.id
		WHERE r.user_id = ?
		  AND recipes_fts MATCH ?
		ORDER BY rank
		LIMIT ?`,
		userID, ftsQuery, q.Limit,
	)
	if err != nil {
		return ListResult{}, fmt.Errorf("search recipes: %w", err)
	}
	defer rows.Close()

	var summaries []RecipeSummary
	for rows.Next() {
		s, err := scanSummary(rows)
		if err != nil {
			return ListResult{}, fmt.Errorf("search scan: %w", err)
		}
		summaries = append(summaries, s)
	}

	return ListResult{Data: summaries, Total: len(summaries)}, nil
}

// GetByID retourne la recette complète (avec ingrédients et étapes).
// Retourne ErrNotFound si la recette n'existe pas ou n'appartient pas à userID.
func (r *Repository) GetByID(ctx context.Context, userID, recipeID int64) (*Recipe, error) {
	recipe := &Recipe{}
	var tagsJSON string
	var description, difficulty, imageURL sql.NullString
	var prepTime, cookTime sql.NullInt64
	var createdAt, updatedAt string

	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, name, description, servings, prep_time, cook_time,
		       difficulty, tags, image_url, created_at, updated_at
		FROM recipes
		WHERE id = ? AND user_id = ?`,
		recipeID, userID,
	).Scan(
		&recipe.ID, &recipe.UserID, &recipe.Name, &description,
		&recipe.Servings, &prepTime, &cookTime,
		&difficulty, &tagsJSON, &imageURL,
		&createdAt, &updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get recipe %d: %w", recipeID, err)
	}

	// Décodage des champs nullable
	recipe.Description = description.String
	recipe.Difficulty  = Difficulty(difficulty.String)
	recipe.ImageURL    = imageURL.String
	if prepTime.Valid { v := int(prepTime.Int64); recipe.PrepTime = &v }
	if cookTime.Valid { v := int(cookTime.Int64); recipe.CookTime = &v }
	recipe.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	recipe.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	if err := json.Unmarshal([]byte(tagsJSON), &recipe.Tags); err != nil {
		recipe.Tags = []string{}
	}

	// Chargement des ingrédients
	recipe.Ingredients, err = r.loadIngredients(ctx, recipeID)
	if err != nil {
		return nil, err
	}

	// Chargement des étapes
	recipe.Steps, err = r.loadSteps(ctx, recipeID)
	if err != nil {
		return nil, err
	}

	return recipe, nil
}

func (r *Repository) loadIngredients(ctx context.Context, recipeID int64) ([]Ingredient, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, recipe_id, name, quantity, unit, sort_order
		FROM recipe_ingredients
		WHERE recipe_id = ?
		ORDER BY sort_order, id`,
		recipeID,
	)
	if err != nil {
		return nil, fmt.Errorf("load ingredients: %w", err)
	}
	defer rows.Close()

	var out []Ingredient
	for rows.Next() {
		var ing Ingredient
		var qty sql.NullFloat64
		if err := rows.Scan(&ing.ID, &ing.RecipeID, &ing.Name, &qty, &ing.Unit, &ing.SortOrder); err != nil {
			return nil, err
		}
		if qty.Valid { ing.Quantity = &qty.Float64 }
		out = append(out, ing)
	}
	return out, rows.Err()
}

func (r *Repository) loadSteps(ctx context.Context, recipeID int64) ([]Step, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, recipe_id, step_order, content
		FROM recipe_steps
		WHERE recipe_id = ?
		ORDER BY step_order`,
		recipeID,
	)
	if err != nil {
		return nil, fmt.Errorf("load steps: %w", err)
	}
	defer rows.Close()

	var out []Step
	for rows.Next() {
		var s Step
		if err := rows.Scan(&s.ID, &s.RecipeID, &s.StepOrder, &s.Content); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ── Écriture ──────────────────────────────────────────────────────────────

// Create insère une nouvelle recette avec ses ingrédients et étapes en une transaction.
func (r *Repository) Create(ctx context.Context, userID int64, input CreateRecipeInput) (*Recipe, error) {
	tagsJSON, err := marshalTags(input.Tags)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("create recipe tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck — rollback no-op si Commit réussi

	// Insertion de la recette
	res, err := tx.ExecContext(ctx, `
		INSERT INTO recipes (user_id, name, description, servings, prep_time, cook_time, difficulty, tags, image_url)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID,
		input.Name,
		nullStr(input.Description),
		input.Servings,
		nullInt(input.PrepTime),
		nullInt(input.CookTime),
		nullStr(string(input.Difficulty)),
		tagsJSON,
		nullStr(input.ImageURL),
	)
	if err != nil {
		return nil, fmt.Errorf("insert recipe: %w", err)
	}
	recipeID, _ := res.LastInsertId()

	// Insertion des ingrédients
	if err := insertIngredients(ctx, tx, recipeID, input.Ingredients); err != nil {
		return nil, err
	}

	// Insertion des étapes
	if err := insertSteps(ctx, tx, recipeID, input.Steps); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("create recipe commit: %w", err)
	}

	return r.GetByID(ctx, userID, recipeID)
}

// Update applique un patch partiel sur une recette existante.
// Si Ingredients ou Steps sont fournis, ils remplacent intégralement les anciens.
func (r *Repository) Update(ctx context.Context, userID, recipeID int64, input UpdateRecipeInput) (*Recipe, error) {
	// Vérification d'appartenance avant toute modification
	existing, err := r.GetByID(ctx, userID, recipeID)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("update recipe tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Construction du SET dynamique (seulement les champs fournis)
	setClauses := []string{"updated_at = datetime('now')"}
	args := []any{}

	if input.Name != nil        { setClauses = append(setClauses, "name = ?");        args = append(args, *input.Name) }
	if input.Description != nil { setClauses = append(setClauses, "description = ?"); args = append(args, *input.Description) }
	if input.Servings != nil    { setClauses = append(setClauses, "servings = ?");    args = append(args, *input.Servings) }
	if input.PrepTime != nil    { setClauses = append(setClauses, "prep_time = ?");   args = append(args, *input.PrepTime) }
	if input.CookTime != nil    { setClauses = append(setClauses, "cook_time = ?");   args = append(args, *input.CookTime) }
	if input.Difficulty != nil  { setClauses = append(setClauses, "difficulty = ?");  args = append(args, string(*input.Difficulty)) }
	if input.ImageURL != nil    { setClauses = append(setClauses, "image_url = ?");   args = append(args, *input.ImageURL) }
	if input.Tags != nil {
		tagsJSON, err := marshalTags(input.Tags)
		if err != nil { return nil, err }
		setClauses = append(setClauses, "tags = ?")
		args = append(args, tagsJSON)
	}

	if len(setClauses) > 1 { // plus que updated_at
		query := fmt.Sprintf("UPDATE recipes SET %s WHERE id = ? AND user_id = ?",
			strings.Join(setClauses, ", "))
		args = append(args, recipeID, userID)
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return nil, fmt.Errorf("update recipe: %w", err)
		}
	}

	// Remplacement complet des ingrédients si fournis
	if input.Ingredients != nil {
		if _, err := tx.ExecContext(ctx, `DELETE FROM recipe_ingredients WHERE recipe_id = ?`, recipeID); err != nil {
			return nil, fmt.Errorf("delete ingredients: %w", err)
		}
		if err := insertIngredients(ctx, tx, recipeID, input.Ingredients); err != nil {
			return nil, err
		}
	}

	// Remplacement complet des étapes si fournies
	if input.Steps != nil {
		if _, err := tx.ExecContext(ctx, `DELETE FROM recipe_steps WHERE recipe_id = ?`, recipeID); err != nil {
			return nil, fmt.Errorf("delete steps: %w", err)
		}
		if err := insertSteps(ctx, tx, recipeID, input.Steps); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("update recipe commit: %w", err)
	}

	_ = existing // utilisé pour la vérification d'appartenance
	return r.GetByID(ctx, userID, recipeID)
}

// Delete supprime une recette et ses dépendances (CASCADE en base).
// Retourne ErrNotFound si la recette n'appartient pas à l'utilisateur.
func (r *Repository) Delete(ctx context.Context, userID, recipeID int64) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM recipes WHERE id = ? AND user_id = ?`, recipeID, userID,
	)
	if err != nil {
		return fmt.Errorf("delete recipe %d: %w", recipeID, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ── Helpers SQL ───────────────────────────────────────────────────────────

func insertIngredients(ctx context.Context, tx *sql.Tx, recipeID int64, items []IngredientInput) error {
	if len(items) == 0 {
		return nil
	}
	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO recipe_ingredients (recipe_id, name, quantity, unit, sort_order) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare ingredients: %w", err)
	}
	defer stmt.Close()

	for i, ing := range items {
		order := ing.SortOrder
		if order == 0 { order = i + 1 }
		if _, err := stmt.ExecContext(ctx, recipeID, ing.Name, ing.Quantity, ing.Unit, order); err != nil {
			return fmt.Errorf("insert ingredient %q: %w", ing.Name, err)
		}
	}
	return nil
}

func insertSteps(ctx context.Context, tx *sql.Tx, recipeID int64, steps []StepInput) error {
	if len(steps) == 0 {
		return nil
	}
	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO recipe_steps (recipe_id, step_order, content) VALUES (?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare steps: %w", err)
	}
	defer stmt.Close()

	for i, s := range steps {
		order := s.StepOrder
		if order == 0 { order = i + 1 }
		if _, err := stmt.ExecContext(ctx, recipeID, order, s.Content); err != nil {
			return fmt.Errorf("insert step %d: %w", order, err)
		}
	}
	return nil
}

// scanSummary lit une ligne de RecipeSummary depuis un *sql.Rows.
func scanSummary(rows *sql.Rows) (RecipeSummary, error) {
	var s RecipeSummary
	var desc, diff, img sql.NullString
	var prep, cook sql.NullInt64
	var tagsJSON, createdAt, updatedAt string

	if err := rows.Scan(
		&s.ID, &s.Name, &desc, &s.Servings, &prep, &cook,
		&diff, &tagsJSON, &img, &createdAt, &updatedAt,
	); err != nil {
		return s, err
	}

	s.Description = desc.String
	s.Difficulty  = Difficulty(diff.String)
	s.ImageURL    = img.String
	if prep.Valid { v := int(prep.Int64); s.PrepTime = &v }
	if cook.Valid { v := int(cook.Int64); s.CookTime = &v }
	s.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	if err := json.Unmarshal([]byte(tagsJSON), &s.Tags); err != nil {
		s.Tags = []string{}
	}
	return s, nil
}

// ── Pagination (curseur opaque = ID encodé en base10) ────────────────────

func encodeCursor(id int64) string {
	return strconv.FormatInt(id, 10)
}

func decodeCursor(cursor string) (int64, error) {
	return strconv.ParseInt(cursor, 10, 64)
}

// ── Helpers de nullabilité ────────────────────────────────────────────────

func nullStr(s string) any {
	if s == "" { return nil }
	return s
}

func nullInt(p *int) any {
	if p == nil { return nil }
	return *p
}

func marshalTags(tags []string) (string, error) {
	if tags == nil { tags = []string{} }
	b, err := json.Marshal(tags)
	if err != nil {
		return "", fmt.Errorf("marshal tags: %w", err)
	}
	return string(b), nil
}

// sanitizeFTSQuery prépare une query FTS5 en ajoutant des guillemets
// autour des termes simples pour éviter les erreurs de syntaxe FTS5.
func sanitizeFTSQuery(q string) string {
	q = strings.TrimSpace(q)
	if q == "" { return `""` }
	// Si la query contient déjà des opérateurs FTS5, on la laisse telle quelle
	if strings.ContainsAny(q, `"*()`) { return q }
	// Sinon on recherche chaque mot de manière préfixe
	terms := strings.Fields(q)
	for i, t := range terms {
		terms[i] = t + "*"
	}
	return strings.Join(terms, " ")
}
