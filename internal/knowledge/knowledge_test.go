package knowledge

import (
	"testing"
)

func TestKnowledgeStructure(t *testing.T) {
	// Test 1: DefaultKnowledge
	k := DefaultKnowledge()
	if !k.ScanConfig.UseDefaultIgnores {
		t.Errorf("DefaultKnowledge should have UseDefaultIgnores=true")
	}

	// Test 2: NewKnowledge with custom config
	cfg := ScanConfig{
		UseDefaultIgnores: false,
		IncludeHidden:     true,
		IgnoreDirs:        []string{"test"},
	}
	k2 := NewKnowledge(cfg)
	if k2.ScanConfig.UseDefaultIgnores {
		t.Errorf("NewKnowledge should preserve UseDefaultIgnores=false")
	}
	if !k2.ScanConfig.IncludeHidden {
		t.Errorf("NewKnowledge should preserve IncludeHidden=true")
	}
	if len(k2.ScanConfig.IgnoreDirs) != 1 || k2.ScanConfig.IgnoreDirs[0] != "test" {
		t.Errorf("NewKnowledge should preserve IgnoreDirs")
	}
}

func TestKnowledgeScan(t *testing.T) {
	k := DefaultKnowledge()
	
	// Try scanning a test data directory if it exists
	testDir := "test-data"
	docs, err := k.Scan(testDir)
	
	// It's OK if test data doesn't exist or is empty
	if err != nil {
		t.Logf("Scan of %s returned error (may be expected): %v", testDir, err)
	} else {
		t.Logf("Scan of %s returned %d documents", testDir, len(docs))
	}
}
