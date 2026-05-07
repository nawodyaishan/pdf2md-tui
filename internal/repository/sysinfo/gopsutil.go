package sysinfo

import (
	"context"
	"time"

	"github.com/nawodyaishan/pdf2md-tui/internal/domain"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// Provider defines the interface for system information.
type Provider interface {
	GetSnapshot() (domain.SysInfo, error)
}

// GopsutilProvider implements Provider using the gopsutil library.
type GopsutilProvider struct{}

// NewProvider creates a new GopsutilProvider.
func NewProvider() Provider {
	return &GopsutilProvider{}
}

// GetSnapshot captures a point-in-time snapshot of CPU and Memory usage.
func (p *GopsutilProvider) GetSnapshot() (domain.SysInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// CPU percent over a short interval
	percents, err := cpu.PercentWithContext(ctx, 100*time.Millisecond, false)
	cpuPct := 0.0
	if err == nil && len(percents) > 0 {
		cpuPct = percents[0]
	}

	// Memory usage
	v, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return domain.SysInfo{CPUUsage: cpuPct}, err
	}

	return domain.SysInfo{
		CPUUsage:    cpuPct,
		MemoryUsed:  v.Used,
		MemoryTotal: v.Total,
		MemoryPct:   v.UsedPercent,
	}, nil
}
