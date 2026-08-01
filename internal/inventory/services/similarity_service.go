package services

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
)

// Poids par défaut du score de similarité — ajustables ici si le métier veut privilégier un
// critère (ex: remonter le poids du prix pour un client sensible au budget).
const (
	weightGenre = 0.3
	weightForme = 0.3
	weightPrix  = 0.4
)

// SimilarityService classe les montures disponibles par ressemblance avec une monture de
// référence, selon 3 critères : genre, forme et prix.
type SimilarityService struct {
	glassRepo *repositories.GlassRepository
}

// NewSimilarityService crée une nouvelle instance
func NewSimilarityService(glassRepo *repositories.GlassRepository) *SimilarityService {
	return &SimilarityService{glassRepo: glassRepo}
}

// FindSimilar classe les montures disponibles (hors réservées/vendues/etc.) par similarité avec
// la monture de référence, du score le plus élevé au plus faible, tronqué à limit.
//
// Le score est une moyenne pondérée (genre/forme/prix) renormalisée sur les seuls critères
// renseignés à la fois côté référence et côté candidat, pour ne pas pénaliser une fiche
// incomplète (ex: prix manquant) plutôt que de lui attribuer un score de 0 par défaut.
func (s *SimilarityService) FindSimilar(reference *models.GlassListItem, limit int) ([]models.SimilarGlass, error) {
	if reference.Gender == nil && reference.Shape == nil && reference.Price == nil {
		return nil, fmt.Errorf("monture de référence sans genre, forme ni prix renseigné : similarité impossible")
	}

	candidates, err := s.glassRepo.FindAvailableExcluding(reference.ID)
	if err != nil {
		return nil, err
	}

	results := make([]models.SimilarGlass, 0, len(candidates))
	for _, candidate := range candidates {
		var weightSum, total float64
		var scoreGenre, scoreForme, scorePrix float64

		if reference.Gender != nil && candidate.Gender != nil {
			scoreGenre = matchGenre(*reference.Gender, *candidate.Gender)
			weightSum += weightGenre
			total += weightGenre * scoreGenre
		}
		if reference.Shape != nil && candidate.Shape != nil {
			scoreForme = matchForme(*reference.Shape, *candidate.Shape)
			weightSum += weightForme
			total += weightForme * scoreForme
		}
		if reference.Price != nil && candidate.Price != nil {
			scorePrix = simPrix(*reference.Price, *candidate.Price)
			weightSum += weightPrix
			total += weightPrix * scorePrix
		}

		if weightSum == 0 {
			// Rien de comparable entre la référence et ce candidat (aucun champ commun renseigné).
			continue
		}

		results = append(results, models.SimilarGlass{
			GlassListItem: candidate,
			Score:         total / weightSum,
			ScoreGenre:    scoreGenre,
			ScoreForme:    scoreForme,
			ScorePrix:     scorePrix,
		})
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// matchGenre compare deux genres : identiques -> 1, l'un des deux UNISEXE -> 0.5 (compatible),
// sinon 0.
func matchGenre(a, b string) float64 {
	a, b = strings.ToUpper(strings.TrimSpace(a)), strings.ToUpper(strings.TrimSpace(b))
	if a == b {
		return 1
	}
	if a == "UNISEXE" || b == "UNISEXE" {
		return 0.5
	}
	return 0
}

// matchForme compare deux formes : identiques -> 1, sinon 0 (pas de table de proximité en v1).
func matchForme(a, b string) float64 {
	return boolToScore(strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b)))
}

func boolToScore(match bool) float64 {
	if match {
		return 1
	}
	return 0
}

// simPrix mesure la proximité relative de deux prix par rapport au prix de référence :
// identique -> 1, dégrade linéairement jusqu'à 0 à ±100% d'écart (ou plus).
func simPrix(reference, candidate float64) float64 {
	if reference <= 0 {
		return 0
	}
	ratio := math.Abs(reference-candidate) / reference
	if ratio > 1 {
		return 0
	}
	return 1 - ratio
}
