package ethos

import (
	"testing"
	"time"
)

func TestGenerateAndVerifyQuantumState(t *testing.T) {
	lattice := NewCodexEthosLattice()
	lattice.now = func() time.Time { return time.Unix(1713700000, 0).UTC() }

	state := lattice.GenerateQuantumState("seed-1")
	if state == "" {
		t.Fatal("expected quantum state to be generated")
	}
	if !lattice.VerifyQuantumState("seed-1", state) {
		t.Fatal("expected quantum state verification to succeed")
	}
}

func TestEthosSignatureRoundTrip(t *testing.T) {
	lattice := NewCodexEthosLattice()
	signature := lattice.CalculateEthosSignature("codex")
	if signature == "" {
		t.Fatal("expected signature")
	}
	if !lattice.VerifyEthosSignature("codex", signature) {
		t.Fatal("expected signature verification to succeed")
	}
}

func TestGenerateSecureHashRoundTrip(t *testing.T) {
	lattice := NewCodexEthosLattice()
	hash := lattice.GenerateSecureHash("payload", "salt")
	if len(hash) != 128 {
		t.Fatalf("expected sha512 hex hash length, got %d", len(hash))
	}
	if !lattice.VerifySecureHash("payload", hash, "salt") {
		t.Fatal("expected secure hash verification to succeed")
	}
}

func TestValidateIdentityZone(t *testing.T) {
	lattice := NewCodexEthosLattice()
	valid, err := lattice.Validate(ZoneIdentity, map[string]any{
		"user_id": "123",
		"tiu":     "token",
		"hash":    "12345678901234567890123456789012",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Fatal("expected identity zone context to validate")
	}
}

func TestInvalidZoneFails(t *testing.T) {
	lattice := NewCodexEthosLattice()
	_, err := lattice.Validate("unknown-zone", map[string]any{})
	if err == nil {
		t.Fatal("expected invalid zone error")
	}
}