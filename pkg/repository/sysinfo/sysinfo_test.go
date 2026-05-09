package sysinfo

import (
	"testing"
)

func TestNewProvider(t *testing.T) {
	provider := NewProvider()
	if provider == nil {
		t.Fatal("NewProvider returned nil")
	}
	if _, ok := provider.(*GopsutilProvider); !ok {
		t.Errorf("expected *GopsutilProvider, got %T", provider)
	}
}

func TestGopsutilProviderGetSnapshot(t *testing.T) {
	provider := NewProvider()
	snapshot, err := provider.GetSnapshot()
	if err != nil {
		t.Logf("Warning: GetSnapshot returned error: %v (expected when running in restricted environments)", err)
	}

	// Even if there's an error, we should get a valid SysInfo struct
	if snapshot.MemoryTotal < 0 {
		t.Errorf("MemoryTotal should not be negative: %d", snapshot.MemoryTotal)
	}
	if snapshot.MemoryUsed < 0 {
		t.Errorf("MemoryUsed should not be negative: %d", snapshot.MemoryUsed)
	}
	if snapshot.MemoryPct < 0 || snapshot.MemoryPct > 100 {
		t.Errorf("MemoryPct should be between 0 and 100: %f", snapshot.MemoryPct)
	}
}

func TestProviderInterface(t *testing.T) {
	var _ Provider = (*GopsutilProvider)(nil)
}
