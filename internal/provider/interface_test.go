package provider

import "testing"

func TestNormalizedMaxTokensUsesExplicitLimit(t *testing.T) {
	options := GenerateOptions{MaxTokens: 512}

	if got := normalizedMaxTokens(options); got != 512 {
		t.Fatalf("normalizedMaxTokens() = %d, want 512", got)
	}
}

func TestNormalizedMaxTokensDefaultsTo4096(t *testing.T) {
	if got := normalizedMaxTokens(GenerateOptions{}); got != 4096 {
		t.Fatalf("normalizedMaxTokens() = %d, want 4096", got)
	}
}
