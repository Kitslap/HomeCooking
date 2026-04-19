// Package storage expose le CRUD de l'inventaire alimentaire,
// la détection des stocks faibles/critiques, le suivi des dates d'expiration
// et la génération automatique d'une liste de courses.
package storage

import "time"

// ── Niveaux de stock ──────────────────────────────────────────────────────

// StockLevel représente l'état du stock d'un article.
type StockLevel string

const (
	LevelOK       StockLevel = "ok"       // quantité > seuil d'alerte
	LevelLow      StockLevel = "low"      // quantité ≤ seuil d'alerte et > 0
	LevelCritical StockLevel = "critical" // quantité = 0 ou expiré
)

// ── Modèle de domaine ─────────────────────────────────────────────────────

// Item représente un article de l'inventaire en base.
type Item struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	Name      string     `json:"name"`
	Quantity  float64    `json:"quantity"`
	Unit      string     `json:"unit"`
	Category  string     `json:"category,omitempty"`
	Expiry    *string    `json:"expiry,omitempty"`    // ISO date "2026-04-30", nullable
	AlertAt   float64    `json:"alert_at"`            // seuil bas (même unité que Quantity)
	Notes     string     `json:"notes,omitempty"`
	Level     StockLevel `json:"level"`               // calculé, non stocké en base
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Alert est un article dont le stock est faible ou critique.
// Utilisé dans les endpoints /storage/alerts et /storage/shopping-list.
type Alert struct {
	ID       int64      `json:"id"`
	Name     string     `json:"name"`
	Quantity float64    `json:"quantity"`
	Unit     string     `json:"unit"`
	Category string     `json:"category,omitempty"`
	AlertAt  float64    `json:"alert_at"`
	Expiry   *string    `json:"expiry,omitempty"`
	Level    StockLevel `json:"level"`
}

// Stats regroupe les métriques d'inventaire de l'utilisateur.
type Stats struct {
	Total          int      `json:"total"`           // nombre total d'articles
	OKCount        int      `json:"ok_count"`        // articles OK
	LowCount       int      `json:"low_count"`       // articles faibles
	CriticalCount  int      `json:"critical_count"`  // articles critiques (vides ou expirés)
	ExpiringCount  int      `json:"expiring_count"`  // articles expirant dans les 7 jours
	AttentionCount int      `json:"attention_count"` // union distincte : low ∪ critical ∪ expiring (articles nécessitant une action)
	Categories     []string `json:"categories"`      // liste des catégories distinctes
}

// ShoppingEntry est une ligne de liste de courses générée automatiquement.
type ShoppingEntry struct {
	ItemID   int64   `json:"item_id"`
	Name     string  `json:"name"`
	Need     float64 `json:"need"`     // quantité manquante (AlertAt - Quantity, min 0)
	Unit     string  `json:"unit"`
	Category string  `json:"category,omitempty"`
	Level    StockLevel `json:"level"`
}

// ── Payloads d'entrée ─────────────────────────────────────────────────────

// CreateItemInput est le payload pour POST /storage.
type CreateItemInput struct {
	Name     string  `json:"name"     binding:"required,min=1,max=120"`
	Quantity float64 `json:"quantity" binding:"required,min=0"`
	Unit     string  `json:"unit"     binding:"required,min=1,max=20"`
	Category string  `json:"category" binding:"omitempty,max=50"`
	Expiry   *string `json:"expiry"   binding:"omitempty"`    // "2026-04-30"
	AlertAt  float64 `json:"alert_at" binding:"min=0"`
	Notes    string  `json:"notes"    binding:"omitempty,max=500"`
}

// UpdateItemInput est le payload pour PATCH /storage/:id.
// Tous les champs sont optionnels (patch partiel).
type UpdateItemInput struct {
	Name     *string  `json:"name"     binding:"omitempty,min=1,max=120"`
	Quantity *float64 `json:"quantity" binding:"omitempty,min=0"`
	Unit     *string  `json:"unit"     binding:"omitempty,min=1,max=20"`
	Category *string  `json:"category" binding:"omitempty,max=50"`
	Expiry   *string  `json:"expiry"`                              // null = supprimer l'expiration
	AlertAt  *float64 `json:"alert_at" binding:"omitempty,min=0"`
	Notes    *string  `json:"notes"    binding:"omitempty,max=500"`
}

// AdjustQuantityInput est le payload pour PATCH /storage/:id/quantity.
// Permet d'incrémenter ou décrémenter sans connaître la valeur actuelle.
type AdjustQuantityInput struct {
	// Delta peut être positif (réapprovisionnement) ou négatif (consommation).
	Delta float64 `json:"delta" binding:"required"`
}

// ListQuery regroupe les paramètres de filtrage et pagination de la liste.
type ListQuery struct {
	Category string     // filtre par catégorie (optionnel)
	Level    StockLevel // filtre par niveau de stock (optionnel)
	Search   string     // recherche par nom (LIKE, optionnel)
	Limit    int        // 1–200, défaut 50
	Offset   int        // pagination offset-based (liste stable contrairement aux recettes)
}

// ListResult est la réponse de la liste paginée.
type ListResult struct {
	Data   []Item `json:"data"`
	Total  int    `json:"total"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}
