// Package main provides system information collection using gopsutil.
//
// This file implements the Collector service that gathers CPU, memory, disk,
// and host information and displays it in text or JSON format.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/petabytecl/gaz"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

// SystemInfo contains collected system information.
type SystemInfo struct {
	// Host information
	Hostname string `json:"hostname"`
	Platform string `json:"platform"`
	Uptime   uint64 `json:"uptime"`

	// CPU information
	CPUModel string  `json:"cpu_model"`
	CPUCores int     `json:"cpu_cores"`
	CPUUsage float64 `json:"cpu_usage"`

	// Memory information
	MemTotal   uint64  `json:"mem_total"`
	MemUsed    uint64  `json:"mem_used"`
	MemPercent float64 `json:"mem_percent"`

	// Disk information
	DiskTotal   uint64  `json:"disk_total"`
	DiskUsed    uint64  `json:"disk_used"`
	DiskPercent float64 `json:"disk_percent"`
}

// Collector gathers system information and displays it.
type Collector struct {
	cfg *SystemInfoConfig
}

// NewCollector creates a new Collector with config for dynamic format reading.
// Format is read dynamically (not cached) to support CLI flag overrides.
func NewCollector(c *gaz.Container) (*Collector, error) {
	cfg, err := gaz.Resolve[*SystemInfoConfig](c)
	if err != nil {
		return nil, err
	}
	return &Collector{cfg: cfg}, nil
}

// Collect gathers system information from gopsutil.
func (c *Collector) Collect() (*SystemInfo, error) {
	info := &SystemInfo{}

	// Host info
	hostInfo, err := host.Info()
	if err == nil {
		info.Hostname = hostInfo.Hostname
		info.Platform = hostInfo.Platform
		info.Uptime = hostInfo.Uptime
	}

	// CPU info
	cpuInfo, err := cpu.Info()
	if err == nil && len(cpuInfo) > 0 {
		info.CPUModel = cpuInfo[0].ModelName
		info.CPUCores = int(cpuInfo[0].Cores)
	}

	// CPU usage - use 100ms interval to get actual reading (not cached empty on first call)
	cpuPercent, err := cpu.Percent(100*time.Millisecond, false)
	if err == nil && len(cpuPercent) > 0 {
		info.CPUUsage = cpuPercent[0]
	}

	// Memory info
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		info.MemTotal = memInfo.Total
		info.MemUsed = memInfo.Used
		info.MemPercent = memInfo.UsedPercent
	}

	// Disk info (root partition)
	diskInfo, err := disk.Usage("/")
	if err == nil {
		info.DiskTotal = diskInfo.Total
		info.DiskUsed = diskInfo.Used
		info.DiskPercent = diskInfo.UsedPercent
	}

	return info, nil
}

// Display outputs the system information in the configured format.
// Format is read dynamically from config to support CLI flag overrides.
func (c *Collector) Display(info *SystemInfo) error {
	switch c.cfg.Format() {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(info)
	default: // "text"
		return c.displayText(info)
	}
}

// displayText outputs system info as a formatted text table.
func (c *Collector) displayText(info *SystemInfo) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintln(w, "System Information")
	fmt.Fprintln(w, "─────────────────")
	fmt.Fprintf(w, "Hostname:\t%s\n", info.Hostname)
	fmt.Fprintf(w, "Platform:\t%s\n", info.Platform)
	fmt.Fprintf(w, "Uptime:\t%s\n", formatDuration(info.Uptime))

	fmt.Fprintln(w)
	fmt.Fprintln(w, "CPU")
	fmt.Fprintf(w, "Model:\t%s\n", info.CPUModel)
	fmt.Fprintf(w, "Cores:\t%d\n", info.CPUCores)
	fmt.Fprintf(w, "Usage:\t%.1f%%\n", info.CPUUsage)

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Memory")
	fmt.Fprintf(w, "Total:\t%s\n", formatBytes(info.MemTotal))
	fmt.Fprintf(w, "Used:\t%s (%.1f%%)\n", formatBytes(info.MemUsed), info.MemPercent)

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Disk (/)")
	fmt.Fprintf(w, "Total:\t%s\n", formatBytes(info.DiskTotal))
	fmt.Fprintf(w, "Used:\t%s (%.1f%%)\n", formatBytes(info.DiskUsed), info.DiskPercent)

	return w.Flush()
}

// formatBytes converts bytes to human-readable format (KB, MB, GB).
func formatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// formatDuration converts seconds to human-readable duration (Xd Xh Xm).
func formatDuration(seconds uint64) string {
	days := seconds / (24 * 3600)
	seconds %= 24 * 3600
	hours := seconds / 3600
	seconds %= 3600
	minutes := seconds / 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
