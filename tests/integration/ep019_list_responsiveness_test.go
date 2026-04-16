//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"pa/internal/jobs"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

type listProfile struct {
	Name           string `json:"name"`
	JobCount       int    `json:"job_count"`
	Samples        int    `json:"samples"`
	P95ThresholdMS int    `json:"p95_threshold_ms"`
}

type profileSet struct {
	Profiles []listProfile `json:"profiles"`
}

func loadListProfiles(t *testing.T) map[string]listProfile {
	t.Helper()
	path := filepath.Join("testdata", "ep019", "list_responsiveness_profiles.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var cfg profileSet
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	out := make(map[string]listProfile, len(cfg.Profiles))
	for _, p := range cfg.Profiles {
		out[p.Name] = p
	}
	return out
}

func p95Millis(durations []time.Duration) float64 {
	if len(durations) == 0 {
		return 0
	}
	ms := make([]float64, len(durations))
	for i := range durations {
		ms[i] = float64(durations[i].Milliseconds())
	}
	sort.Float64s(ms)
	idx := int(math.Ceil(0.95*float64(len(ms)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(ms) {
		idx = len(ms) - 1
	}
	return ms[idx]
}

func assertListResponsiveness(profile listProfile, p95 float64) error {
	if p95 > float64(profile.P95ThresholdMS) {
		return fmt.Errorf("profile %q threshold exceeded: p95=%.2fms > %dms", profile.Name, p95, profile.P95ThresholdMS)
	}
	return nil
}

// Covers AC-19.022: profile evaluation fails when measured responsiveness exceeds configured threshold.
func TestEP019_ListResponsiveness_ThresholdViolation(t *testing.T) {
	profile := listProfile{Name: "strict", P95ThresholdMS: 10}
	durations := []time.Duration{11 * time.Millisecond, 13 * time.Millisecond, 15 * time.Millisecond}
	p95 := p95Millis(durations)
	if err := assertListResponsiveness(profile, p95); err == nil {
		t.Fatalf("expected threshold violation, got nil")
	}
}

// Covers AC-19.022: list responsiveness is validated against selected deployment profile.
func TestEP019_ListResponsiveness_ProfileAcceptance(t *testing.T) {
	profiles := loadListProfiles(t)
	selected := os.Getenv("PA_LIST_RESPONSIVENESS_PROFILE")
	if selected == "" {
		selected = "baseline"
	}
	profile, ok := profiles[selected]
	if !ok {
		t.Fatalf("unknown PA_LIST_RESPONSIVENESS_PROFILE=%q", selected)
	}

	store, err := jobs.Open(filepath.Join(t.TempDir(), "jobs.sqlite"))
	if err != nil {
		t.Fatalf("jobs.Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	mgr := jobs.NewManager(store, nil, slog.New(slog.DiscardHandler))
	ctx := context.Background()

	for i := 0; i < profile.JobCount; i++ {
		_, err := store.CreateJob(ctx, jobs.JobInput{
			Name:           fmt.Sprintf("job-%03d", i),
			ScheduleExpr:   "0 9 * * *",
			TimeZone:       "UTC",
			Instruction:    "Collect AI digest",
			DeliveryChatID: 1,
			Status:         jobs.StatusActive,
			OverlapPolicy:  jobs.OverlapSingleInstance,
			TimeoutPolicy:  jobs.TimeoutCancelAfter,
		})
		if err != nil {
			t.Fatalf("CreateJob(%d): %v", i, err)
		}
	}

	durations := make([]time.Duration, 0, profile.Samples)
	for i := 0; i < profile.Samples; i++ {
		start := time.Now()
		reply, handled, err := mgr.HandleCommand(ctx, 100, "/jobs list")
		if err != nil || !handled {
			t.Fatalf("HandleCommand(list): err=%v handled=%v", err, handled)
		}
		if reply == "" {
			t.Fatal("list reply is empty")
		}
		durations = append(durations, time.Since(start))
	}

	p95 := p95Millis(durations)
	if err := assertListResponsiveness(profile, p95); err != nil {
		t.Fatal(err)
	}
}
