package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/shared"
)

type CountryRepository interface {
	ListCountries() ([]models.Country, error)
}

type CountryHandler struct {
	repo CountryRepository
}

func NewCountryHandler(repo CountryRepository) *CountryHandler {
	return &CountryHandler{repo: repo}
}

func (h *CountryHandler) List(c *gin.Context) {
	countries, err := h.repo.ListCountries()
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"countries": countries})
}
