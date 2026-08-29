package mobilecapture

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/shared"
	"golang.org/x/net/websocket"
)

const sessionLifetime = 15 * time.Minute
const maxPhotoSize = 10 << 20

type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
	At   time.Time   `json:"at"`
}

type photo struct {
	ContentType string
	Data        []byte
}
type session struct {
	ID, PCToken, DeviceToken string
	ExpiresAt                time.Time
	photo                    *photo
	clients                  map[chan Event]struct{}
}

type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*session
}

func NewManager() *Manager { return &Manager{sessions: make(map[string]*session)} }
func token() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func (m *Manager) Create() (*session, error) {
	id, e := token()
	if e != nil {
		return nil, e
	}
	pc, e := token()
	if e != nil {
		return nil, e
	}
	device, e := token()
	if e != nil {
		return nil, e
	}
	s := &session{ID: id, PCToken: pc, DeviceToken: device, ExpiresAt: time.Now().Add(sessionLifetime), clients: make(map[chan Event]struct{})}
	m.mu.Lock()
	m.sessions[id] = s
	m.mu.Unlock()
	return s, nil
}
func same(a, b string) bool { return a != "" && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }
func (m *Manager) get(id, tok string, pc bool) (*session, bool) {
	m.mu.RLock()
	s := m.sessions[id]
	m.mu.RUnlock()
	if s == nil || time.Now().After(s.ExpiresAt) || !(same(tok, s.DeviceToken) || pc && same(tok, s.PCToken)) {
		return nil, false
	}
	return s, true
}
func (m *Manager) publish(s *session, e Event) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for ch := range s.clients {
		select {
		case ch <- e:
		default:
		}
	}
}
func (m *Manager) subscribe(s *session) chan Event {
	ch := make(chan Event, 16)
	m.mu.Lock()
	s.clients[ch] = struct{}{}
	m.mu.Unlock()
	return ch
}
func (m *Manager) unsubscribe(s *session, ch chan Event) {
	m.mu.Lock()
	delete(s.clients, ch)
	m.mu.Unlock()
}

type Handler struct{ manager *Manager }

func NewHandler(m *Manager) *Handler { return &Handler{manager: m} }
func (h *Handler) sessionFrom(c *gin.Context, pc bool) (*session, bool) {
	s, ok := h.manager.get(c.Param("id"), c.Query("token"), pc)
	if !ok {
		shared.Unauthorized(c, "Session mobile invalide ou expirée")
		return nil, false
	}
	return s, true
}
func publicBase(c *gin.Context) string {
	scheme := "http"
	if c.GetHeader("X-Forwarded-Proto") == "https" || c.Request.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host
}
func (h *Handler) Create(c *gin.Context) {
	s, err := h.manager.Create()
	if err != nil {
		shared.InternalError(c, "Création de session impossible")
		return
	}
	base := publicBase(c)
	deviceURL := fmt.Sprintf("%s/mobile-capture.html#session=%s&token=%s", base, s.ID, s.DeviceToken)
	shared.Created(c, gin.H{"id": s.ID, "pair_code": s.ID[:8], "expires_at": s.ExpiresAt, "device_url": deviceURL, "qr_url": fmt.Sprintf("%s/api/v1/inventory/mobile-capture/sessions/%s/qr?token=%s", base, s.ID, s.PCToken), "events_url": fmt.Sprintf("%s/api/v1/inventory/mobile-capture/sessions/%s/events?token=%s", base, s.ID, s.PCToken)})
}
func (h *Handler) QR(c *gin.Context) {
	s, ok := h.sessionFrom(c, true)
	if !ok {
		return
	}
	c.Header("Cross-Origin-Resource-Policy", "cross-origin")
	content := fmt.Sprintf("%s/mobile-capture.html#session=%s&token=%s", publicBase(c), s.ID, s.DeviceToken)
	code, err := qr.Encode(content, qr.M, qr.Auto)
	if err != nil {
		shared.InternalError(c, "QR impossible à générer")
		return
	}
	scaled, err := barcode.Scale(code, 360, 360)
	if err != nil {
		shared.InternalError(c, "QR impossible à dimensionner")
		return
	}
	var out bytes.Buffer
	if err := png.Encode(&out, scaled); err != nil {
		shared.InternalError(c, "QR impossible à encoder")
		return
	}
	c.Data(http.StatusOK, "image/png", out.Bytes())
}
func (h *Handler) Events(c *gin.Context) {
	s, ok := h.sessionFrom(c, true)
	if !ok {
		return
	}
	server := websocket.Server{
		Handshake: func(*websocket.Config, *http.Request) error { return nil },
		Handler: func(ws *websocket.Conn) {
		ch := h.manager.subscribe(s)
		defer h.manager.unsubscribe(s, ch)
		for event := range ch {
			if err := websocket.JSON.Send(ws, event); err != nil {
				return
			}
		}
		},
	}
	server.ServeHTTP(c.Writer, c.Request)
}
func (h *Handler) Connect(c *gin.Context) {
	s, ok := h.sessionFrom(c, false)
	if !ok {
		return
	}
	h.manager.publish(s, Event{Type: "device.connected", Data: gin.H{"connected": true}, At: time.Now()})
	shared.Success(c, http.StatusOK, gin.H{"connected": true, "expires_at": s.ExpiresAt})
}
func (h *Handler) Scan(c *gin.Context) {
	s, ok := h.sessionFrom(c, false)
	if !ok {
		return
	}
	var req struct {
		Barcode string `json:"barcode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Barcode) == "" {
		shared.BadRequest(c, "Code-barres requis")
		return
	}
	barcode := strings.ToUpper(strings.TrimSpace(req.Barcode))
	h.manager.publish(s, Event{Type: "barcode.scanned", Data: gin.H{"barcode": barcode}, At: time.Now()})
	shared.Success(c, http.StatusOK, gin.H{"received": true, "barcode": barcode})
}
func (h *Handler) UploadPhoto(c *gin.Context) {
	s, ok := h.sessionFrom(c, false)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxPhotoSize)
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		shared.BadRequest(c, "Photo requise (10 Mo maximum)")
		return
	}
	defer file.Close()
	typ := header.Header.Get("Content-Type")
	if typ != "image/jpeg" && typ != "image/png" && typ != "image/webp" {
		shared.BadRequest(c, "JPEG, PNG ou WebP requis")
		return
	}
	data, err := io.ReadAll(file)
	if err != nil {
		shared.BadRequest(c, "Photo illisible")
		return
	}
	h.manager.mu.Lock()
	s.photo = &photo{ContentType: typ, Data: data}
	h.manager.mu.Unlock()
	url := fmt.Sprintf("/api/v1/inventory/mobile-capture/sessions/%s/photo?token=%s", s.ID, s.PCToken)
	h.manager.publish(s, Event{Type: "photo.received", Data: gin.H{"photo_url": url, "size": len(data)}, At: time.Now()})
	shared.Success(c, http.StatusCreated, gin.H{"received": true})
}
func (h *Handler) Photo(c *gin.Context) {
	s, ok := h.sessionFrom(c, true)
	if !ok {
		return
	}
	h.manager.mu.RLock()
	p := s.photo
	h.manager.mu.RUnlock()
	if p == nil {
		shared.NotFound(c, "Aucune photo reçue")
		return
	}
	c.Data(http.StatusOK, p.ContentType, p.Data)
}
func (h *Handler) Close(c *gin.Context) {
	s, ok := h.sessionFrom(c, true)
	if !ok {
		return
	}
	h.manager.mu.Lock()
	delete(h.manager.sessions, s.ID)
	h.manager.mu.Unlock()
	shared.Success(c, http.StatusOK, gin.H{"closed": true})
}

// Keep json imported when compiling with older Go tooling that removes websocket JSON references differently.
var _ = json.Valid
