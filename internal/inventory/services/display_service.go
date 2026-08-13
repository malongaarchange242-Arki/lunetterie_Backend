package services

import (
	"fmt"
	"log"
	"strings"

	authRepositories "github.com/lunetterie/backend/internal/auth/repositories"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
)

// DisplayService gère la mise en présentoir des montures (le code-barres ne change jamais)
type DisplayService struct {
	glassRepo    *repositories.GlassRepository
	movementRepo *repositories.MovementRepository
	allocation   *AllocationService
	stationRepo  *authRepositories.StationRepository
	transferRepo *repositories.TransferRepository
	userRepo     *authRepositories.UserRepository
	locationRepo *repositories.LocationRepository
}

// NewDisplayService crée une nouvelle instance
func NewDisplayService(glassRepo *repositories.GlassRepository, movementRepo *repositories.MovementRepository, allocation *AllocationService, stationRepo *authRepositories.StationRepository, transferRepo *repositories.TransferRepository, userRepo *authRepositories.UserRepository, locationRepo *repositories.LocationRepository) *DisplayService {
	return &DisplayService{glassRepo: glassRepo, movementRepo: movementRepo, allocation: allocation, stationRepo: stationRepo, transferRepo: transferRepo, userRepo: userRepo, locationRepo: locationRepo}
}

// AssignPresentoirSlot pose une monture dans un casier désigné, au lieu du premier casier
// libre que retient l'attribution automatique (allocation_service.go findOrCreateLocation).
//
// La vendeuse range la monture là où il y a physiquement de la place, pas là où le système
// l'a décidé : sans ce geste, le casier enregistré et le casier réel divergent dès la
// première journée, et l'emplacement affiché n'aide plus personne à retrouver une monture.
//
// Même ordre que RelocateGlass : on occupe le nouveau casier AVANT de libérer l'ancien. Un
// casier resté marqué occupé à tort se corrige d'un scan ; une monture sans emplacement se
// cherche en rayon.
func (s *DisplayService) AssignPresentoirSlot(barcode string, stationID int64, locationCode string, userID int64) (*models.StorageLocation, error) {
	code := strings.ToUpper(strings.TrimSpace(locationCode))
	if code == "" {
		return nil, fmt.Errorf("casier requis")
	}

	glass, err := s.glassRepo.GetByBarcode(barcode)
	if err != nil {
		return nil, err
	}
	if glass.StationID != stationID {
		return nil, fmt.Errorf("la monture %s n'est pas à cette station", barcode)
	}
	// Un casier de présentoir ne se donne qu'à une monture exposée : en accepter une qui est
	// en caisse ou au laboratoire marquerait une place occupée par une monture absente.
	if glass.Status != models.StatusEnPresentoir {
		return nil, fmt.Errorf("statut actuel « %s » : seule une monture en présentoir occupe un casier", glass.Status)
	}

	location, err := s.locationRepo.FindOrCreatePresentoirByCode(stationID, code)
	if err != nil {
		return nil, err
	}

	// Déjà dans ce casier : ne rien faire. Poursuivre libérerait l'ancien emplacement, qui
	// est le même que le nouveau — la monture se retrouverait dans un casier marqué libre.
	if glass.LocationID != nil && *glass.LocationID == location.ID {
		return location, nil
	}

	occupant, err := s.locationRepo.FindGlassBarcodeAtLocation(location.ID)
	if err != nil {
		return nil, err
	}
	if occupant != "" {
		return nil, fmt.Errorf("le casier %s est déjà occupé par %s", code, occupant)
	}

	oldLocationID := glass.LocationID
	if err := s.locationRepo.UpdateStatus(location.ID, "OCCUPE"); err != nil {
		return nil, err
	}
	if err := s.glassRepo.UpdateLocation(glass.ID, location.ID); err != nil {
		if freeErr := s.allocation.FreeLocation(location.ID); freeErr != nil {
			log.Printf("⚠️  Casier #%d occupé mais non attribué et non libéré: %v", location.ID, freeErr)
		}
		return nil, err
	}
	if oldLocationID != nil {
		if err := s.allocation.FreeLocation(*oldLocationID); err != nil {
			log.Printf("⚠️  Erreur libération ancien casier (glass #%d): %v", glass.ID, err)
		}
	}

	movement := &models.Movement{
		GlassID:        glass.ID,
		FromStationID:  &glass.StationID,
		ToStationID:    &glass.StationID,
		FromLocationID: oldLocationID,
		ToLocationID:   &location.ID,
		Action:         models.ActionRangement,
		UserID:         userID,
	}
	if err := s.movementRepo.Create(movement); err != nil {
		log.Printf("⚠️  Erreur création mouvement rangement (glass #%d): %v", glass.ID, err)
	}

	location.Status = "OCCUPE"
	return location, nil
}

// RelocateGlass attribue un nouvel emplacement libre à une monture, dans la même zone de la
// même station — utilisé quand on réimprime son étiquette et qu'on la repose ailleurs.
//
// L'ordre compte : on réserve le nouvel emplacement AVANT de libérer l'ancien. L'inverse
// laisserait, en cas d'échec de la recherche, une monture sans emplacement. Ici le pire cas
// est un emplacement resté marqué occupé, qui se corrige, alors qu'une monture égarée non.
//
// FindFreeLocation ne rend que des emplacements LIBRE : celui qu'occupe actuellement la
// monture est donc exclu d'office, le nouvel emplacement est toujours différent.
func (s *DisplayService) RelocateGlass(barcode string, userID int64) (*models.StorageLocation, error) {
	glass, err := s.glassRepo.GetByBarcode(barcode)
	if err != nil {
		return nil, err
	}

	zone := models.ZoneStock
	switch glass.Status {
	case models.StatusEnPresentoir:
		zone = models.ZonePresentoir
	case models.StatusEnLaboratoire:
		zone = models.ZoneLaboratoire
	}

	location, err := s.allocation.FindFreeLocation(glass.StationID, zone)
	if err != nil {
		return nil, err
	}

	oldLocationID := glass.LocationID
	if err := s.glassRepo.UpdateLocation(glass.ID, location.ID); err != nil {
		// La monture n'a pas bougé : on rend l'emplacement qu'on venait de réserver.
		if freeErr := s.allocation.FreeLocation(location.ID); freeErr != nil {
			log.Printf("⚠️  Emplacement #%d réservé mais non attribué et non libéré: %v", location.ID, freeErr)
		}
		return nil, err
	}

	if oldLocationID != nil {
		if err := s.allocation.FreeLocation(*oldLocationID); err != nil {
			log.Printf("⚠️  Erreur libération ancien emplacement (glass #%d): %v", glass.ID, err)
		}
	}

	movement := &models.Movement{
		GlassID:        glass.ID,
		FromStationID:  &glass.StationID,
		ToStationID:    &glass.StationID,
		FromLocationID: oldLocationID,
		ToLocationID:   &location.ID,
		Action:         models.ActionRangement,
		UserID:         userID,
	}
	if err := s.movementRepo.Create(movement); err != nil {
		log.Printf("⚠️  Erreur création mouvement rangement (glass #%d): %v", glass.ID, err)
	}

	return location, nil
}

// placeableStatuses liste les statuts à partir desquels une monture peut être placée sur le présentoir ou en laboratoire
var placeableStatuses = map[models.GlassStatus]bool{
	models.StatusEnStockSousStation: true,
	models.StatusEnStockGeneral:     true,
	// Une monture déjà exposée (présentoir ou labo) ailleurs reste déplaçable : la rescanner à un
	// nouveau poste vaut confirmation qu'elle a physiquement changé d'endroit.
	models.StatusEnPresentoir:  true,
	models.StatusEnLaboratoire: true,
	// Une monture posée en caisse mais non encaissée (le client renonce) doit pouvoir
	// retourner au présentoir par un simple scan là-bas.
	models.StatusEnCaisse: true,
}

// PlaceOnDisplay est déclenché par la recherche d'un code-barres sur presentoir.html, quel que
// soit le poste (Présentoir, Laboratoire, ou un magasin "station" comme Station Pointe-Noire).
//
// Cas EN_TRANSIT (monture envoyée par transfert, ex: depuis scan.html) : le scan vaut
// confirmation d'arrivée dans la grande majorité des cas — SAUF si un transfert actif existe
// mais vise explicitement un AUTRE poste, auquel cas le scan est refusé (avec une note), pour ne
// pas laisser un poste récupérer une monture destinée ailleurs. Une monture EN_TRANSIT sans
// aucun transfert actif retrouvé (jamais créé via le vrai circuit, données de test...) n'est
// PAS bloquée : mieux vaut recevoir la monture que la laisser bloquée sans recours dans l'app.
//   - Vers le Présentoir ou le Laboratoire : le scan vaut à la fois réception et mise en
//     présentoir/labo, en une seule action (pas de notion de "stock" intermédiaire pour ces
//     postes dédiés — tout ce qui y arrive est destiné à l'exposition/au traitement direct).
//   - Vers un magasin "station" (ex: Station Pointe-Noire) : le scan clôture le transfert (et le
//     transfert entier si c'était la dernière monture) et fait atterrir la monture en stock
//     local (EN_STOCK_SOUS_STATION) avec un emplacement de zone STOCK — PAS en présentoir.
//
// Pour tout autre statut (déjà en stock, déjà exposée ailleurs...) : la mise en présentoir/labo
// automatique ne se déclenche QUE si le poste scanné est lui-même le poste dédié Présentoir ou
// Laboratoire. À un poste "station" (magasin), rechercher une monture reste une simple
// consultation — aucun changement de statut. Faire passer une monture de "en stock local" à "sur
// le présentoir" depuis un magasin se fait explicitement via le bouton "Envoyer" (transfert réel
// vers le poste Présentoir), pas par une simple recherche.
//
// Le retour (string, error) renvoie en 2e position une erreur réelle (ex: monture introuvable),
// et en 1re position une note explicative facultative quand l'appel n'a rien changé (ex: "en
// transit vers un autre poste") — utile pour comprendre depuis l'UI pourquoi un scan n'a aucun
// effet, plutôt que d'échouer en silence.
func (s *DisplayService) PlaceOnDisplay(barcode string, stationID, userID int64) (string, error) {
	glass, err := s.glassRepo.GetByBarcode(barcode)
	if err != nil {
		return "", err
	}
	caisse := s.isCaissePoste(stationID, userID)

	// Rien à faire quand la monture est déjà posée là où elle doit être. Pour un caissier, le
	// bon état c'est EN_CAISSE seul : une monture encore EN_PRESENTOIR qu'il scanne, c'est
	// justement le client qui l'apporte au comptoir — elle doit basculer, pas être ignorée.
	settled := glass.Status == models.StatusEnPresentoir || glass.Status == models.StatusEnLaboratoire || glass.Status == models.StatusEnCaisse
	if caisse {
		settled = glass.Status == models.StatusEnCaisse
	}
	if glass.StationID == stationID && settled {
		return "", nil
	}

	if glass.Status == models.StatusEnTransit {
		note, blocking := s.completeTransferReception(glass.ID, stationID, userID)
		if blocking {
			return note, nil
		}
		// Le Présentoir, le Laboratoire et la Caisse n'ont pas de "stock local" distinct de
		// l'exposition : l'arrivée y vaut placement direct, en une seule étape. Un magasin
		// "station" (ex: Station Pointe-Noire) atterrit d'abord en stock local (deux étapes).
		if s.isDirectPlacementStation(stationID) || caisse {
			return "", s.placeGlass(glass, stationID, userID, caisse)
		}
		return "", s.receiveIntoLocalStock(glass, stationID, userID)
	}

	// Le placement automatique au scan ne s'applique qu'aux postes dédiés (Présentoir,
	// Laboratoire, Caisse). À un poste "station" (magasin, ex: Station Pointe-Noire),
	// rechercher une monture déjà en stock local reste une simple consultation — le passage au
	// présentoir se fait explicitement via le bouton "Envoyer" (transfert réel vers Présentoir).
	if !s.isDirectPlacementStation(stationID) && !caisse {
		return "", nil
	}

	if !placeableStatuses[glass.Status] {
		return fmt.Sprintf("Statut actuel « %s » : cette monture ne peut pas être placée sur le présentoir depuis ce statut.", glass.Status), nil
	}
	return "", s.placeGlass(glass, stationID, userID, caisse)
}

// SkippedBarcode explique pourquoi une monture sélectionnée n'a pas suivi le mouvement.
type SkippedBarcode struct {
	Barcode string `json:"barcode"`
	Reason  string `json:"reason"`
}

// SendToCaisse fait passer les montures sélectionnées du présentoir au comptoir
// d'encaissement (EN_PRESENTOIR -> EN_CAISSE). Déclenché par le bouton « Envoyer » du poste
// Présentoir, là où le vendeur pose physiquement la monture sur le comptoir.
//
// La monture ne change PAS de station : le caissier tient le comptoir du magasin et sa liste
// est filtrée sur son propre station_id (stockListStatus() dans presentoir.js). Passer par un
// transfert inter-stations la ferait transiter par EN_TRANSIT et exigerait un second scan au
// comptoir — deux gestes pour un seul mouvement réel. Seul l'emplacement change : le casier de
// présentoir est libéré (il redevient à regarnir) au profit d'un emplacement de stock, comme
// quand le caissier scanne lui-même la monture.
//
// Le lot n'est pas atomique : chaque monture est traitée pour elle-même et les refus sont
// renvoyés avec leur raison, plutôt que de faire échouer l'envoi entier pour une monture déjà
// partie. L'erreur n'est renvoyée que si RIEN n'a pu être envoyé.
func (s *DisplayService) SendToCaisse(stationID int64, barcodes []string, userID int64) ([]string, []SkippedBarcode, error) {
	if len(barcodes) == 0 {
		return nil, nil, fmt.Errorf("aucune monture sélectionnée")
	}

	sent := make([]string, 0, len(barcodes))
	skipped := make([]SkippedBarcode, 0)

	for _, barcode := range barcodes {
		glass, err := s.glassRepo.GetByBarcode(barcode)
		if err != nil {
			skipped = append(skipped, SkippedBarcode{Barcode: barcode, Reason: "monture introuvable"})
			continue
		}
		// Déjà au comptoir : le vendeur a cliqué deux fois, ou le caissier l'a scannée entre
		// temps. Rien à faire, et surtout pas une erreur.
		if glass.Status == models.StatusEnCaisse {
			skipped = append(skipped, SkippedBarcode{Barcode: barcode, Reason: "déjà en caisse"})
			continue
		}
		if glass.Status != models.StatusEnPresentoir {
			skipped = append(skipped, SkippedBarcode{Barcode: barcode, Reason: fmt.Sprintf("statut « %s » : seule une monture en présentoir part en caisse", glass.Status)})
			continue
		}
		if glass.StationID != stationID {
			skipped = append(skipped, SkippedBarcode{Barcode: barcode, Reason: fmt.Sprintf("exposée à %s, pas à ce poste", s.stationDisplayName(glass.StationID))})
			continue
		}

		if err := s.placeGlass(glass, stationID, userID, true); err != nil {
			skipped = append(skipped, SkippedBarcode{Barcode: barcode, Reason: err.Error()})
			continue
		}
		sent = append(sent, barcode)
	}

	if len(sent) == 0 {
		return sent, skipped, fmt.Errorf("aucune monture n'a pu être envoyée en caisse")
	}
	return sent, skipped, nil
}

// receiveIntoLocalStock fait atterrir une monture reçue par transfert dans le stock local de la
// station (zone STOCK), sans la mettre sur le présentoir. Reflète TransferService.ReceiveItem,
// mais déclenché ici automatiquement par la simple recherche du code-barres.
func (s *DisplayService) receiveIntoLocalStock(glass *models.Glass, stationID, userID int64) error {
	location, err := s.allocation.FindOrCreateStockLocation(stationID)
	if err != nil {
		log.Printf("⚠️  Impossible d'assigner un emplacement de stock pour la réception de la monture #%d: %v", glass.ID, err)
		return nil
	}

	oldStationID := glass.StationID
	oldLocationID := glass.LocationID
	if oldLocationID != nil {
		if err := s.allocation.FreeLocation(*oldLocationID); err != nil {
			log.Printf("⚠️  Erreur libération ancien emplacement (glass #%d): %v", glass.ID, err)
		}
	}
	if err := s.glassRepo.UpdateStationAndLocation(glass.ID, stationID, location.ID); err != nil {
		return err
	}
	if err := s.glassRepo.UpdateStatus(glass.ID, models.StatusEnStockSousStation); err != nil {
		return err
	}

	movement := &models.Movement{
		GlassID:        glass.ID,
		FromStationID:  &oldStationID,
		ToStationID:    &stationID,
		FromLocationID: oldLocationID,
		ToLocationID:   &location.ID,
		Action:         models.ActionReceptionStation,
		UserID:         userID,
	}
	if err := s.movementRepo.Create(movement); err != nil {
		log.Printf("⚠️  Erreur création mouvement réception station (glass #%d): %v", glass.ID, err)
	}
	return nil
}

// placeGlass assigne un emplacement présentoir (ou laboratoire) et bascule le statut en
// conséquence (EN_PRESENTOIR, ou EN_LABORATOIRE si la station scannée est le Laboratoire).
func (s *DisplayService) placeGlass(glass *models.Glass, stationID, userID int64, isCaisse bool) error {
	// La caisse n'expose rien : une monture au comptoir attend d'être payée. Elle occupe un
	// emplacement de stock, pas un casier de présentoir qu'elle bloquerait pour rien.
	var location *models.StorageLocation
	var err error
	if isCaisse {
		location, err = s.allocation.FindOrCreateStockLocation(stationID)
	} else {
		location, err = s.allocation.FindOrCreatePresentoirLocation(stationID, glass.Barcode)
	}
	if err != nil {
		// Remonté en erreur, pas avalé : SendToCaisse répond à un clic sur « Envoyer » et
		// annoncerait la monture partie en caisse alors que rien n'a bougé.
		log.Printf("⚠️  Impossible d'assigner un emplacement pour la monture #%d: %v", glass.ID, err)
		return fmt.Errorf("aucun emplacement disponible pour la monture %s", glass.Barcode)
	}

	oldStationID := glass.StationID
	oldLocationID := glass.LocationID
	if oldLocationID != nil {
		if err := s.allocation.FreeLocation(*oldLocationID); err != nil {
			log.Printf("⚠️  Erreur libération ancien emplacement (glass #%d): %v", glass.ID, err)
		}
	}
	if err := s.glassRepo.UpdateStationAndLocation(glass.ID, stationID, location.ID); err != nil {
		return err
	}

	status := models.StatusEnPresentoir
	action := models.ActionMiseEnPresentoir
	switch {
	// Le rôle prime sur la station : un caissier qui scanne met en caisse, où qu'il soit.
	case isCaisse:
		status = models.StatusEnCaisse
		action = models.ActionMiseEnCaisse
	case s.isLaboratoireStation(stationID):
		status = models.StatusEnLaboratoire
		action = models.ActionEnvoiLaboratoire
	}

	if err := s.glassRepo.UpdateStatus(glass.ID, status); err != nil {
		return err
	}

	movement := &models.Movement{
		GlassID:        glass.ID,
		FromStationID:  &oldStationID,
		ToStationID:    &stationID,
		FromLocationID: oldLocationID,
		ToLocationID:   &location.ID,
		Action:         action,
		UserID:         userID,
	}
	if err := s.movementRepo.Create(movement); err != nil {
		log.Printf("⚠️  Erreur création mouvement mise en présentoir (glass #%d): %v", glass.ID, err)
	}
	return nil
}

// completeTransferReception clôture, en best-effort, la réception du transfert actif d'une
// monture. Renvoie (note, blocking) : blocking=true UNIQUEMENT quand un transfert actif existe
// mais vise un AUTRE poste que celui scanné — dans ce cas, et uniquement celui-ci, la réception
// est refusée (pour ne pas laisser un poste récupérer une monture destinée ailleurs). Dans tous
// les autres cas (aucun transfert actif retrouvé, service indisponible, erreur de clôture),
// blocking est false : le scan vaut quand même confirmation d'arrivée, plutôt que de laisser la
// monture bloquée EN_TRANSIT indéfiniment sans recours possible depuis l'app.
func (s *DisplayService) completeTransferReception(glassID, stationID, userID int64) (string, bool) {
	if s.transferRepo == nil {
		return "", false
	}

	item, err := s.transferRepo.GetActiveItemByGlassID(glassID)
	if err != nil {
		// Aucun transfert actif retrouvé : rien à clôturer, mais on ne bloque pas la réception.
		return "", false
	}
	transfer, err := s.transferRepo.GetByID(item.TransferID)
	if err != nil {
		return "", false
	}
	if transfer.ToStationID != stationID {
		return fmt.Sprintf("Cette monture est en transit vers %s (station #%d), pas vers ce poste-ci : impossible de la réceptionner ici.", s.stationDisplayName(transfer.ToStationID), transfer.ToStationID), true
	}

	if err := s.transferRepo.MarkItemReceived(item.ID); err != nil {
		log.Printf("⚠️  Erreur clôture ligne de transfert (item #%d): %v", item.ID, err)
		return "", false
	}

	remaining, err := s.transferRepo.CountItemsNotReceived(transfer.ID)
	if err != nil {
		log.Printf("⚠️  Erreur comptage montures restantes (transfert #%d): %v", transfer.ID, err)
		return "", false
	}
	if remaining == 0 {
		if err := s.transferRepo.MarkReceived(transfer.ID, userID); err != nil {
			log.Printf("⚠️  Erreur clôture transfert #%d: %v", transfer.ID, err)
		}
	}
	return "", false
}

func (s *DisplayService) stationDisplayName(stationID int64) string {
	if s.stationRepo != nil {
		if station, err := s.stationRepo.GetByID(stationID); err == nil {
			return station.Name
		}
	}
	return fmt.Sprintf("la station #%d", stationID)
}

func (s *DisplayService) isLaboratoireStation(stationID int64) bool {
	return s.stationNameEquals(stationID, "Laboratoire")
}

func (s *DisplayService) isPresentoirStation(stationID int64) bool {
	return s.stationNameEquals(stationID, "Présentoir")
}

func (s *DisplayService) isCaisseStation(stationID int64) bool {
	return s.stationNameEquals(stationID, "Caisse")
}

// isCaissePoste : le poste Caisse se reconnaît d'abord au RÔLE de celui qui scanne, pas à la
// station. Le caissier tient le comptoir d'un magasin existant (station Présentoir) : côté
// station rien ne le distingue d'un vendeur, seul son rôle dit que son scan vaut mise en
// caisse. La station « Caisse » reste reconnue pour une installation qui lui dédierait un poste.
func (s *DisplayService) isCaissePoste(stationID, userID int64) bool {
	return s.isCaisseStation(stationID) || s.userHasRole(userID, "CAISSIER")
}

func (s *DisplayService) userHasRole(userID int64, role string) bool {
	if s.userRepo == nil {
		return false
	}
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		// Sans rôle lisible on retombe sur le comportement historique (mise en présentoir)
		// plutôt que d'échouer : le scan doit rester utilisable.
		log.Printf("⚠️  Rôle introuvable pour l'utilisateur #%d, scan traité comme non-caisse: %v", userID, err)
		return false
	}
	return strings.EqualFold(user.RoleName, role)
}

// isDirectPlacementStation : postes où l'arrivée d'une monture vaut placement immédiat, sans
// étape de stock local. Le client apporte sa monture au comptoir comme on la pose au
// présentoir — le scan constate le geste, il ne l'annonce pas.
func (s *DisplayService) isDirectPlacementStation(stationID int64) bool {
	return s.isPresentoirStation(stationID) || s.isLaboratoireStation(stationID) || s.isCaisseStation(stationID)
}

func (s *DisplayService) stationNameEquals(stationID int64, name string) bool {
	if s.stationRepo == nil {
		return false
	}

	station, err := s.stationRepo.GetByID(stationID)
	if err != nil {
		log.Printf("⚠️  Impossible de récupérer la station #%d: %v", stationID, err)
		return false
	}
	return strings.EqualFold(strings.TrimSpace(station.Name), name)
}
