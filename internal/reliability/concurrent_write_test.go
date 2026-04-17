// Package reliability hosts EP-022 cross-store reliability tests.
//
// TestConcurrentWrites_NoBusyErrors (REQ-22.013, AC-22.010) exercises both
// Local SQLite Stores (vector store on vec_summaries/vec_turns/vec_tools and
// the jobs store) under four concurrent writers with the PRAGMA policy
// produced by sqlitepragma.RecommendedPolicy. Must be run with -race.
package reliability

import (
	"context"
	"pa/internal/jobs"
	"pa/internal/sqlitepragma"
	vectorsqlite "pa/internal/vector/sqlite"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// iterations is the per-writer work budget. The test asserts that every writer
// completed the full budget (AC-22.010: no busy/locked under contention).
const iterations = 200

// Covers AC-22.010 (EP-022): with the PRAGMA policy applied, concurrent writers
// across vector tables and the jobs store complete their full iteration budget
// without SQLITE_BUSY / database is locked, and are not silently aborted by
// the deadline safety net.
func TestConcurrentWrites_NoBusyErrors(t *testing.T) {
	dir := t.TempDir()
	vecPath := filepath.Join(dir, "vec.sqlite")
	jobsPath := filepath.Join(dir, "jobs.sqlite")
	vecPolicy := sqlitepragma.RecommendedPolicy(false)
	jobsPolicy := sqlitepragma.RecommendedPolicy(true)

	const dim = 4

	summ, err := vectorsqlite.NewWithTable(vecPath, dim, vectorsqlite.TableSummaries, vecPolicy)
	if err != nil {
		t.Fatalf("open vec_summaries: %v", err)
	}
	defer func() { _ = summ.Close() }()

	turns, err := vectorsqlite.NewWithTable(vecPath, dim, vectorsqlite.TableTurns, vecPolicy)
	if err != nil {
		t.Fatalf("open vec_turns: %v", err)
	}
	defer func() { _ = turns.Close() }()

	tools, err := vectorsqlite.NewWithTable(vecPath, dim, vectorsqlite.TableTools, vecPolicy)
	if err != nil {
		t.Fatalf("open vec_tools: %v", err)
	}
	defer func() { _ = tools.Close() }()

	jobsStore, err := jobs.Open(jobsPath, jobsPolicy)
	if err != nil {
		t.Fatalf("open jobs: %v", err)
	}
	defer func() { _ = jobsStore.Close() }()

	// Generous deadline so writers reliably complete on slow CI; the test fails
	// if any writer does not reach its iteration budget.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	errCh := make(chan error, 4)
	var wg sync.WaitGroup
	wg.Add(4)

	counts := map[string]*int64{
		"summ": new(int64),
		"turn": new(int64),
		"tool": new(int64),
		"jobs": new(int64),
	}

	go runVectorWriter(ctx, &wg, errCh, "summ", summ, counts["summ"])
	go runVectorWriter(ctx, &wg, errCh, "turn", turns, counts["turn"])
	go runVectorWriter(ctx, &wg, errCh, "tool", tools, counts["tool"])
	go runJobsWriter(ctx, &wg, errCh, jobsStore, counts["jobs"])

	wg.Wait()
	close(errCh)

	for e := range errCh {
		if isBusyOrLocked(e) {
			t.Fatalf("SQLITE_BUSY / database is locked observed (AC-22.010): %v", e)
		}
		t.Fatalf("writer error: %v", e)
	}

	for label, c := range counts {
		got := atomic.LoadInt64(c)
		if got != iterations {
			t.Fatalf("writer %q completed %d/%d iterations (AC-22.010: every writer must complete its full budget)", label, got, iterations)
		}
		t.Logf("writer %q completed %d iterations", label, got)
	}
}

func runVectorWriter(ctx context.Context, wg *sync.WaitGroup, errCh chan<- error, label string, st *vectorsqlite.Store, done *int64) {
	defer wg.Done()
	embed := []float32{0.1, 0.2, 0.3, 0.4}
	for i := 0; i < iterations; i++ {
		if ctx.Err() != nil {
			errCh <- ctx.Err()
			return
		}
		id := label + "-" + strconv.Itoa(i)
		if err := st.Add(ctx, id, embed, "text-"+id); err != nil {
			errCh <- err
			return
		}
		atomic.AddInt64(done, 1)
	}
}

func runJobsWriter(ctx context.Context, wg *sync.WaitGroup, errCh chan<- error, st *jobs.Store, done *int64) {
	defer wg.Done()
	for i := 0; i < iterations; i++ {
		if ctx.Err() != nil {
			errCh <- ctx.Err()
			return
		}
		_, err := st.CreateJob(ctx, jobs.JobInput{
			Name:           "job-" + strconv.Itoa(i),
			ScheduleExpr:   "once:2099-01-01T00:00:00Z",
			TimeZone:       "UTC",
			Instruction:    "noop",
			DeliveryChatID: 1,
			Status:         "active",
			OverlapPolicy:  "skip",
			TimeoutPolicy:  "fail",
		})
		if err != nil {
			errCh <- err
			return
		}
		atomic.AddInt64(done, 1)
	}
}

// isBusyOrLocked returns true when err carries the SQLite busy / locked
// signature that the PRAGMA policy is intended to prevent.
func isBusyOrLocked(err error) bool {
	if err == nil {
		return false
	}
	m := err.Error()
	return strings.Contains(m, "SQLITE_BUSY") ||
		strings.Contains(m, "database is locked") ||
		strings.Contains(m, "database table is locked")
}
