package ssh

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// Stats holds resource metrics collected from a server.
type Stats struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemPercent    float64 `json:"mem_percent"`
	MemTotalMB    int64   `json:"mem_total_mb"`
	MemUsedMB     int64   `json:"mem_used_mb"`
	DiskPercent   float64 `json:"disk_percent"`
	DiskTotalGB   int64   `json:"disk_total_gb"`
	DiskUsedGB    int64   `json:"disk_used_gb"`
	UptimeSeconds int64   `json:"uptime_seconds"`
}

// statsCommand emits marker-delimited blocks parsed by ParseStats.
const statsCommand = `echo '###MEM###'; cat /proc/meminfo; ` +
	`echo '###DISK###'; df -kP /; ` +
	`echo '###CPU1###'; grep '^cpu ' /proc/stat; sleep 0.3; ` +
	`echo '###CPU2###'; grep '^cpu ' /proc/stat; ` +
	`echo '###UP###'; cat /proc/uptime`

// CollectStats runs the stats command over the runner and parses the result.
func CollectStats(ctx context.Context, r Runner) (*Stats, error) {
	out, stderr, err := r.Run(ctx, statsCommand)
	if err != nil {
		return nil, fmt.Errorf("collect stats: %w (%s)", err, stderr)
	}
	return ParseStats(out)
}

// ParseStats parses the marker-delimited output of statsCommand.
func ParseStats(out string) (*Stats, error) {
	sections := splitSections(out)
	s := &Stats{}

	if mem := sections["MEM"]; mem != "" {
		total, avail := parseMeminfo(mem)
		if total > 0 {
			used := total - avail
			s.MemTotalMB = total / 1024
			s.MemUsedMB = used / 1024
			s.MemPercent = round1(float64(used) / float64(total) * 100)
		}
	}

	if disk := sections["DISK"]; disk != "" {
		s.DiskTotalGB, s.DiskUsedGB, s.DiskPercent = parseDf(disk)
	}

	if c1, c2 := sections["CPU1"], sections["CPU2"]; c1 != "" && c2 != "" {
		s.CPUPercent = parseCPU(c1, c2)
	}

	if up := sections["UP"]; up != "" {
		fields := strings.Fields(up)
		if len(fields) > 0 {
			if v, err := strconv.ParseFloat(fields[0], 64); err == nil {
				s.UptimeSeconds = int64(v)
			}
		}
	}
	return s, nil
}

func splitSections(out string) map[string]string {
	sections := map[string]string{}
	var cur string
	var b strings.Builder
	flush := func() {
		if cur != "" {
			sections[cur] = b.String()
		}
		b.Reset()
	}
	for _, line := range strings.Split(out, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "###") && strings.HasSuffix(t, "###") {
			flush()
			cur = strings.Trim(t, "#")
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	flush()
	return sections
}

// parseMeminfo returns MemTotal and MemAvailable in kB.
func parseMeminfo(block string) (total, avail int64) {
	for _, line := range strings.Split(block, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, _ := strconv.ParseInt(fields[1], 10, 64)
		switch fields[0] {
		case "MemTotal:":
			total = val
		case "MemAvailable:":
			avail = val
		}
	}
	return total, avail
}

// parseDf parses `df -kP /` output (1K blocks) into GB totals and percent.
func parseDf(block string) (totalGB, usedGB int64, percent float64) {
	lines := strings.Split(strings.TrimSpace(block), "\n")
	if len(lines) < 2 {
		return 0, 0, 0
	}
	fields := strings.Fields(lines[len(lines)-1])
	if len(fields) < 5 {
		return 0, 0, 0
	}
	totalK, _ := strconv.ParseInt(fields[1], 10, 64)
	usedK, _ := strconv.ParseInt(fields[2], 10, 64)
	pct := strings.TrimSuffix(fields[4], "%")
	p, _ := strconv.ParseFloat(pct, 64)
	return totalK / (1024 * 1024), usedK / (1024 * 1024), p
}

// parseCPU computes busy percentage from two /proc/stat cpu samples.
func parseCPU(sample1, sample2 string) float64 {
	idle1, total1 := cpuTotals(sample1)
	idle2, total2 := cpuTotals(sample2)
	dTotal := total2 - total1
	dIdle := idle2 - idle1
	if dTotal <= 0 {
		return 0
	}
	busy := float64(dTotal-dIdle) / float64(dTotal) * 100
	return round1(busy)
}

func cpuTotals(sample string) (idle, total int64) {
	fields := strings.Fields(strings.TrimSpace(sample))
	// fields[0] == "cpu", then user nice system idle iowait irq softirq ...
	for i := 1; i < len(fields); i++ {
		v, err := strconv.ParseInt(fields[i], 10, 64)
		if err != nil {
			continue
		}
		total += v
		if i == 4 { // idle column
			idle = v
		}
	}
	return idle, total
}

func round1(f float64) float64 {
	return float64(int64(f*10+0.5)) / 10
}
