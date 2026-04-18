package httperror_test

import (
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"

	"github.com/Kitslap/HomeCooking/internal/httperror"
)

// Structs miroir des payloads réels pour tester les messages produits
// sans avoir à dépendre des packages métier.
type itemInput struct {
	Name     string   `binding:"required,min=1,max=120" validate:"required,min=1,max=120"`
	Quantity float64  `binding:"required,min=0"         validate:"required,min=0"`
	Unit     string   `binding:"required,min=1,max=20"  validate:"required,min=1,max=20"`
	Category string   `binding:"omitempty,max=50"       validate:"omitempty,max=50"`
}

type stepInput struct {
	StepOrder int    `validate:"required,min=1"`
	Content   string `validate:"required,min=1,max=2000"`
}

type recipeInput struct {
	Name       string     `validate:"required,min=2,max=120"`
	Servings   int        `validate:"required,min=1,max=50"`
	Difficulty string     `validate:"omitempty,oneof=facile moyen difficile"`
	Steps      []stepInput `validate:"omitempty,max=50,dive"`
}

func TestFormatBindingError_RequiredQuantity(t *testing.T) {
	v := validator.New()
	// Quantity=0 n'échoue PAS "required" car 0 != zero-value pour float64 avec required ?
	// En fait required détecte la "zero value" : pour un float64, c'est 0.
	// On laisse donc Quantity=0 pour provoquer la violation "required".
	err := v.Struct(itemInput{Name: "f", Unit: "pcs"})
	if err == nil {
		t.Fatal("attendu: erreur de validation")
	}
	msg := httperror.FormatBindingError(err)
	if !strings.Contains(msg, "La quantité") {
		t.Errorf("message doit parler de 'La quantité', got %q", msg)
	}
	if !strings.Contains(msg, "obligatoire") {
		t.Errorf("message doit dire 'obligatoire', got %q", msg)
	}
	// Ne doit plus exposer la trace brute
	if strings.Contains(msg, "CreateItemInput") || strings.Contains(msg, "failed on the") {
		t.Errorf("message expose la trace brute: %q", msg)
	}
}

func TestFormatBindingError_NameMin(t *testing.T) {
	v := validator.New()
	err := v.Struct(recipeInput{Name: "a", Servings: 4})
	if err == nil {
		t.Fatal("attendu: erreur 'min'")
	}
	msg := httperror.FormatBindingError(err)
	if !strings.Contains(msg, "Le nom") {
		t.Errorf("message doit référencer 'Le nom', got %q", msg)
	}
	if !strings.Contains(msg, "2") {
		t.Errorf("message doit mentionner la borne min (2), got %q", msg)
	}
}

func TestFormatBindingError_NestedSlice(t *testing.T) {
	v := validator.New()
	err := v.Struct(recipeInput{
		Name:     "Ma recette",
		Servings: 4,
		Steps:    []stepInput{{StepOrder: 0, Content: ""}},
	})
	if err == nil {
		t.Fatal("attendu: erreur dans Steps[0]")
	}
	msg := httperror.FormatBindingError(err)
	if !strings.Contains(msg, "étape #1") {
		t.Errorf("message doit mentionner 'étape #1', got %q", msg)
	}
}

func TestFormatBindingError_Oneof(t *testing.T) {
	v := validator.New()
	err := v.Struct(recipeInput{Name: "Ma recette", Servings: 4, Difficulty: "trop_chaud"})
	if err == nil {
		t.Fatal("attendu: erreur 'oneof'")
	}
	msg := httperror.FormatBindingError(err)
	if !strings.Contains(msg, "facile") || !strings.Contains(msg, "difficile") {
		t.Errorf("message doit lister les choix, got %q", msg)
	}
}

func TestFormatBindingError_Nil(t *testing.T) {
	if got := httperror.FormatBindingError(nil); got != "" {
		t.Errorf("nil doit retourner chaîne vide, got %q", got)
	}
}
