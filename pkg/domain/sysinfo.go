package domain

// SysInfo represents a snapshot of system resource usage.
type SysInfo struct {
	CPUUsage    float64 // Percentage
	MemoryUsed  uint64  // Bytes
	MemoryTotal uint64  // Bytes
	MemoryPct   float64 // Percentage
}
