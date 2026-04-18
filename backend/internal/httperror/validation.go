// Package httperror centralise la conversion des erreurs techniques
// (binding Gin / go-playground/validator, JSON malformé, etc.) en messages
// francophones clairs destinés à l'utilisateur final.
//
// Objectif UX : ne jamais renvoyer dans la réponse HTTP la représentation
// brute d'une erreur go-playground/validator du type :
//
//	Key: 'CreateItemInput.Quantity' Error:Field validation for 'Quantity'
//	failed on the 'required' tag
//
// Ces messages sont incompréhensibles pour un utilisateur non-technique.
// FormatBindingError produit à la place des messages du style :
//
//	« La quantité est obligatoire »
//	« Le nom doit contenir au moins 2 caractères »
//	« Le nom de l'ingrédient #1 est obligatoire »
//
// Le mapping struct/field → libellé FR est centralisé ici pour rester
// cohérent quels que soient les endpoints, et pour faciliter l'ajout
// de nouveaux champs métier sans devoir éditer chaque handler.
package httperror

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// FormatBindingError convertit une erreur retournée par Gin's ShouldBindJSON /
// ShouldBindQuery (go-playground/validator + json.Unmarshal) en un message
// FR explicite et lisible. Si l'erreur ne rentre dans aucun cas connu, un
// message générique est retourné plutôt que la chaîne brute de la lib.
func FormatBindingError(err error) string {
	if err == nil {
		return ""
	}

	// Erreurs de validation (tags required/min/max/oneof/...)
	var ves validator.ValidationErrors
	if errors.As(err, &ves) {
		msgs := make([]string, 0, len(ves))
		seen := make(map[string]struct{}, len(ves))
		for _, fe := range ves {
			m := friendlyMessage(fe)
			if _, dup := seen[m]; dup {
				continue
			}
			seen[m] = struct{}{}
			msgs = append(msgs, m)
		}
		if len(msgs) == 0 {
			return "Les données envoyées sont invalides."
		}
		if len(msgs) == 1 {
			return msgs[0] + "."
		}
		return strings.Join(msgs, " · ") + "."
	}

	// Erreurs de parsing JSON (type mismatch, JSON malformé, champ inconnu…)
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		label := fieldLabel(typeErr.Field)
		return fmt.Sprintf("%s a un format invalide (attendu : %s).", label, humanType(typeErr.Type.String()))
	}
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return "Le format JSON envoyé est invalide."
	}

	// Fallback — on masque toute trace interne potentiellement cryptique.
	return "Les données envoyées sont invalides."
}

// ── Construction du message par erreur de champ ──────────────────────────────

func friendlyMessage(fe validator.FieldError) string {
	label := fieldLabel(fe.StructNamespace())
	param := fe.Param()

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s est obligatoire", label)

	case "min":
		switch fe.Kind().String() {
		case "string":
			if param == "1" {
				return fmt.Sprintf("%s ne peut pas être vide", label)
			}
			return fmt.Sprintf("%s doit contenir au moins %s caractères", label, param)
		case "slice", "array", "map":
			return fmt.Sprintf("%s doit contenir au moins %s élément(s)", label, param)
		default:
			return fmt.Sprintf("%s doit être supérieur ou égal à %s", label, param)
		}

	case "max":
		switch fe.Kind().String() {
		case "string":
			return fmt.Sprintf("%s ne peut pas dépasser %s caractères", label, param)
		case "slice", "array", "map":
			return fmt.Sprintf("%s ne peut pas contenir plus de %s élément(s)", label, param)
		default:
			return fmt.Sprintf("%s doit être inférieur ou égal à %s", label, param)
		}

	case "oneof":
		// param = "facile moyen difficile" → "facile, moyen ou difficile"
		choices := strings.Fields(param)
		var joined string
		switch len(choices) {
		case 0:
			joined = param
		case 1:
			joined = choices[0]
		default:
			joined = strings.Join(choices[:len(choices)-1], ", ") + " ou " + choices[len(choices)-1]
		}
		return fmt.Sprintf("%s doit être : %s", label, joined)

	case "url":
		return fmt.Sprintf("%s doit être une URL valide", label)

	case "email":
		return fmt.Sprintf("%s doit être une adresse email valide", label)

	case "eq", "eqfield":
		return fmt.Sprintf("%s doit être égal à %s", label, param)

	case "ne", "nefield":
		return fmt.Sprintf("%s ne peut pas être égal à %s", label, param)

	case "gt":
		return fmt.Sprintf("%s doit être strictement supérieur à %s", label, param)
	case "gte":
		return fmt.Sprintf("%s doit être supérieur ou égal à %s", label, param)
	case "lt":
		return fmt.Sprintf("%s doit être strictement inférieur à %s", label, param)
	case "lte":
		return fmt.Sprintf("%s doit être inférieur ou égal à %s", label, param)

	case "alphanum":
		return fmt.Sprintf("%s ne doit contenir que des lettres et chiffres", label)
	case "numeric":
		return fmt.Sprintf("%s doit être un nombre", label)

	default:
		return fmt.Sprintf("%s est invalide", label)
	}
}

// ── Traduction struct.Field → libellé FR ─────────────────────────────────────

// labelMap donne le libellé FR à afficher pour chaque champ métier connu.
// Les clés utilisent le nom de struct Go (pas le tag json) afin d'être robustes
// à d'éventuels changements de sérialisation JSON.
var labelMap = map[string]string{
	// Auth / Setup
	"Username":    "Le nom d'utilisateur",
	"Password":    "Le mot de passe",

	// Storage / Inventaire
	"Name":        "Le nom",
	"Quantity":    "La quantité",
	"Unit":        "L'unité",
	"Category":    "La catégorie",
	"Expiry":      "La date d'expiration",
	"AlertAt":     "Le seuil d'alerte",
	"Notes":       "Les notes",
	"Delta":       "La variation de quantité",

	// Recipe
	"Description": "La description",
	"Servings":    "Le nombre de portions",
	"PrepTime":    "Le temps de préparation",
	"CookTime":    "Le temps de cuisson",
	"Difficulty":  "La difficulté",
	"Tags":        "Les tags",
	"ImageURL":    "L'URL de l'image",
	"Ingredients": "Les ingrédients",
	"Steps":       "Les étapes",
	"StepOrder":   "L'ordre de l'étape",
	"Content":     "Le contenu de l'étape",
}

// fieldLabel extrait de `StructNamespace()` le libellé FR à afficher.
// Le namespace peut ressembler à :
//
//	"CreateItemInput.Quantity"
//	"CreateRecipeInput.Ingredients[0].Name"
//	"UpdateRecipeInput.Name"
//
// Pour les champs imbriqués dans un slice (Ingredients/Steps), on injecte
// le numéro de l'élément (1-indexé pour l'utilisateur).
func fieldLabel(namespace string) string {
	parts := strings.Split(namespace, ".")
	last := parts[len(parts)-1]
	parent := ""
	if len(parts) >= 2 {
		parent = parts[len(parts)-2]
	}

	// Cas imbriqué : Ingredients[0].Name / Steps[2].Content / ...
	if parent != "" {
		if i := strings.Index(parent, "["); i > 0 && strings.HasSuffix(parent, "]") {
			parentName := parent[:i]
			idxStr := parent[i+1 : len(parent)-1]
			idx := displayIndex(idxStr)

			switch parentName {
			case "Ingredients":
				switch last {
				case "Name":
					return fmt.Sprintf("Le nom de l'ingrédient #%s", idx)
				case "Quantity":
					return fmt.Sprintf("La quantité de l'ingrédient #%s", idx)
				case "Unit":
					return fmt.Sprintf("L'unité de l'ingrédient #%s", idx)
				case "SortOrder":
					return fmt.Sprintf("L'ordre de l'ingrédient #%s", idx)
				}
			case "Steps":
				switch last {
				case "Content":
					return fmt.Sprintf("Le contenu de l'étape #%s", idx)
				case "StepOrder":
					return fmt.Sprintf("L'ordre de l'étape #%s", idx)
				}
			case "Tags":
				return fmt.Sprintf("Le tag #%s", idx)
			}
		}
	}

	if lbl, ok := labelMap[last]; ok {
		return lbl
	}

	// Si on ne connaît pas le champ, on renvoie un libellé générique discret
	// (évite d'exposer le nom de struct Go).
	return "Un champ"
}

// displayIndex convertit l'index 0-based de validator en index 1-based utilisateur.
func displayIndex(raw string) string {
	var n int
	if _, err := fmt.Sscanf(raw, "%d", &n); err != nil {
		return raw
	}
	return fmt.Sprintf("%d", n+1)
}

// humanType traduit le nom Go d'un type (string, int, bool, float64…)
// en libellé FR simple pour les erreurs json.UnmarshalTypeError.
func humanType(goType string) string {
	switch goType {
	case "string":
		return "texte"
	case "bool":
		return "vrai/faux"
	case "int", "int32", "int64":
		return "nombre entier"
	case "float32", "float64":
		return "nombre"
	}
	return goType
}
