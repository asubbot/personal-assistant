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
	"strings"
	"sync"
	"testing"
	"time"
)

// Covers AC-22.010: with the PRAGMA policy applied, concurrent writers across
// vector tables and the jobs store do not return SQLITE_BUSY / database is locked.
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

	deadline := time.Now().Add(4 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	const iterations = 200

	errCh := make(chan error, 4)
	var wg sync.WaitGroup
	wg.Add(4)

	go runVectorWriter(ctx, &wg, errCh, "summ", summ, iterations)
	go runVectorWriter(ctx, &wg, errCh, "turn", turns, iterations)
	go runVectorWriter(ctx, &wg, errCh, "tool", tools, iterations)
	go runJobsWriter(ctx, &wg, errCh, jobsStore, iterations)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("concurrent writers did not complete within 5s budget")
	}

	close(errCh)
	for e := range errCh {
		if isBusyOrLocked(e) {
			t.Fatalf("SQLITE_BUSY / database is locked observed: %v", e)
		}
		t.Fatalf("writer error: %v", e)
	}
}

func runVectorWriter(ctx context.Context, wg *sync.WaitGroup, errCh chan<- error, label string, st *vectorsqlite.Store, iterations int) {
	defer wg.Done()
	embed := []float32{0.1, 0.2, 0.3, 0.4}
	for i := 0; i < iterations; i++ {
		if ctx.Err() != nil {
			return
		}
		id := label + "-" + itoa(i)
		if err := st.Add(ctx, id, embed, "text-"+id); err != nil {
			if ctx.Err() == nil {
				errCh <- err
			}
			return
		}
	}
}

func runJobsWriter(ctx context.Context, wg *sync.WaitGroup, errCh chan<- error, st *jobs.Store, iterations int) {
	defer wg.Done()
	for i := 0; i < iterations; i++ {
		if ctx.Err() != nil {
			return
		}
		_, err := st.CreateJob(ctx, jobs.JobInput{
			Name:           "job-" + itoa(i),
			ScheduleExpr:   "once:2099-01-01T00:00:00Z",
			TimeZone:       "UTC",
			Instruction:    "noop",
			DeliveryChatID: 1,
			Status:         "active",
			OverlapPolicy:  "skip",
			TimeoutPolicy:  "fail",
		})
		if err != nil {
			if ctx.Err() == nil {
				errCh <- err
			}
			return
		}
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

// itoa avoids strconv import noise for a test-only helper.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}
