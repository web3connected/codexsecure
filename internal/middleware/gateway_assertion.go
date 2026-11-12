package middleware

// GatewayAssertion verifies the Codex Gateway's identity assertion on the channel
// routes (Phase C6) and installs the CommunicationPrincipal that ChannelAuth
// requires. It implements the existing C4 contract (codex_gateway
// src/auth/gatewayAssertion.ts, CHANNEL_ROUTES.md); it adds no field.
//
// X-CodexSecure-Gateway-Assertion is a compact JWS, EdDSA / Ed25519:
//
//	header  {"alg":"EdDSA","kid":<kid>,"typ":"codexsecure-gateway+jwt"}
//	claims  {"app","aud":"codexsecure-api","channel_sites":[<site>],"exp","iat",
//	         "iss":"codex-gateway","method","path","rid","sub":"application:<app>"}
//
// The signature covers `<header>.<claims>` as sent. `path` is the exact upstream
// request path without the query, so an assertion authorizes one method, path and
// site for its 30-second lifetime.
//
// Configuration:
//
//	CODEXSECURE_GATEWAY_ASSERTION_KEYS — comma-separated <kid>:<x>, where x is the
//	    "x" member of the gateway's public Ed25519 JWK (base64url, 32 bytes).
//	    Keep the old kid listed while the gateway rotates to a new one.
//
// The middleware never answers a request itself. On any failure it installs
// nothing, so ChannelAuth keeps its responses: 401 SERVICE_KEY_INVALID for the
// service key, which is still required, and 403 ADMIN_UNAUTHORIZED without a
// verified gateway identity. Without configured keys nothing is ever installed.

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

const (
	GatewayAssertionHeader   = "X-CodexSecure-Gateway-Assertion"
	gatewayAssertionAlg      = "EdDSA"
	gatewayAssertionType     = "codexsecure-gateway+jwt"
	gatewayAssertionIssuer   = "codex-gateway"
	gatewayAssertionAudience = "codexsecure-api"
	gatewayAssertionSkew     = 5 * time.Second
	gatewayAssertionMaxLife  = 60 * time.Second
	gatewayAssertionMaxSize  = 4 << 10
)

// The gateway's own patterns (gatewayAssertion.ts, applicationKeys.ts,
// applicationSites.ts), so Go accepts no identity the gateway could not issue.
var (
	gatewayKIDPattern  = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)
	gatewayAppPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}$`)
	gatewaySitePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	gatewayRIDPattern  = regexp.MustCompile(`^[\x21-\x7e]{1,128}$`)
	b64url             = base64.RawURLEncoding.Strict()
)

type gatewayAssertionHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	Typ string `json:"typ"`
}

type gatewayAssertionClaims struct {
	BodySHA256   string   `json:"body_sha256,omitempty"`
	App          string   `json:"app"`
	Aud          string   `json:"aud"`
	ChannelSites []string `json:"channel_sites,omitempty"`
	ReadSites    []string `json:"read_sites,omitempty"`
	AdminSites   []string `json:"admin_sites,omitempty"`
	Exp          *int64   `json:"exp"`
	Iat          *int64   `json:"iat"`
	Iss          string   `json:"iss"`
	Method       string   `json:"method"`
	Path         string   `json:"path"`
	Rid          string   `json:"rid"`
	Sub          string   `json:"sub"`
}

// GatewayVerifier holds the trusted gateway keys and the replay cache.
type GatewayVerifier struct {
	keys map[string]ed25519.PublicKey
	now  func() time.Time

	mu   sync.Mutex
	seen map[string]time.Time // rid → expiry
}

// NewGatewayVerifier copies keys; now defaults to time.Now.
func NewGatewayVerifier(keys map[string]ed25519.PublicKey, now func() time.Time) *GatewayVerifier {
	if now == nil {
		now = time.Now
	}
	copied := make(map[string]ed25519.PublicKey, len(keys))
	for kid, key := range keys {
		copied[kid] = append(ed25519.PublicKey(nil), key...)
	}
	return &GatewayVerifier{keys: copied, now: now, seen: map[string]time.Time{}}
}

// ParseGatewayAssertionKeys parses CODEXSECURE_GATEWAY_ASSERTION_KEYS.
func ParseGatewayAssertionKeys(raw string) (map[string]ed25519.PublicKey, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, errors.New("CODEXSECURE_GATEWAY_ASSERTION_KEYS is not set")
	}
	keys := map[string]ed25519.PublicKey{}
	for _, part := range strings.Split(value, ",") {
		kid, x, ok := strings.Cut(strings.TrimSpace(part), ":")
		if !ok || !gatewayKIDPattern.MatchString(kid) {
			return nil, errors.New("entries must be <kid>:<base64url Ed25519 public key>")
		}
		if _, dup := keys[kid]; dup {
			return nil, fmt.Errorf("duplicate kid %q", kid)
		}
		key, err := b64url.DecodeString(x)
		if err != nil || len(key) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("kid %q is not a base64url Ed25519 public key", kid)
		}
		keys[kid] = key
	}
	return keys, nil
}

// GatewayAssertion builds the middleware from the environment, as ChannelAuth
// reads SERVICE_API_KEY. Missing or invalid configuration fails closed.
func GatewayAssertion() gin.HandlerFunc {
	keys, err := ParseGatewayAssertionKeys(os.Getenv("CODEXSECURE_GATEWAY_ASSERTION_KEYS"))
	if err != nil {
		log.Warn().Err(err).Msg("gateway assertion keys unavailable — channel routes will refuse every request")
		return func(c *gin.Context) { c.Next() }
	}
	kids := make([]string, 0, len(keys))
	for kid := range keys {
		kids = append(kids, kid)
	}
	log.Info().Strs("kids", kids).Msg("gateway assertion keys loaded")
	return NewGatewayVerifier(keys, nil).Middleware()
}

// Middleware installs the principal only for a verified assertion.
func (v *GatewayVerifier) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		values := c.Request.Header.Values(GatewayAssertionHeader)
		if len(values) == 0 {
			c.Next()
			return
		}
		if len(values) > 1 {
			v.reject(c, errors.New("multiple assertion headers"))
			return
		}
		site := c.Param("site")
		claims, err := v.Verify(values[0], c.Request.Method, c.Request.URL.EscapedPath(), site)
		if err != nil {
			v.reject(c, err)
			return
		}
		if len(claims.AdminSites) > 0 {
			body, readErr := io.ReadAll(io.LimitReader(c.Request.Body, (64<<10)+1))
			if readErr != nil || len(body) > 64<<10 {
				v.reject(c, errors.New("invalid administrator request body"))
				return
			}
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
			digest := sha256.Sum256(body)
			if claims.BodySHA256 != hex.EncodeToString(digest[:]) {
				v.reject(c, errors.New("administrator body digest mismatch"))
				return
			}
		}
		SetCommunicationPrincipal(c, CommunicationPrincipal{
			GatewayAuthenticated: true,
			Actor:                claims.Sub,
			ChannelSites:         gatewayGrants(claims.ChannelSites),
			ReadSites:            gatewayGrants(claims.ReadSites),
			AdminSites:           gatewayGrants(claims.AdminSites),
		})
		c.Next()
	}
}

func (v *GatewayVerifier) reject(c *gin.Context, err error) {
	// The reason only; never the assertion.
	log.Warn().Str("path", c.Request.URL.Path).Str("remote_addr", c.ClientIP()).Str("reason", err.Error()).Msg("gateway assertion rejected")
	c.Next()
}

// Verify checks one assertion against the request it arrived with.
func (v *GatewayVerifier) Verify(token, method, path, site string) (*gatewayAssertionClaims, error) {
	if len(v.keys) == 0 {
		return nil, errors.New("no gateway keys configured")
	}
	if len(token) > gatewayAssertionMaxSize {
		return nil, errors.New("assertion too large")
	}
	segments := strings.Split(token, ".")
	if len(segments) != 3 {
		return nil, errors.New("assertion is not a compact JWS")
	}

	var header gatewayAssertionHeader
	if err := decodeSegment(segments[0], &header); err != nil {
		return nil, fmt.Errorf("header: %w", err)
	}
	if header.Alg != gatewayAssertionAlg || header.Typ != gatewayAssertionType {
		return nil, errors.New("unexpected alg or typ")
	}
	key, ok := v.keys[header.Kid]
	if !ok {
		return nil, errors.New("unknown kid")
	}
	signature, err := b64url.DecodeString(segments[2])
	if err != nil || len(signature) != ed25519.SignatureSize {
		return nil, errors.New("malformed signature")
	}
	if !ed25519.Verify(key, []byte(segments[0]+"."+segments[1]), signature) {
		return nil, errors.New("signature does not verify")
	}

	var claims gatewayAssertionClaims
	if err := decodeSegment(segments[1], &claims); err != nil {
		return nil, fmt.Errorf("claims: %w", err)
	}
	if claims.Iss != gatewayAssertionIssuer || claims.Aud != gatewayAssertionAudience {
		return nil, errors.New("unexpected iss or aud")
	}
	if claims.Iat == nil || claims.Exp == nil {
		return nil, errors.New("iat and exp are required")
	}
	iat, exp := time.Unix(*claims.Iat, 0), time.Unix(*claims.Exp, 0)
	now := v.now()
	if exp.Before(iat) || exp.Sub(iat) > gatewayAssertionMaxLife {
		return nil, errors.New("invalid lifetime")
	}
	if iat.After(now.Add(gatewayAssertionSkew)) {
		return nil, errors.New("issued in the future")
	}
	if exp.Before(now.Add(-gatewayAssertionSkew)) {
		return nil, errors.New("expired")
	}
	if !gatewayAppPattern.MatchString(claims.App) || (len(claims.AdminSites) == 0 && claims.Sub != "application:"+claims.App) || (len(claims.AdminSites) > 0 && (claims.Sub == "" || strings.TrimSpace(claims.Sub) != claims.Sub || utf8.RuneCountInString(claims.Sub) > 128 || strings.IndexFunc(claims.Sub, unicode.IsControl) >= 0)) {
		return nil, errors.New("invalid application identity")
	}
	if claims.Method != method {
		return nil, errors.New("method mismatch")
	}
	if claims.Path != path {
		return nil, errors.New("path mismatch")
	}
	// The gateway grants exactly the requested site.
	grants := append(append(append([]string{}, claims.ChannelSites...), claims.ReadSites...), claims.AdminSites...)
	if len(grants) != 1 || !gatewaySitePattern.MatchString(grants[0]) || grants[0] != site {
		return nil, errors.New("site not authorized")
	}
	if !gatewayRIDPattern.MatchString(claims.Rid) {
		return nil, errors.New("invalid rid")
	}
	if !v.firstUse(claims.Rid, exp.Add(gatewayAssertionSkew), now) {
		return nil, errors.New("replayed rid")
	}
	return &claims, nil
}

// firstUse records rid until the assertion can no longer be accepted. The cache
// is per process; every gateway request carries a fresh UUID rid.
func (v *GatewayVerifier) firstUse(rid string, until, now time.Time) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	for seen, expiry := range v.seen {
		if expiry.Before(now) {
			delete(v.seen, seen)
		}
	}
	if _, replayed := v.seen[rid]; replayed {
		return false
	}
	v.seen[rid] = until
	return true
}

// decodeSegment requires base64url without padding and one JSON object with only
// known members.
func decodeSegment(segment string, out any) error {
	raw, err := b64url.DecodeString(segment)
	if err != nil {
		return errors.New("not base64url")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return errors.New("not the expected JSON object")
	}
	if decoder.More() {
		return errors.New("trailing data")
	}
	return nil
}

func gatewayGrants(sites []string) map[string]bool {
	grants := make(map[string]bool, len(sites))
	for _, site := range sites {
		grants[site] = true
	}
	return grants
}
