package jobs

import (
	"context"
	"errors"
	"pa/internal/sqlitepragma"
	"path/filepath"
	"testing"
	"time"
)

func testPolicy() sqlitepragma.Policy { return sqlitepragma.RecommendedPolicy(true) }

func openTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "jobs.sqlite")
	st, err := Open(path, testPolicy())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func createBaselineJob(t *testing.T, st *Store) Job {
	t.Helper()
	j, err := st.CreateJob(context.Background(), JobInput{
		Name:           "morning-ai-digest",
		ScheduleExpr:   "0 9 * * *",
		TimeZone:       "UTC",
		Instruction:    "Collect AI digest",
		DeliveryChatID: 1001,
		Status:         StatusActive,
		OverlapPolicy:  "single_instance",
		TimeoutPolicy:  "cancel_after_limit",
	})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	return j
}

func updateBaselineJobState(t *testing.T, st *Store, jobID string, next time.Time) {
	t.Helper()
	ctx := context.Background()
	if err := st.SetJobNextRun(ctx, jobID, &next); err != nil {
		t.Fatalf("SetJobNextRun: %v", err)
	}
	if err := st.SetJobStatus(ctx, jobID, StatusPaused); err != nil {
		t.Fatalf("SetJobStatus: %v", err)
	}
	if err := st.SetJobLastRunStatus(ctx, jobID, "success"); err != nil {
		t.Fatalf("SetJobLastRunStatus: %v", err)
	}
}

func assertSinglePausedJob(t *testing.T, st *Store) {
	t.Helper()
	all, err := st.ListJobs(context.Background())
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("ListJobs len = %d, want 1", len(all))
	}
	if all[0].NextRunAt == nil {
		t.Fatal("ListJobs[0].NextRunAt is nil")
	}
	if all[0].Status != StatusPaused {
		t.Fatalf("ListJobs[0].Status = %q, want %q", all[0].Status, StatusPaused)
	}
	if all[0].LastRunStatus != "success" {
		t.Fatalf("ListJobs[0].LastRunStatus = %q", all[0].LastRunStatus)
	}
}

// Covers AC-19.001: CreateJob persists a stable unique job id.
// Covers AC-19.004: List/Get returns persisted job with next_run_at support.
// Covers AC-19.017: DeleteJob removes persisted job.
func TestStore_CreateGetListUpdateDelete(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	j := createBaselineJob(t, st)
	if j.ID == "" {
		t.Fatal("CreateJob: empty id")
	}

	got, err := st.GetJob(ctx, j.ID)
	if err != nil {
		t.Fatalf("GetJob: %v", err)
	}
	if got.Name != "morning-ai-digest" {
		t.Fatalf("GetJob.Name = %q", got.Name)
	}

	next := time.Now().UTC().Add(5 * time.Minute).Round(0)
	updateBaselineJobState(t, st, j.ID, next)
	assertSinglePausedJob(t, st)

	if err := st.DeleteJob(ctx, j.ID); err != nil {
		t.Fatalf("DeleteJob: %v", err)
	}
	if _, err := st.GetJob(ctx, j.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetJob after delete err = %v, want ErrNotFound", err)
	}
}

// Covers AC-19.002: store can load persisted jobs after reopen.
func TestStore_Reopen_PreservesJobs(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "jobs.sqlite")
	st, err := Open(path, testPolicy())
	if err != nil {
		t.Fatalf("Open(1): %v", err)
	}
	j, err := st.CreateJob(ctx, JobInput{
		Name:           "daily-summary",
		ScheduleExpr:   "0 8 * * *",
		TimeZone:       "UTC",
		Instruction:    "Send summary",
		DeliveryChatID: 1,
		Status:         StatusActive,
		OverlapPolicy:  "single_instance",
		TimeoutPolicy:  "cancel_after_limit",
	})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	_ = st.Close()

	st2, err := Open(path, testPolicy())
	if err != nil {
		t.Fatalf("Open(2): %v", err)
	}
	t.Cleanup(func() { _ = st2.Close() })
	all, err := st2.ListJobs(ctx)
	if err != nil {
		t.Fatalf("ListJobs(2): %v", err)
	}
	if len(all) != 1 || all[0].ID != j.ID {
		t.Fatalf("ListJobs(2) = %+v, want job id %q", all, j.ID)
	}
}

// Covers AC-19.004: RecordRun and GetLastRun persist deterministic latest run metadata.
func TestStore_RecordRun_GetLastRun(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	j, err := st.CreateJob(ctx, JobInput{
		Name:           "hourly-check",
		ScheduleExpr:   "0 * * * *",
		TimeZone:       "UTC",
		Instruction:    "Check status",
		DeliveryChatID: 42,
		Status:         StatusActive,
		OverlapPolicy:  "single_instance",
		TimeoutPolicy:  "cancel_after_limit",
	})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	start1 := time.Now().UTC().Add(-2 * time.Minute)
	finish1 := start1.Add(15 * time.Second)
	if _, err := st.RecordRun(ctx, JobRunInput{
		JobID:              j.ID,
		TriggerType:        "schedule",
		StartedAt:          start1,
		FinishedAt:         &finish1,
		Outcome:            "success",
		FailureReasonClass: "",
	}); err != nil {
		t.Fatalf("RecordRun(1): %v", err)
	}
	start2 := time.Now().UTC().Add(-1 * time.Minute)
	if _, err := st.RecordRun(ctx, JobRunInput{
		JobID:              j.ID,
		TriggerType:        "run_now",
		StartedAt:          start2,
		FinishedAt:         nil,
		Outcome:            "running",
		FailureReasonClass: "",
	}); err != nil {
		t.Fatalf("RecordRun(2): %v", err)
	}

	last, err := st.GetLastRun(ctx, j.ID)
	if err != nil {
		t.Fatalf("GetLastRun: %v", err)
	}
	if last == nil {
		t.Fatal("GetLastRun: nil")
	}
	if last.TriggerType != "run_now" {
		t.Fatalf("last.TriggerType = %q, want run_now", last.TriggerType)
	}
	if last.Outcome != "running" {
		t.Fatalf("last.Outcome = %q, want running", last.Outcome)
	}
}

// Covers AC-19.017: deleting a job removes dependent rows via foreign key cascade.
func TestStore_DeleteJob_CascadesRelatedRows(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	j, err := st.CreateJob(ctx, JobInput{
		Name:           "cascade-check",
		ScheduleExpr:   "*/5 * * * *",
		TimeZone:       "UTC",
		Instruction:    "run",
		DeliveryChatID: 9,
		Status:         StatusActive,
		OverlapPolicy:  "single_instance",
		TimeoutPolicy:  "cancel_after_limit",
	})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	if _, err := st.RecordRun(ctx, JobRunInput{
		JobID:       j.ID,
		TriggerType: "schedule",
		StartedAt:   time.Now().UTC(),
		Outcome:     "success",
	}); err != nil {
		t.Fatalf("RecordRun: %v", err)
	}
	if _, err := st.CreateDeleteChallenge(ctx, j.ID, 1, 5*time.Minute); err != nil {
		t.Fatalf("CreateDeleteChallenge: %v", err)
	}
	if err := st.DeleteJob(ctx, j.ID); err != nil {
		t.Fatalf("DeleteJob: %v", err)
	}
	last, err := st.GetLastRun(ctx, j.ID)
	if err != nil {
		t.Fatalf("GetLastRun after delete: %v", err)
	}
	if last != nil {
		t.Fatalf("GetLastRun after delete = %+v, want nil", last)
	}
}
