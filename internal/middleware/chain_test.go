package middleware

import (
	"testing"

	"github.com/web3connected/codexsecure-api/internal/analyzer"
	"github.com/web3connected/codexsecure-api/internal/fingerprint"
	"github.com/web3connected/codexsecure-api/internal/policy"
)

func TestChainOrderDeclaresExpectedStageSequence(t *testing.T) {
	chain := NewChain(analyzer.NewAnalyzer(policy.NewRegistry()), policy.NewRegistry(), fingerprint.NewAnalyzer(nil), nil)

	got := chain.Order()
	want := []string{
		StageRequestIdentity,
		StageZoneDetection,
		StageMethodValidation,
		StageTIUValidation,
		StageAnalysisEnforcement,
		StageExecutionSync,
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d stages, got %d", len(want), len(got))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected stage %d to be %q, got %q", i, want[i], got[i])
		}
	}
}

func TestBuildWithExecutionSyncAppendsExecutionSyncLast(t *testing.T) {
	registry := policy.NewRegistry()
	chain := NewChain(analyzer.NewAnalyzer(registry), registry, fingerprint.NewAnalyzer(nil), nil)

	base := chain.Build()
	withExecutionSync := chain.BuildWithExecutionSync("test")

	if len(withExecutionSync) != len(base)+1 {
		t.Fatalf("expected execution sync chain to add one middleware, got %d vs %d", len(withExecutionSync), len(base))
	}
	if len(withExecutionSync) == 0 {
		t.Fatal("expected non-empty middleware chain")
	}
	if len(base) == 0 {
		t.Fatal("expected non-empty base middleware chain")
	}
	if len(withExecutionSync[:len(base)]) != len(base) {
		t.Fatal("expected execution sync to be appended after the base chain")
	}
}