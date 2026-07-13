// Package unpoly implements the Unpoly v3.14 server-side protocol.
// Reference: https://unpoly.com/up.protocol
package unpoly

import (
	"encoding/json"
	"net/http"
	"strings"
)

// ─── Request headers (sent by Unpoly) ───

const (
	HeaderVersion  = "X-Up-Version"  // e.g. "1.0" – present when request comes from Unpoly
	HeaderTarget   = "X-Up-Target"   // CSS selector for the target fragment
	HeaderValidate = "X-Up-Validate" // field name(s) to validate, or ":unknown" for server-driven validation
	HeaderMode     = "X-Up-Mode"     // layer mode (e.g. "drawer", "modal", "popup")
)

// ─── Response headers (read by Unpoly) ───

const (
	HeaderRespTarget      = "X-Up-Target"       // override target selector
	HeaderRespLocation    = "X-Up-Location"     // redirect after fragment update
	HeaderRespMethod      = "X-Up-Method"       // HTTP method for the redirect (GET by default)
	HeaderAcceptLayer     = "X-Up-Accept-Layer" // "true"/"false" – accept or prevent overlay close; or JSON for context
	HeaderDismissLayer    = "X-Up-Dismiss-Layer" // set to any value (e.g. "true") to dismiss current layer
	HeaderOpenLayer       = "X-Up-Open-Layer"   // JSON: {"mode":"...","target":"..."} — open a new layer (experimental)
	HeaderEvents          = "X-Up-Events"       // JSON array of events to emit: [{"type":"...", ...}]
	HeaderExpireCache     = "X-Up-Expire-Cache" // CSS selector pattern to evict from cache
	HeaderEvictCache      = "X-Up-Evict-Cache"  // CSS selector pattern to evict from cache (alias)
)

// CSS classes Unpoly manages for feedback
const (
	ClassActive       = "up-active"       // added to the origin element during a request
	ClassLoading      = "up-loading"      // added to the target while loading
	ClassRevalidating = "up-revalidating" // added to the target during revalidation
)

// Event is a single Unpoly event to emit via X-Up-Events.
type Event struct {
	Type    string         `json:"type"`
	Target  string         `json:"target,omitempty"`
	Props   map[string]any `json:"props,omitempty"`
	Options map[string]any `json:"options,omitempty"`
}

// LayerDef describes a layer to open via X-Up-Open-Layer.
type LayerDef struct {
	Mode   string `json:"mode"`   // "drawer", "modal", "popup", etc.
	Target string `json:"target"` // URL to load in the layer
}

// ─── Request inspection ───

// IsUnpolyRequest reports whether the request originated from Unpoly.
func IsUnpolyRequest(r *http.Request) bool {
	return r.Header.Get(HeaderVersion) != ""
}

// IsValidation reports whether the request is a server-driven validation check.
func IsValidation(r *http.Request) bool {
	return r.Header.Get(HeaderValidate) != ""
}

// IsLayer reports whether the request was triggered from within an overlay layer.
func IsLayer(r *http.Request) bool {
	return r.Header.Get(HeaderMode) != ""
}

// ValidateField returns the field name being validated, or ":unknown".
func ValidateField(r *http.Request) string {
	return r.Header.Get(HeaderValidate)
}

// Target returns the CSS selector of the fragment being targeted.
func Target(r *http.Request) string {
	return r.Header.Get(HeaderTarget)
}

// ─── Response helpers ───

// SetTarget overrides the target selector for the response fragment.
func SetTarget(w http.ResponseWriter, selector string) {
	w.Header().Set(HeaderRespTarget, selector)
}

// SetLocation instructs Unpoly to navigate the full page after inserting the fragment.
func SetLocation(w http.ResponseWriter, url, method string) {
	w.Header().Set(HeaderRespLocation, url)
	if method != "" {
		w.Header().Set(HeaderRespMethod, method)
	}
}

// AcceptLayer controls whether an overlay should close.
// Pass "true" (string), "false" (string), or nil to signal a JSON null.
func AcceptLayer(w http.ResponseWriter, value string) {
	if value == "true" || value == "false" {
		w.Header().Set(HeaderAcceptLayer, value)
		return
	}
	w.Header().Set(HeaderAcceptLayer, "null")
}

// DismissLayer instructs Unpoly to dismiss the current overlay.
func DismissLayer(w http.ResponseWriter) {
	w.Header().Set(HeaderDismissLayer, "true")
}

// EmitEvents sends one or more events to the client.
func EmitEvents(w http.ResponseWriter, events []Event) {
	b, err := json.Marshal(events)
	if err != nil {
		return
	}
	w.Header().Set(HeaderEvents, string(b))
}

// OpenLayer instructs Unpoly to open a new overlay.
func OpenLayer(w http.ResponseWriter, mode, target string) {
	def := LayerDef{Mode: mode, Target: target}
	b, err := json.Marshal([]LayerDef{def})
	if err != nil {
		return
	}
	w.Header().Set(HeaderOpenLayer, string(b))
}

// ExpireCache evicts pages from Unpoly's client-side cache that match the selector.
func ExpireCache(w http.ResponseWriter, pattern string) {
	w.Header().Set(HeaderExpireCache, pattern)
}

// EvictCache is an alias for ExpireCache.
func EvictCache(w http.ResponseWriter, pattern string) {
	w.Header().Set(HeaderEvictCache, pattern)
}

// Vary sets the Vary header with X-Up-Version and X-Up-Target
// so HTTP caches don't serve a full-page response for an Unpoly fragment request.
func Vary(w http.ResponseWriter) {
	w.Header().Add("Vary", "X-Up-Version")
	w.Header().Add("Vary", "X-Up-Target")
}

// ─── Validation helpers ───

// ValidationErrors is a map of field name → error message, suitable for JSON serialization.
type ValidationErrors map[string]string

// WriteValidationErrors writes validation errors as JSON and sets status 422.
func WriteValidationErrors(w http.ResponseWriter, errors ValidationErrors) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	b, _ := json.Marshal(errors)
	w.Write(b)
}

// AcceptLayerJSON prevents the overlay from closing and sends JSON context.
func AcceptLayerJSON(w http.ResponseWriter, data any) {
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	w.Header().Set(HeaderAcceptLayer, string(b))
}

// IsUnpolyTarget checks if the current request targets a specific CSS selector.
func IsUnpolyTarget(r *http.Request, selector string) bool {
	return strings.TrimSpace(r.Header.Get(HeaderTarget)) == selector
}
