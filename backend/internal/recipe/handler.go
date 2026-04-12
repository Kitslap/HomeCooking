package recipe

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/pierreburbaud/cooking-home/internal/middleware"
)

// RegisterRoutes enregistre l'ensemble des routes CRUD sur le groupe fourni.
// Le groupe doit être protégé par le middleware JWTAuth.
//
//	GET    /recipes          — liste paginée + recherche FTS5 (?search=, ?tag=, ?cursor=, ?limit=)
//	POST   /recipes          — création complète (recette + ingrédients + étapes)
//	GET    /recipes/:id      — détail complet
//	PATCH  /recipes/:id      — mise à jour partielle (patch)
//	DELETE /recipes/:id      — suppression (cascade ingrédients + étapes)
func RegisterRoutes(r *gin.RouterGroup, repo *Repository) {
	g := r.Group("/recipes")
	g.GET("",     listHandler(repo))
	g.POST("",    createHandler(repo))
	g.GET("/:id", getHandler(repo))
	g.PATCH("/:id", updateHandler(repo))
	g.DELETE("/:id", deleteHandler(repo))
}

// ── GET /recipes ──────────────────────────────────────────────────────────

// listHandler retourne la liste paginée des recettes de l'utilisateur authentifié.
// Paramètres de query :
//   - search  : recherche FTS5 sur nom, description, tags
//   - tag     : filtre exact sur un tag (ex: ?tag=végé)
//   - cursor  : curseur de pagination opaque (retourné dans next_cursor)
//   - limit   : nombre de résultats (1–100, défaut 20)
func listHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.UserIDFromCtx(c)

		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		if limit < 1 { limit = 1 }
		if limit > 100 { limit = 100 }

		q := ListQuery{
			Limit:  limit,
			Cursor: c.Query("cursor"),
			Search: c.Query("search"),
			Tag:    c.Query("tag"),
		}

		result, err := repo.List(c.Request.Context(), userID, q)
		if err != nil {
			log.Error().Err(err).Int64("user_id", userID).Msg("list recipes: erreur repository")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

// ── POST /recipes ─────────────────────────────────────────────────────────

// createHandler crée une nouvelle recette avec ses ingrédients et étapes.
// Retourne la recette complète créée (201 Created).
func createHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.UserIDFromCtx(c)

		var input CreateRecipeInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		recipe, err := repo.Create(c.Request.Context(), userID, input)
		if err != nil {
			log.Error().Err(err).Int64("user_id", userID).Str("name", input.Name).Msg("create recipe: erreur repository")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
			return
		}

		log.Info().
			Int64("user_id", userID).
			Int64("recipe_id", recipe.ID).
			Str("name", recipe.Name).
			Msg("recipe créée")

		c.JSON(http.StatusCreated, recipe)
	}
}

// ── GET /recipes/:id ──────────────────────────────────────────────────────

// getHandler retourne le détail complet d'une recette (ingrédients + étapes inclus).
// Retourne 404 si la recette n'existe pas ou n'appartient pas à l'utilisateur.
func getHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.UserIDFromCtx(c)

		recipeID, err := parseID(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id invalide"})
			return
		}

		recipe, err := repo.GetByID(c.Request.Context(), userID, recipeID)
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "recette introuvable"})
			return
		}
		if err != nil {
			log.Error().Err(err).Int64("recipe_id", recipeID).Msg("get recipe: erreur repository")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
			return
		}

		c.JSON(http.StatusOK, recipe)
	}
}

// ── PATCH /recipes/:id ────────────────────────────────────────────────────

// updateHandler applique un patch partiel sur une recette.
// Seuls les champs présents dans le JSON sont modifiés.
// Si "ingredients" ou "steps" sont fournis, ils remplacent intégralement les anciens.
func updateHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.UserIDFromCtx(c)

		recipeID, err := parseID(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id invalide"})
			return
		}

		var input UpdateRecipeInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		recipe, err := repo.Update(c.Request.Context(), userID, recipeID, input)
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "recette introuvable"})
			return
		}
		if err != nil {
			log.Error().Err(err).Int64("recipe_id", recipeID).Msg("update recipe: erreur repository")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
			return
		}

		log.Info().
			Int64("user_id", userID).
			Int64("recipe_id", recipeID).
			Msg("recipe mise à jour")

		c.JSON(http.StatusOK, recipe)
	}
}

// ── DELETE /recipes/:id ───────────────────────────────────────────────────

// deleteHandler supprime une recette et toutes ses dépendances (CASCADE).
// Retourne 204 No Content en cas de succès.
func deleteHandler(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.UserIDFromCtx(c)

		recipeID, err := parseID(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id invalide"})
			return
		}

		if err := repo.Delete(c.Request.Context(), userID, recipeID); errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "recette introuvable"})
			return
		} else if err != nil {
			log.Error().Err(err).Int64("recipe_id", recipeID).Msg("delete recipe: erreur repository")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
			return
		}

		log.Info().
			Int64("user_id", userID).
			Int64("recipe_id", recipeID).
			Msg("recipe supprimée")

		c.Status(http.StatusNoContent)
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────

// parseID extrait et valide un paramètre de route entier positif.
func parseID(c *gin.Context, param string) (int64, error) {
	raw := c.Param(param)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("id doit être un entier positif")
	}
	return id, nil
}
