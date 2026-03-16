package knowledge

import (
	"testing"
	"time"
)

func TestTokenize_Basic(t *testing.T) {
	tok := NewTokenizer(TokenizerConfig{})
	tokens := tok.Tokenize("Authentication Token Validation!")
	want := []string{"authentication", "token", "validation"}
	if !equalStringSlices(tokens, want) {
		t.Errorf("Tokenize = %v, want %v", tokens, want)
	}
}

func TestTokenize_PreservesHyphens(t *testing.T) {
	tok := NewTokenizer(TokenizerConfig{})
	tokens := tok.Tokenize("api-gateway")
	if len(tokens) != 1 || tokens[0] != "api-gateway" {
		t.Errorf("Tokenize(\"api-gateway\") = %v, want [api-gateway]", tokens)
	}
}

func TestTokenize_HyphenAtWordBoundary(t *testing.T) {
	tok := NewTokenizer(TokenizerConfig{})
	// Hyphen at start/end should not be preserved in the same token.
	tokens := tok.Tokenize("-leading and trailing-")
	// "leading" and "trailing" should appear without hyphens; "and" depends on stop words.
	for _, tk := range tokens {
		if len(tk) > 0 && (tk[0] == '-' || tk[len(tk)-1] == '-') {
			t.Errorf("token %q has leading/trailing hyphen", tk)
		}
	}
}

func TestTokenize_Lowercase(t *testing.T) {
	tok := NewTokenizer(TokenizerConfig{})
	tokens := tok.Tokenize("UPPER lower Mixed")
	for _, tk := range tokens {
		for _, r := range tk {
			if r >= 'A' && r <= 'Z' {
				t.Errorf("token %q contains uppercase letter", tk)
			}
		}
	}
}

func TestTokenize_StopWords(t *testing.T) {
	tok := NewTokenizer(TokenizerConfig{
		RemoveStopWords: true,
		StopWords:       map[string]struct{}{"the": {}, "is": {}, "a": {}},
	})
	tokens := tok.Tokenize("the cat is a mammal")
	for _, tk := range tokens {
		switch tk {
		case "the", "is", "a":
			t.Errorf("stop word %q should have been removed", tk)
		}
	}
	// "cat" and "mammal" must still be present.
	found := make(map[string]bool)
	for _, tk := range tokens {
		found[tk] = true
	}
	for _, want := range []string{"cat", "mammal"} {
		if !found[want] {
			t.Errorf("expected %q in tokens", want)
		}
	}
}

func TestTokenize_MinLength(t *testing.T) {
	tok := NewTokenizer(TokenizerConfig{MinTokenLen: 3})
	tokens := tok.Tokenize("a ab abc abcd")
	for _, tk := range tokens {
		if len([]rune(tk)) < 3 {
			t.Errorf("token %q is shorter than MinTokenLen=3", tk)
		}
	}
}

func TestTokenize_Unicode(t *testing.T) {
	tok := NewTokenizer(TokenizerConfig{})
	// Cyrillic, Japanese, and accented chars.
	tokens := tok.Tokenize("Привет мир")
	if len(tokens) != 2 {
		t.Errorf("expected 2 Unicode tokens, got %d: %v", len(tokens), tokens)
	}
}

func TestTokenize_EmptyInput(t *testing.T) {
	tok := NewTokenizer(TokenizerConfig{})
	tokens := tok.Tokenize("")
	if len(tokens) != 0 {
		t.Errorf("expected 0 tokens for empty input, got %d", len(tokens))
	}
}

func TestTokenize_PunctuationOnly(t *testing.T) {
	tok := NewTokenizer(TokenizerConfig{})
	tokens := tok.Tokenize("!@#$%^&*()")
	if len(tokens) != 0 {
		t.Errorf("expected 0 tokens for punctuation-only input, got %v", tokens)
	}
}

func TestTokenize_URL(t *testing.T) {
	tok := NewTokenizer(TokenizerConfig{MinTokenLen: 2})
	// URLs should be split on non-word characters.
	tokens := tok.Tokenize("https://example.com/api/v2")
	// Should produce tokens from the URL parts (https, example, com, api, v2).
	if len(tokens) == 0 {
		t.Error("expected some tokens from URL")
	}
}

func TestTokenize_Numbers(t *testing.T) {
	tok := NewTokenizer(TokenizerConfig{})
	tokens := tok.Tokenize("v2 api3 42")
	// Numbers should be kept as part of tokens.
	found := make(map[string]bool)
	for _, tk := range tokens {
		found[tk] = true
	}
	if !found["v2"] {
		t.Errorf("expected v2, got %v", tokens)
	}
}

func TestTokenizeWithDefaults_AcceptsProseText(t *testing.T) {
	// A typical markdown document excerpt.
	text := "The authentication service validates JWT tokens and returns user claims."
	tokens := TokenizeWithDefaults(text)
	// Stop words should be removed; content words should remain.
	found := make(map[string]bool)
	for _, tk := range tokens {
		found[tk] = true
	}

	stopWords := []string{"the", "and"}
	for _, sw := range stopWords {
		if found[sw] {
			t.Errorf("stop word %q should be removed", sw)
		}
	}

	// "returns" is NOT a stop word — it is a content verb that should be kept.
	contentWords := []string{"authentication", "service", "validates", "jwt", "tokens", "user", "claims", "returns"}
	for _, cw := range contentWords {
		if !found[cw] {
			t.Errorf("content word %q should be present", cw)
		}
	}
}

func BenchmarkTokenize1000Words(b *testing.B) {
	// Generate a ~1000-word document.
	words := make([]byte, 0, 6000)
	wordList := []string{"authentication", "service", "gateway", "token", "validation",
		"request", "response", "handler", "middleware", "database"}
	for range 100 {
		for _, w := range wordList {
			words = append(words, w...)
			words = append(words, ' ')
		}
	}
	text := string(words)
	tok := NewTokenizer(DefaultTokenizerConfig())

	b.ResetTimer()
	for range b.N {
		_ = tok.Tokenize(text)
	}
}

// equalStringSlices returns true if a and b contain the same elements in order.
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Ensure time package is used (imported for potential fixture use).
var _ = time.Now
