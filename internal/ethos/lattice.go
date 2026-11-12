package ethos

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	ZoneTrust    = "trust"
	ZoneAuth     = "auth"
	ZoneSecure   = "secure"
	ZoneAdmin    = "admin"
	ZoneIdentity = "identity"
)

type PrimeVector struct {
	Value     int     `json:"value"`
	Harmonic  float64 `json:"harmonic"`
	Resonance float64 `json:"resonance"`
}

type HarmonicBase struct {
	Frequency float64   `json:"frequency"`
	Amplitude float64   `json:"amplitude"`
	Phase     float64   `json:"phase"`
	Overtones []float64 `json:"overtones"`
}

type ResonanceField struct {
	FieldStrength float64 `json:"field_strength"`
	Coherence     float64 `json:"coherence"`
	Stability     float64 `json:"stability"`
	Entropy       float64 `json:"entropy"`
}

type LatticeMatrix struct {
	PrimeVectors   []PrimeVector  `json:"prime_vectors"`
	HarmonicBase   HarmonicBase   `json:"harmonic_base"`
	ResonanceField ResonanceField `json:"resonance_field"`
}

type LatticePoint struct {
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Z         float64 `json:"z"`
	Energy    float64 `json:"energy"`
	Harmonic  float64 `json:"harmonic"`
	Resonance float64 `json:"resonance"`
}

type CodexEthosLattice struct {
	latticeMatrix LatticeMatrix
	ethosCore     map[string]bool
	quantumState  map[string]string
	mu            sync.RWMutex
	now           func() time.Time
}

func NewCodexEthosLattice() *CodexEthosLattice {
	lattice := &CodexEthosLattice{
		ethosCore:    map[string]bool{},
		quantumState: map[string]string{},
		now:          time.Now,
	}
	lattice.initializeLattice()
	lattice.initializeEthos()
	return lattice
}

func (lattice *CodexEthosLattice) initializeLattice() {
	lattice.latticeMatrix = LatticeMatrix{
		PrimeVectors:   lattice.generatePrimeVectors(),
		HarmonicBase:   lattice.generateHarmonicBase(),
		ResonanceField: lattice.generateResonanceField(),
	}
}

func (lattice *CodexEthosLattice) initializeEthos() {
	lattice.ethosCore = map[string]bool{
		"integrity":    true,
		"authenticity": true,
		"harmony":      true,
		"balance":      true,
	}
}

func (lattice *CodexEthosLattice) generatePrimeVectors() []PrimeVector {
	primes := []int{2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47}
	vectors := make([]PrimeVector, 0, len(primes))
	for _, prime := range primes {
		vectors = append(vectors, PrimeVector{
			Value:     prime,
			Harmonic:  lattice.calculateHarmonic(float64(prime)),
			Resonance: lattice.calculateResonance(float64(prime)),
		})
	}
	return vectors
}

func (lattice *CodexEthosLattice) generateHarmonicBase() HarmonicBase {
	return HarmonicBase{
		Frequency: 432,
		Amplitude: 1.0,
		Phase:     0,
		Overtones: []float64{432, 864, 1296, 1728, 2160},
	}
}

func (lattice *CodexEthosLattice) generateResonanceField() ResonanceField {
	return ResonanceField{
		FieldStrength: 1.0,
		Coherence:     0.95,
		Stability:     0.98,
		Entropy:       0.02,
	}
}

func (lattice *CodexEthosLattice) calculateHarmonic(value float64) float64 {
	return math.Mod(value*432, 1000) / 1000
}

func (lattice *CodexEthosLattice) calculateResonance(value float64) float64 {
	return math.Sin((value*math.Pi)/180)*0.5 + 0.5
}

func (lattice *CodexEthosLattice) GetLatticeMatrix() LatticeMatrix {
	return lattice.latticeMatrix
}

func (lattice *CodexEthosLattice) GetEthosCore() map[string]bool {
	copyCore := make(map[string]bool, len(lattice.ethosCore))
	for key, value := range lattice.ethosCore {
		copyCore[key] = value
	}
	return copyCore
}

func (lattice *CodexEthosLattice) GenerateQuantumState(seed string) string {
	stateBytes := sha256.Sum256([]byte(seed + lattice.now().UTC().Format(time.RFC3339Nano)))
	state := hex.EncodeToString(stateBytes[:])

	lattice.mu.Lock()
	lattice.quantumState[seed] = state
	lattice.mu.Unlock()

	return state
}

func (lattice *CodexEthosLattice) VerifyQuantumState(seed string, state string) bool {
	lattice.mu.RLock()
	storedState, ok := lattice.quantumState[seed]
	lattice.mu.RUnlock()
	return ok && storedState == state
}

func (lattice *CodexEthosLattice) CalculateEthosSignature(data string) string {
	if data == "" {
		return ""
	}

	ethosKeys := make([]string, 0, len(lattice.ethosCore))
	for key := range lattice.ethosCore {
		ethosKeys = append(ethosKeys, key)
	}
	sort.Strings(ethosKeys)

	builder := strings.Builder{}
	for index, character := range []byte(data) {
		ethosKey := ethosKeys[index%len(ethosKeys)]
		modifier := 0
		if lattice.ethosCore[ethosKey] {
			modifier = 1
		}
		builder.WriteString(fmt.Sprintf("%x", (int(character)+modifier)%16))
	}

	return builder.String()
}

func (lattice *CodexEthosLattice) VerifyEthosSignature(data string, signature string) bool {
	return lattice.CalculateEthosSignature(data) == signature
}

func (lattice *CodexEthosLattice) GetLatticePoint(coordinates map[string]float64) LatticePoint {
	x := coordinates["x"]
	y := coordinates["y"]
	z := coordinates["z"]

	return LatticePoint{
		X:         x,
		Y:         y,
		Z:         z,
		Energy:    math.Sqrt(x*x + y*y + z*z),
		Harmonic:  lattice.calculateHarmonic(x + y + z),
		Resonance: lattice.calculateResonance(x * y * z),
	}
}

func (lattice *CodexEthosLattice) GenerateSecureHash(input string, salt string) string {
	ethosSignature := lattice.CalculateEthosSignature(input)
	latticeData, err := json.Marshal(lattice.latticeMatrix)
	if err != nil {
		latticeData = []byte("{}")
	}
	combined := input + salt + ethosSignature + string(latticeData)
	hashBytes := sha512.Sum512([]byte(combined))
	return hex.EncodeToString(hashBytes[:])
}

func (lattice *CodexEthosLattice) VerifySecureHash(input string, hash string, salt string) bool {
	return lattice.GenerateSecureHash(input, salt) == hash
}

func (lattice *CodexEthosLattice) Validate(zone string, context map[string]any) (bool, error) {
	if !isValidZone(zone) {
		return false, fmt.Errorf("invalid zone: %s", zone)
	}

	requiredFields := []string{"user_id", "tiu", "hash"}
	for _, field := range requiredFields {
		if isEmpty(context[field]) {
			return false, nil
		}
	}

	switch zone {
	case ZoneTrust:
		return lattice.validateTrustZone(context), nil
	case ZoneAuth:
		return lattice.validateAuthZone(context), nil
	case ZoneSecure:
		return lattice.validateSecureZone(context), nil
	case ZoneAdmin:
		return lattice.validateAdminZone(context), nil
	case ZoneIdentity:
		return lattice.validateIdentityZone(context), nil
	default:
		return false, nil
	}
}

func (lattice *CodexEthosLattice) validateTrustZone(context map[string]any) bool {
	if trustScore, ok := numericValue(context["trust_score"]); ok && trustScore < 0.5 {
		return false
	}
	if deviceID, ok := context["device_id"]; ok && isEmpty(deviceID) {
		return false
	}
	return true
}

func (lattice *CodexEthosLattice) validateAuthZone(context map[string]any) bool {
	email, _ := context["email"].(string)
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

func (lattice *CodexEthosLattice) validateSecureZone(context map[string]any) bool {
	return !isEmpty(context["device_key"])
}

func (lattice *CodexEthosLattice) validateAdminZone(context map[string]any) bool {
	trustScore, ok := numericValue(context["trust_score"])
	return lattice.validateSecureZone(context) && ok && trustScore >= 0.8
}

func (lattice *CodexEthosLattice) validateIdentityZone(context map[string]any) bool {
	requiredFields := []string{"user_id", "hash", "tiu"}
	for _, field := range requiredFields {
		if isEmpty(context[field]) {
			return false
		}
	}
	hashValue, _ := context["hash"].(string)
	return len(hashValue) >= 32
}

func (lattice *CodexEthosLattice) ReportSuspiciousActivity(userIdentifier string, metadata map[string]any) {
	fields := map[string]any{
		"user_identifier": userIdentifier,
		"zone":            ZoneTrust,
		"timestamp":       lattice.now().UTC().Format(time.RFC3339),
	}
	for key, value := range metadata {
		fields[key] = value
	}
	log.Warn().Fields(fields).Msg("Suspicious activity detected")
}

func isValidZone(zone string) bool {
	switch zone {
	case ZoneTrust, ZoneAuth, ZoneSecure, ZoneAdmin, ZoneIdentity:
		return true
	default:
		return false
	}
}

func isEmpty(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(typed) == ""
	default:
		return false
	}
}

func numericValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	default:
		return 0, false
	}
}