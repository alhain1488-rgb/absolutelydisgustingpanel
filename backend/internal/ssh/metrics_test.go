package ssh

import (
	"context"
	"testing"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/ssh/sshtest"
)

const sampleStats = `###MEM###
MemTotal:        2048000 kB
MemFree:          500000 kB
MemAvailable:    1024000 kB
###DISK###
Filesystem     1024-blocks     Used Available Capacity Mounted on
/dev/vda1         41943040 20971520  20971520      50% /
###CPU1###
cpu  100 0 100 800 0 0 0 0 0 0
###CPU2###
cpu  200 0 200 1400 0 0 0 0 0 0
###UP###
123456.78 100000.00
`

func TestParseStats(t *testing.T) {
	s, err := ParseStats(sampleStats)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if s.MemTotalMB != 2000 {
		t.Errorf("MemTotalMB: got %d want 2000", s.MemTotalMB)
	}
	if s.MemUsedMB != 1000 {
		t.Errorf("MemUsedMB: got %d want 1000", s.MemUsedMB)
	}
	if s.MemPercent != 50 {
		t.Errorf("MemPercent: got %v want 50", s.MemPercent)
	}
	if s.DiskTotalGB != 40 || s.DiskUsedGB != 20 {
		t.Errorf("disk GB: got %d/%d want 40/20", s.DiskTotalGB, s.DiskUsedGB)
	}
	if s.DiskPercent != 50 {
		t.Errorf("DiskPercent: got %v want 50", s.DiskPercent)
	}
	// CPU: total delta=800, idle delta=600 -> busy 25%.
	if s.CPUPercent != 25 {
		t.Errorf("CPUPercent: got %v want 25", s.CPUPercent)
	}
	if s.UptimeSeconds != 123456 {
		t.Errorf("Uptime: got %d want 123456", s.UptimeSeconds)
	}
}

func TestCollectStats_Runner(t *testing.T) {
	fake := sshtest.NewFakeRunner()
	fake.Default = sshtest.Response{Stdout: sampleStats}
	s, err := CollectStats(context.Background(), fake)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if s.MemPercent != 50 {
		t.Fatalf("expected 50%% mem, got %v", s.MemPercent)
	}
}

func TestParseStats_Empty(t *testing.T) {
	s, err := ParseStats("")
	if err != nil {
		t.Fatalf("parse empty: %v", err)
	}
	if s.MemTotalMB != 0 || s.CPUPercent != 0 {
		t.Fatalf("expected zero stats, got %+v", s)
	}
}
