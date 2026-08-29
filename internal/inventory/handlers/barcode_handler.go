package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/services"
	"github.com/lunetterie/backend/internal/shared"
)

type BarcodeHandler struct {
	barcodeService *services.BarcodeService
}

func NewBarcodeHandler(barcodeService *services.BarcodeService) *BarcodeHandler {
	return &BarcodeHandler{barcodeService: barcodeService}
}

// Next réserve le prochain code-barres et le rend au poste de scan, qui l'affiche sur
// l'aperçu de l'étiquette puis le renvoie à l'enregistrement.
//
// Sans cette route, l'aperçu montrait un numéro fabriqué au hasard dans le navigateur : il
// ressemblait à un identifiant, n'en était pas un, et ne correspondait jamais au code que
// la monture allait recevoir.
//
// ⚠️ Le numéro est consommé à l'appel — `nextval` ne se rembobine pas. Un enregistrement
// abandonné après l'aperçu laisse donc un trou dans la numérotation. C'est sans danger : la
// séquence garantit l'unicité, pas la continuité, et un trou vaut mieux qu'un doublon.
// GET /api/v1/inventory/barcodes/next
func (h *BarcodeHandler) Next(c *gin.Context) {
	barcode, err := h.barcodeService.GenerateBarcode()
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	shared.Success(c, http.StatusOK, gin.H{"barcode": barcode})
}

// Label retourne le PNG Code 128 d'un code de valise ou de carton.
func (h *BarcodeHandler) Label(c *gin.Context) {
	code := strings.TrimSpace(c.Param("code"))
	code = strings.TrimPrefix(code, "/")
	code = strings.TrimSuffix(code, ".png")
	if code == "" {
		shared.BadRequest(c, "code-barres requis")
		return
	}
	image, err := h.barcodeService.GenerateBarcodeImage(code)
	if err != nil {
		shared.BadRequest(c, err.Error())
		return
	}
	c.Data(http.StatusOK, "image/png", image)
}
