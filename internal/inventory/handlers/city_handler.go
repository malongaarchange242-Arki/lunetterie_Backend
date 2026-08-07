package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/shared"
)

type CityRepository interface {
	ListCitiesByCountry(countryID int64) ([]models.City, error)
}

type CityHandler struct {
	repo CityRepository
}

func NewCityHandler(repo CityRepository) *CityHandler {
	return &CityHandler{repo: repo}
}

func (h *CityHandler) ListByCountry(c *gin.Context) {
	countryIDParam := c.Query("country_id")
	if countryIDParam == "" {
		shared.BadRequest(c, "country_id est requis")
		return
	}

	countryID, err := strconv.ParseInt(countryIDParam, 10, 64)
	if err != nil {
		shared.BadRequest(c, "country_id invalide")
		return
	}

	cities, err := h.repo.ListCitiesByCountry(countryID)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	shared.Success(c, http.StatusOK, gin.H{"cities": cities})
}
