package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

// ErrNotFound est retourné quand un article est introuvable ou n'appartient pas à l'utilisateur.
var ErrNotFound = errors.New("Article introuvable.")

// ErrQuantityBelowZero est retourné quand un ajustement rendrait la quantité négative.
var ErrQuantityBelowZero = errors.New("La quantité ne peut pas être négative.")

// ErrInvalidInput est un sentinel utilisé par les handlers pour détecter
// les erreurs de validation métier déjà formatées en FR. Le message affichable
// provient du fmt.Errorf qui wrappe cet error, pas de la valeur sentinel elle-même.
var ErrInvalidInput = errors.New("invalid input")

// invalidInput construit une erreur "métier" dont le .Error() est directement
// le message FR destiné à l'utilisateur, tout en restant détectable via
// errors.Is(err, ErrInvalidInput) côté handler.
type businessError struct{ msg string }

func (e *businessError) Error() string { return e.msg }
func (e *businessError) Unwrap() error { return ErrInvalidInput }

func invalidInput(msg string) error {
	return &businessError{msg: msg}
}

// Repository encapsule toutes les requêtes SQL liées à l'inventaire.
type Repository struct {
	db *sql.DB
}

// NewRepository crée un Repository à partir d'une connexion SQL existante.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ── Lecture ───────────────────────────────────────────────────────────────

// List retourne la liste paginée des articles d'un utilisateur.
// Les filtres Category, Level et Search sont combinables.
// Le niveau de stock (ok/low/critical) est calculé à la volée.
func (r *Repository) List(ctx context.Context, userID int64, q ListQuery) (ListResult, error) {
	if q.Limit <= 0 || q.Limit > 200 {
		q.Limit = 50
	}

	// ── Construction dynamique de la clause WHERE ─────────────────────────
	conditions := []string{"user_id = ?"}
	args := []any{userID}

	if q.Category != "" {
		conditions = append(conditions, "category = ?")
		args = append(args, q.Category)
	}
	if q.Search != "" {
		conditions = append(conditions, "name LIKE ?")
		args = append(args, "%"+sanitizeLike(q.Search)+"%")
	}

	// Filtre par niveau — traduit en conditions SQL sur quantity et alert_at
	switch q.Level {
	case LevelOK:
		conditions = append(conditions, "quantity > alert_at")
	case LevelLow:
		conditions = append(conditions, "quantity > 0 AND quantity <= alert_at")
	case LevelCritical:
		// Critique = stock vide OU article expiré (date < aujourd'hui)
		conditions = append(conditions,
			"(quantity = 0 OR (expiry IS NOT NULL AND date(expiry) < date('now')))")
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	// ── Comptage total pour la pagination ─────────────────────────────────
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM storage_items %s", where)
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return ListResult{}, fmt.Errorf("count storage: %w", err)
	}

	// ── Requête paginée ───────────────────────────────────────────────────
	listArgs := append(args, q.Limit, q.Offset)
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, user_id, name, quantity, unit, category, expiry, alert_at, notes, created_at, updated_at
		FROM storage_items
		%s
		ORDER BY category ASC, name ASC
		LIMIT ? OFFSET ?`, where),
		listArgs...,
	)
	if err != nil {
		return ListResult{}, fmt.Errorf("list storage: %w", err)
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return ListResult{}, fmt.Errorf("list scan: %w", err)
		}
		item.Level = computeLevel(item)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ListResult{}, fmt.Errorf("list rows: %w", err)
	}

	return ListResult{
		Data:   items,
		Total:  total,
		Offset: q.Offset,
		Limit:  q.Limit,
	}, nil
}

// GetByID retourne un article complet avec son niveau de stock calculé.
// Retourne ErrNotFound si l'article n'existe pas ou appartient à un autre utilisateur.
func (r *Repository) GetByID(ctx context.Context, userID, itemID int64) (*Item, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, name, quantity, unit, category, expiry, alert_at, notes, created_at, updated_at
		FROM storage_items
		WHERE id = ? AND user_id = ?`,
		itemID, userID,
	)

	item, err := scanItemRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get storage item %d: %w", itemID, err)
	}

	item.Level = computeLevel(item)
	return &item, nil
}

// GetAlerts retourne tous les articles dont le stock est faible ou critique,
// triés par criticité décroissante puis par nom.
func (r *Repository) GetAlerts(ctx context.Context, userID int64) ([]Alert, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, quantity, unit, category, alert_at, expiry
		FROM storage_items
		WHERE user_id = ?
		  AND (
		    quantity = 0
		    OR quantity <= alert_at
		    OR (expiry IS NOT NULL AND date(expiry) < date('now'))
		  )
		ORDER BY
		  -- Critique en premier : vide ou expiré
		  CASE WHEN quantity = 0
		            OR (expiry IS NOT NULL AND date(expiry) < date('now'))
		       THEN 0 ELSE 1 END ASC,
		  name ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get alerts: %w", err)
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var a Alert
		var category sql.NullString
		var expiry sql.NullString
		if err := rows.Scan(&a.ID, &a.Name, &a.Quantity, &a.Unit, &category, &a.AlertAt, &expiry); err != nil {
			return nil, fmt.Errorf("alerts scan: %w", err)
		}
		a.Category = category.String
		if expiry.Valid {
			a.Expiry = &expiry.String
		}
		a.Level = computeLevelFromValues(a.Quantity, a.AlertAt, a.Expiry)
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

// GetStats calcule les métriques globales de l'inventaire pour un utilisateur.
func (r *Repository) GetStats(ctx context.Context, userID int64) (Stats, error) {
	var stats Stats

	// Comptages en une seule passe via agrégats conditionnels.
	//
	// attention_count = union distincte des articles nécessitant une action,
	// c'est-à-dire ceux dont le stock est trop bas (low OU critical) OU dont
	// la péremption est dans les 7 jours (ou déjà dépassée). Formulation en
	// négation positive pour éviter le double comptage d'un article qui
	// cumule plusieurs raisons : on compte "pas tranquille" = PAS (quantité
	// largement suffisante ET péremption lointaine/inexistante).
	err := r.db.QueryRowContext(ctx, `
		SELECT
		  COUNT(*) AS total,
		  SUM(CASE WHEN quantity > alert_at
		            AND (expiry IS NULL OR date(expiry) >= date('now'))
		       THEN 1 ELSE 0 END) AS ok_count,
		  SUM(CASE WHEN quantity > 0 AND quantity <= alert_at
		            AND (expiry IS NULL OR date(expiry) >= date('now'))
		       THEN 1 ELSE 0 END) AS low_count,
		  SUM(CASE WHEN quantity = 0
		            OR (expiry IS NOT NULL AND date(expiry) < date('now'))
		       THEN 1 ELSE 0 END) AS critical_count,
		  SUM(CASE WHEN expiry IS NOT NULL
		            AND date(expiry) >= date('now')
		            AND date(expiry) <= date('now', '+7 days')
		       THEN 1 ELSE 0 END) AS expiring_count,
		  SUM(CASE WHEN NOT (
		              quantity > alert_at
		              AND (expiry IS NULL OR date(expiry) > date('now', '+7 days'))
		            )
		       THEN 1 ELSE 0 END) AS attention_count
		FROM storage_items
		WHERE user_id = ?`,
		userID,
	).Scan(&stats.Total, &stats.OKCount, &stats.LowCount, &stats.CriticalCount, &stats.ExpiringCount, &stats.AttentionCount)
	if err != nil {
		return Stats{}, fmt.Errorf("stats: %w", err)
	}

	// Liste des catégories distinctes (triées)
	cats, err := r.getCategories(ctx, userID)
	if err != nil {
		return Stats{}, err
	}
	stats.Categories = cats

	return stats, nil
}

func (r *Repository) getCategories(ctx context.Context, userID int64) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT category FROM storage_items
		WHERE user_id = ? AND category != ''
		ORDER BY category ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("categories: %w", err)
	}
	defer rows.Close()

	var cats []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

// GetShoppingList retourne la liste de courses calculée automatiquement :
// tous les articles en alerte, avec la quantité manquante (AlertAt - Quantity).
func (r *Repository) GetShoppingList(ctx context.Context, userID int64) ([]ShoppingEntry, error) {
	alerts, err := r.GetAlerts(ctx, userID)
	if err != nil {
		return nil, err
	}

	var list []ShoppingEntry
	for _, a := range alerts {
		need := math.Max(0, a.AlertAt-a.Quantity)
		// Si critique (vide ou expiré) sans seuil défini, on met 1 unité symbolique
		if need == 0 && a.Level == LevelCritical {
			need = 1
		}
		list = append(list, ShoppingEntry{
			ItemID:   a.ID,
			Name:     a.Name,
			Need:     need,
			Unit:     a.Unit,
			Category: a.Category,
			Level:    a.Level,
		})
	}
	return list, nil
}

// ── Écriture ──────────────────────────────────────────────────────────────

// Create insère un nouvel article d'inventaire.
func (r *Repository) Create(ctx context.Context, userID int64, input CreateItemInput) (*Item, error) {
	if err := validateExpiry(input.Expiry); err != nil {
		return nil, err
	}

	// Le schéma SQL impose category/unit NOT NULL DEFAULT '' : on stocke
	// une chaîne vide quand l'utilisateur n'a rien renseigné, plutôt que NULL
	// (qui ferait échouer la contrainte → "erreur interne" côté front).
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO storage_items (user_id, name, quantity, unit, category, expiry, alert_at, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		userID,
		input.Name,
		input.Quantity,
		input.Unit,
		input.Category,   // "" autorisé (NOT NULL DEFAULT '')
		input.Expiry,     // *string — nil → NULL en base (colonne nullable)
		input.AlertAt,
		nullStr(input.Notes),
	)
	if err != nil {
		return nil, fmt.Errorf("create storage item: %w", err)
	}

	id, _ := res.LastInsertId()
	return r.GetByID(ctx, userID, id)
}

// Update applique un patch partiel sur un article.
// Seuls les champs non-nil du payload sont modifiés.
// Expiry à nil dans le payload = suppression de la date d'expiration.
func (r *Repository) Update(ctx context.Context, userID, itemID int64, input UpdateItemInput) (*Item, error) {
	// Vérification d'appartenance
	if _, err := r.GetByID(ctx, userID, itemID); err != nil {
		return nil, err
	}

	if input.Expiry != nil && *input.Expiry != "" {
		if err := validateExpiry(input.Expiry); err != nil {
			return nil, err
		}
	}

	setClauses := []string{"updated_at = datetime('now')"}
	args := []any{}

	if input.Name     != nil { setClauses = append(setClauses, "name = ?");     args = append(args, *input.Name) }
	if input.Quantity != nil { setClauses = append(setClauses, "quantity = ?"); args = append(args, *input.Quantity) }
	if input.Unit     != nil { setClauses = append(setClauses, "unit = ?");     args = append(args, *input.Unit) }
	if input.Category != nil { setClauses = append(setClauses, "category = ?"); args = append(args, *input.Category) }
	if input.AlertAt  != nil { setClauses = append(setClauses, "alert_at = ?"); args = append(args, *input.AlertAt) }
	if input.Notes    != nil { setClauses = append(setClauses, "notes = ?");    args = append(args, *input.Notes) }

	// Champ Expiry : présent dans le payload (même si vide string = supprimer)
	// On détecte la présence via un champ séparé dans le JSON — ici *string suffit.
	// Si *input.Expiry == "" → on passe NULL, sinon on passe la valeur.
	if input.Expiry != nil {
		if *input.Expiry == "" {
			setClauses = append(setClauses, "expiry = NULL")
		} else {
			setClauses = append(setClauses, "expiry = ?")
			args = append(args, *input.Expiry)
		}
	}

	if len(setClauses) == 1 {
		// Aucun champ à modifier — on retourne l'existant tel quel
		return r.GetByID(ctx, userID, itemID)
	}

	query := fmt.Sprintf("UPDATE storage_items SET %s WHERE id = ? AND user_id = ?",
		strings.Join(setClauses, ", "))
	args = append(args, itemID, userID)

	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return nil, fmt.Errorf("update storage item %d: %w", itemID, err)
	}

	return r.GetByID(ctx, userID, itemID)
}

// AdjustQuantity incrémente ou décrémente atomiquement la quantité d'un article.
// Retourne ErrQuantityBelowZero si le résultat serait négatif.
func (r *Repository) AdjustQuantity(ctx context.Context, userID, itemID int64, delta float64) (*Item, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("adjust quantity tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Lecture et verrouillage de la quantité courante
	var current float64
	err = tx.QueryRowContext(ctx,
		`SELECT quantity FROM storage_items WHERE id = ? AND user_id = ?`,
		itemID, userID,
	).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("adjust quantity read: %w", err)
	}

	newQty := current + delta
	if newQty < 0 {
		return nil, ErrQuantityBelowZero
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE storage_items SET quantity = ?, updated_at = datetime('now') WHERE id = ? AND user_id = ?`,
		newQty, itemID, userID,
	); err != nil {
		return nil, fmt.Errorf("adjust quantity update: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("adjust quantity commit: %w", err)
	}

	return r.GetByID(ctx, userID, itemID)
}

// Delete supprime un article de l'inventaire.
// Retourne ErrNotFound si l'article n'appartient pas à l'utilisateur.
func (r *Repository) Delete(ctx context.Context, userID, itemID int64) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM storage_items WHERE id = ? AND user_id = ?`, itemID, userID,
	)
	if err != nil {
		return fmt.Errorf("delete storage item %d: %w", itemID, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ── Helpers scan ──────────────────────────────────────────────────────────

// itemScanner est une interface commune à *sql.Row et *sql.Rows.
type itemScanner interface {
	Scan(dest ...any) error
}

func scanItem(rows *sql.Rows) (Item, error) {
	return scanItemScanner(rows)
}

func scanItemRow(row *sql.Row) (Item, error) {
	return scanItemScanner(row)
}

func scanItemScanner(s itemScanner) (Item, error) {
	var item Item
	var category, expiry, notes sql.NullString
	var createdAt, updatedAt string

	if err := s.Scan(
		&item.ID, &item.UserID, &item.Name,
		&item.Quantity, &item.Unit,
		&category, &expiry, &item.AlertAt, &notes,
		&createdAt, &updatedAt,
	); err != nil {
		return item, err
	}

	item.Category = category.String
	item.Notes = notes.String
	if expiry.Valid {
		item.Expiry = &expiry.String
	}
	item.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	item.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return item, nil
}

// ── Calcul du niveau de stock ─────────────────────────────────────────────

// computeLevel détermine le StockLevel d'un Item.
func computeLevel(item Item) StockLevel {
	return computeLevelFromValues(item.Quantity, item.AlertAt, item.Expiry)
}

// computeLevelFromValues est la logique pure, testable sans Item.
func computeLevelFromValues(quantity, alertAt float64, expiry *string) StockLevel {
	// Expiré = critique
	if expiry != nil && *expiry != "" {
		exp, err := time.Parse("2006-01-02", *expiry)
		if err == nil && exp.Before(time.Now().Truncate(24*time.Hour)) {
			return LevelCritical
		}
	}
	if quantity <= 0 {
		return LevelCritical
	}
	if alertAt > 0 && quantity <= alertAt {
		return LevelLow
	}
	return LevelOK
}

// ── Validation ────────────────────────────────────────────────────────────

// validateExpiry vérifie que la date d'expiration est au format ISO "2006-01-02".
// Retourne ErrInvalidInput (wrappé) avec un message FR utilisateur si le format est mauvais.
func validateExpiry(expiry *string) error {
	if expiry == nil || *expiry == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", *expiry); err != nil {
		return invalidInput("La date d'expiration doit être au format JJ/MM/AAAA.")
	}
	return nil
}

// sanitizeLike échappe les caractères spéciaux LIKE pour SQLite.
func sanitizeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
