package storage

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/pierreburbaud/cooking-home/internal/middleware"
)

// RegisterRoutes enregistre l'ensemble des routes de l'inventaire sur le groupe fourni.
// Le groupe doit être protégé par le middleware JWTAuth.
//
//	GET    /storage                — liste paginée + filtres (?category=, ?level=, ?search=, ?limit=, ?offset=)
//	POST   /storage                — création d'un article
//	GET    /storage/stats          — métriques globales (totaux, niveaux, catégories)
//	GET    /storage/alerts         — articles faibles + critiques + expirés
//	GET    /storage/shopping-list  — liste de courses générée automatiquement
//	GET    /storage/:id            — détail d'un article
//	PATCH  /storage/:id            — mise à jour partielle
//	DELETE /storage/:id            — suppression
//	PATCH  /storage/:id/quantity   — ajustement atomique de la quantité (±delta)
func RegisterRoutes(r *gin.RouterGroup, repo *Repository) {
	g := r.Group("/storage")

	// Routes sans paramètre d'ID — déclarées AVANT /:id pour éviter les conflits de routing
	g.GET("",              listHandler(repo))
	g.POST("",             createHandler(repo))
	g.GET("/stats",        statsHandler(repo))
	g.GET("/alerts",       alertsHandler(repo))
	g.GET("/shopping-list", shoppingListHandler(repo))

	// Routes avec paramètre d'ID
	g.GET("/:id",           getHandler(repo))
	g.PATCH("/:id",         updateHandler(repo))
	g.DELETE("/:id",        deleteHandler(repo))
	g.PATCH("/:id/quantity", adjustQuantityHandler(repo))
}

// ── GET /storage ──────────────────────────────────────────────────────────

// listHandler retourne la liste paginée de l'inventaire.
// Paramètres de query :
//   - category : filtre exact sur la catégorie (ex: "Féculents")
//   - level    : filtre par niveau — "ok" | "low" | "critical"
//   - search   : recherche partielle sur le nom (insensible à la casse)
//   - limit    : nombre de résultats (1–200, défaut 50)
//   - offset   : décalage pour la pagination offset-based
func listHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.UserIDFromCtx(c)

		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		if limit < 1  { limit = 1 }
		if limit > 200 { limit = 200 }
		if offset < 0 { offset = 0 }

		q := ListQuery{
			Category: c.Query("category"),
			Level:    StockLevel(c.Query("level")),
			Search:   c.Query("search"),
			Limit:    limit,
			Offset:   offset,
		}

		// Validation du filtre level
		if q.Level != "" && q.Level != LevelOK && q.Level != LevelLow && q.Level != LevelCritical {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "level invalide — valeurs acceptées : ok, low, critical",
			})
			return
		}

		result, err := repo.List(c.Request.Context(), userID, q)
		if err != nil {
			log.Error().Err(err).Int64("user_id", userID).Msg("list storage: erreur repository")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

// ── GET /storage/stats ────────────────────────────────────────────────────

// statsHandler retourne les métriques globales de l'inventaire :
// total d'articles, répartition ok/low/critical, articles expirant bientôt,
// et liste des catégories.
func statsHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.UserIDFromCtx(c)

		stats, err := repo.GetStats(c.Request.Context(), userID)
		if err != nil {
			log.Error().Err(err).Int64("user_id", userID).Msg("storage stats: erreur repository")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
			return
		}

		c.JSON(http.StatusOK, stats)
	}
}

// ── GET /storage/alerts ───────────────────────────────────────────────────

// alertsHandler retourne les articles dont le stock est faible ou critique,
// triés par criticité décroissante. Inclut les articles expirés.
func alertsHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.UserIDFromCtx(c)

		alerts, err := repo.GetAlerts(c.Request.Context(), userID)
		if err != nil {
			log.Error().Err(err).Int64("user_id", userID).Msg("storage alerts: erreur repository")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
			return
		}

		// Réponse toujours un tableau (jamais null)
		if alerts == nil {
			alerts = []Alert{}
		}

		c.JSON(http.StatusOK, gin.H{
			"data":  alerts,
			"count": len(alerts),
		})
	}
}

// ── GET /storage/shopping-list ────────────────────────────────────────────

// shoppingListHandler retourne la liste de courses calculée automatiquement :
// pour chaque article en alerte, la quantité manquante est calculée (alertAt - quantity).
func shoppingListHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.UserIDFromCtx(c)

		list, err := repo.GetShoppingList(c.Request.Context(), userID)
		if err != nil {
			log.Error().Err(err).Int64("user_id", userID).Msg("shopping list: erreur repository")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
			return
		}

		if list == nil {
			list = []ShoppingEntry{}
		}

		c.JSON(http.StatusOK, gin.H{
			"data":  list,
			"count": len(list),
		})
	}
}

// ── POST /storage ─────────────────────────────────────────────────────────

// createHandler crée un nouvel article dans l'inventaire.
// Retourne l'article créé avec son niveau de stock calculé (201 Created).
func createHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.UserIDFromCtx(c)

		var input CreateItemInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		item, err := repo.Create(c.Request.Context(), userID, input)
		if err != nil {
			log.Error().Err(err).Int64("user_id", userID).Str("name", input.Name).Msg("create storage item: erreur repository")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
			return
		}

		log.Info().
			Int64("user_id", userID).
			Int64("item_id", item.ID).
			Str("name", item.Name).
			Msg("storage item créé")

		c.JSON(http.StatusCreated, item)
	}
}

// ── GET /storage/:id ──────────────────────────────────────────────────────

// getHandler retourne le détail complet d'un article avec son niveau de stock.
func getHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.UserIDFromCtx(c)

		itemID, err := parseID(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id invalide"})
			return
		}

		item, err := repo.GetByID(c.Request.Context(), userID, itemID)
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "article introuvable"})
			return
		}
		if err != nil {
			log.Error().Err(err).Int64("item_id", itemID).Msg("get storage item: erreur repository")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
			return
		}

		c.JSON(http.StatusOK, item)
	}
}

// ── PATCH /storage/:id ────────────────────────────────────────────────────

// updateHandler applique un patch partiel sur un article.
// Seuls les champs présents dans le JSON sont modifiés.
// Pour supprimer la date d'expiration, passer "expiry": "".
func updateHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.UserIDFromCtx(c)

		itemID, err := parseID(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id invalide"})
			return
		}

		var input UpdateItemInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		item, err := repo.Update(c.Request.Context(), userID, itemID, input)
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "article introuvable"})
			return
		}
		if err != nil {
			log.Error().Err(err).Int64("item_id", itemID).Msg("update storage item: erreur repository")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		log.Info().
			Int64("user_id", userID).
			Int64("item_id", itemID).
			Msg("storage item mis à jour")

		c.JSON(http.StatusOK, item)
	}
}

// ── DELETE /storage/:id ───────────────────────────────────────────────────

// deleteHandler supprime un article de l'inventaire.
func deleteHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.UserIDFromCtx(c)

		itemID, err := parseID(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id invalide"})
			return
		}

		if err := repo.Delete(c.Request.Context(), userID, itemID); errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "article introuvable"})
			return
		} else if err != nil {
			log.Error().Err(err).Int64("item_id", itemID).Msg("delete storage item: erreur repository")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
			return
		}

		log.Info().
			Int64("user_id", userID).
			Int64("item_id", itemID).
			Msg("storage item supprimé")

		c.Status(http.StatusNoContent)
	}
}

// ── PATCH /storage/:id/quantity ───────────────────────────────────────────

// adjustQuantityHandler incrémente ou décrémente atomiquement la quantité.
// Payload : { "delta": -100.0 }  (négatif = consommation, positif = réapprovisionnement)
// Retourne 422 si le résultat serait négatif.
func adjustQuantityHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.UserIDFromCtx(c)

		itemID, err := parseID(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id invalide"})
			return
		}

		var input AdjustQuantityInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		item, err := repo.AdjustQuantity(c.Request.Context(), userID, itemID, input.Delta)
		switch {
		case errors.Is(err, ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "article introuvable"})
		case errors.Is(err, ErrQuantityBelowZero):
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": "impossible — la quantité résultante serait négative",
			})
		case err != nil:
			log.Error().Err(err).Int64("item_id", itemID).Float64("delta", input.Delta).
				Msg("adjust quantity: erreur repository")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
		default:
			log.Info().
				Int64("user_id", userID).
				Int64("item_id", itemID).
				Float64("delta", input.Delta).
				Float64("new_qty", item.Quantity).
				Msg("quantité ajustée")
			c.JSON(http.StatusOK, item)
		}
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────

func parseID(c *gin.Context, param string) (int64, error) {
	raw := c.Param(param)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("id doit être un entier positif")
	}
	return id, nil
}
