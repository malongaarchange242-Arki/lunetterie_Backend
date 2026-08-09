package handlers

import (
	"testing"

	"github.com/lunetterie/backend/internal/inventory/models"
)

// La conversion des champs d'ordonnance décide de ce qui atterrit en base. Une valeur mal
// lue n'échoue pas bruyamment : elle part à NULL et personne ne s'en aperçoit avant de
// vouloir compter les progressifs. D'où ces cas, tirés de ce que le formulaire laisse
// réellement saisir.
func TestOptionalDecimalAcceptsWhatTheFormAllows(t *testing.T) {
	cases := []struct {
		name  string
		raw   string
		want  float64
		isNil bool
	}{
		{name: "sphère positive telle qu'affichée", raw: "+1.00", want: 1},
		{name: "cylindre négatif", raw: "-0.50", want: -0.5},
		{name: "virgule décimale, clavier français", raw: "-0,75", want: -0.75},
		{name: "degré collé à l'axe", raw: "60°", want: 60},
		{name: "espaces autour", raw: "  2.25 ", want: 2.25},
		{name: "vide", raw: "", isNil: true},
		{name: "saisie à moitié tapée", raw: "+", isNil: true},
		{name: "texte", raw: "n/a", isNil: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := optionalDecimal(tc.raw)
			if tc.isNil {
				if got != nil {
					t.Fatalf("optionalDecimal(%q) = %v, attendu nil", tc.raw, *got)
				}
				return
			}
			if got == nil {
				t.Fatalf("optionalDecimal(%q) = nil, attendu %v", tc.raw, tc.want)
			}
			if *got != tc.want {
				t.Fatalf("optionalDecimal(%q) = %v, attendu %v", tc.raw, *got, tc.want)
			}
		})
	}
}

// L'axe est borné par une contrainte CHECK : une valeur hors bornes ferait échouer toute
// la transaction, donc elle doit être écartée avant l'insertion.
func TestOptionalAxisRejectsOutOfRange(t *testing.T) {
	if got := optionalAxis("181"); got != nil {
		t.Fatalf("axe 181 accepté (%v) alors que la table le refuse", *got)
	}
	if got := optionalAxis("-1"); got != nil {
		t.Fatalf("axe -1 accepté (%v) alors que la table le refuse", *got)
	}
	if got := optionalAxis("0"); got == nil || *got != 0 {
		t.Fatalf("axe 0 doit être conservé : c'est une valeur valide, pas une absence")
	}
	if got := optionalAxis("180°"); got == nil || *got != 180 {
		t.Fatalf("axe 180° doit être lu comme 180")
	}
}

// La remise dépasse rarement 100 %, mais une saisie aberrante ne doit pas faire perdre la
// proforma : on la ramène dans les bornes plutôt que de laisser la contrainte rejeter.
func TestBuildPrescriptionClampsRemise(t *testing.T) {
	high := buildPrescription(&models.ProformaPrescriptionRequest{RemisePct: 150})
	if high.RemisePct != 100 {
		t.Fatalf("remise 150 %% = %v, attendu 100", high.RemisePct)
	}
	negative := buildPrescription(&models.ProformaPrescriptionRequest{RemisePct: -10})
	if negative.RemisePct != 0 {
		t.Fatalf("remise -10 %% = %v, attendu 0", negative.RemisePct)
	}
}

// Une proforma envoyée sans bloc ordonnance reste valide : les clients qui n'ont pas migré
// continuent d'appeler l'endpoint avec le seul champ note.
func TestBuildPrescriptionNilWhenAbsent(t *testing.T) {
	if got := buildPrescription(nil); got != nil {
		t.Fatalf("buildPrescription(nil) = %v, attendu nil", got)
	}
}

// « PARTICULIER » ou un champ laissé vide n'est pas une société : NULL plutôt qu'une
// chaîne vide, sans quoi une requête « clients société » les compterait.
func TestOptionalTextDropsBlank(t *testing.T) {
	if got := optionalText("   "); got != nil {
		t.Fatalf("optionalText(espaces) = %q, attendu nil", *got)
	}
	if got := optionalText(" Progressif "); got == nil || *got != "Progressif" {
		t.Fatalf("optionalText doit conserver la valeur en la débarrassant de ses espaces")
	}
}
