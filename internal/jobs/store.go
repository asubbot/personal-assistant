package jobs

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"pa/internal/sqlitepragma"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var ErrNotFound = errors.New("jobs: not found")

const (
	StatusActive = "active"
	StatusPaused = "paused"
)

// Store manages scheduled jobs persistence in SQLite.
type Store struct {
	db *sql.DB
}

type Job struct {
	ID             string
	Name           string
	ScheduleExpr   string
	TimeZone       string
	Instruction    string
	DeliveryChatID int64
	Status         string
	OverlapPolicy  string
	TimeoutPolicy  string
	NextRunAt      *time.Time
	LastRunStatus  string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type JobInput struct {
	Name           string
	ScheduleExpr   string
	TimeZone       string
	Instruction    string
	DeliveryChatID int64
	Status         string
	OverlapPolicy  string
	TimeoutPolicy  string
}

type JobRun struct {
	RunID              string
	JobID              string
	TriggerType        string
	StartedAt          time.Time
	FinishedAt         *time.Time
	Outcome            string
	FailureReasonClass string
	CreatedAt          time.Time
}

type JobRunInput struct {
	JobID              string
	TriggerType        string
	StartedAt          time.Time
	FinishedAt         *time.Time
	Outcome            string
	FailureReasonClass string
}

type DeleteChallenge struct {
	Token             string
	JobID             string
	RequestedByUserID int64
	ExpiresAt         time.Time
	CreatedAt         time.Time
}

// Open opens the scheduled-jobs SQLite database and applies the PRAGMA policy
// on every new connection via the DSN; see EP-022.
//
// policy must have ForeignKeys=true for the jobs store (FK cascades are required).
func Open(path string, policy sqlitepragma.Policy) (*Store, error) {
	if !policy.ForeignKeys {
		return nil, fmt.Errorf("jobs: policy.ForeignKeys must be true (jobs store relies on FK cascades)")
	}
	dsn, err := sqlitepragma.BuildDSN(path, policy)
	if err != nil {
		return nil, fmt.Errorf("jobs: build dsn: %w", err)
	}
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("jobs: open db: %w", err)
	}
	if err := sqlitepragma.VerifyOnOpen(context.Background(), db, policy); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("jobs: verify pragma: %w", err)
	}
	st := &Store{db: db}
	if err := st.initSchema(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return st, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

func (s *Store) initSchema(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS jobs (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			schedule_expr TEXT NOT NULL,
			time_zone TEXT NOT NULL,
			instruction TEXT NOT NULL,
			delivery_chat_id INTEGER NOT NULL,
			status TEXT NOT NULL,
			overlap_policy TEXT NOT NULL,
			timeout_policy TEXT NOT NULL,
			next_run_at TEXT,
			last_run_status TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS job_runs (
			run_id TEXT PRIMARY KEY,
			job_id TEXT NOT NULL,
			trigger_type TEXT NOT NULL,
			started_at TEXT NOT NULL,
			finished_at TEXT,
			outcome TEXT NOT NULL,
			failure_reason_class TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_job_runs_job_id_started ON job_runs(job_id, started_at DESC)`,
		`CREATE TABLE IF NOT EXISTS delete_challenges (
			token TEXT PRIMARY KEY,
			job_id TEXT NOT NULL,
			requested_by_user_id INTEGER NOT NULL,
			expires_at TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
		)`,
		`INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES (1, ?)`,
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for i, q := range stmts {
		if i == len(stmts)-1 {
			if _, err := s.db.ExecContext(ctx, q, now); err != nil {
				return fmt.Errorf("jobs: init schema migrations: %w", err)
			}
			continue
		}
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("jobs: init schema: %w", err)
		}
	}
	return nil
}

func (s *Store) CreateJob(ctx context.Context, in JobInput) (Job, error) {
	id, err := newID("job")
	if err != nil {
		return Job{}, err
	}
	now := time.Now().UTC()
	status := in.Status
	if status == "" {
		status = StatusActive
	}
	created := now.Format(time.RFC3339Nano)
	_, err = s.db.ExecContext(ctx, `INSERT INTO jobs(
		id, name, schedule_expr, time_zone, instruction, delivery_chat_id, status,
		overlap_policy, timeout_policy, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, in.Name, in.ScheduleExpr, in.TimeZone, in.Instruction, in.DeliveryChatID, status,
		in.OverlapPolicy, in.TimeoutPolicy, created, created,
	)
	if err != nil {
		return Job{}, fmt.Errorf("jobs: insert job: %w", err)
	}
	return s.GetJob(ctx, id)
}

func (s *Store) GetJob(ctx context.Context, id string) (Job, error) {
	var (
		j                    Job
		nextRun, created, up string
	)
	err := s.db.QueryRowContext(ctx, `SELECT
		id, name, schedule_expr, time_zone, instruction, delivery_chat_id, status,
		overlap_policy, timeout_policy, COALESCE(next_run_at, ''), last_run_status, created_at, updated_at
		FROM jobs WHERE id = ?`, id).Scan(
		&j.ID, &j.Name, &j.ScheduleExpr, &j.TimeZone, &j.Instruction, &j.DeliveryChatID, &j.Status,
		&j.OverlapPolicy, &j.TimeoutPolicy, &nextRun, &j.LastRunStatus, &created, &up,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, ErrNotFound
	}
	if err != nil {
		return Job{}, fmt.Errorf("jobs: get job: %w", err)
	}
	j.CreatedAt, err = parseTS(created)
	if err != nil {
		return Job{}, err
	}
	j.UpdatedAt, err = parseTS(up)
	if err != nil {
		return Job{}, err
	}
	if nextRun != "" {
		t, perr := parseTS(nextRun)
		if perr != nil {
			return Job{}, perr
		}
		j.NextRunAt = &t
	}
	return j, nil
}

func (s *Store) ListJobs(ctx context.Context) ([]Job, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT
		id, name, schedule_expr, time_zone, instruction, delivery_chat_id, status,
		overlap_policy, timeout_policy, COALESCE(next_run_at, ''), last_run_status, created_at, updated_at
		FROM jobs ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("jobs: list jobs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Job
	for rows.Next() {
		var (
			j                    Job
			nextRun, created, up string
		)
		if err := rows.Scan(
			&j.ID, &j.Name, &j.ScheduleExpr, &j.TimeZone, &j.Instruction, &j.DeliveryChatID, &j.Status,
			&j.OverlapPolicy, &j.TimeoutPolicy, &nextRun, &j.LastRunStatus, &created, &up,
		); err != nil {
			return nil, fmt.Errorf("jobs: scan list jobs: %w", err)
		}
		j.CreatedAt, err = parseTS(created)
		if err != nil {
			return nil, err
		}
		j.UpdatedAt, err = parseTS(up)
		if err != nil {
			return nil, err
		}
		if nextRun != "" {
			t, perr := parseTS(nextRun)
			if perr != nil {
				return nil, perr
			}
			j.NextRunAt = &t
		}
		out = append(out, j)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("jobs: list rows: %w", err)
	}
	return out, nil
}

func (s *Store) SetJobStatus(ctx context.Context, id, status string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE jobs SET status = ?, updated_at = ? WHERE id = ?`,
		status, time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return fmt.Errorf("jobs: update status: %w", err)
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) SetJobNextRun(ctx context.Context, id string, nextRun *time.Time) error {
	next := ""
	if nextRun != nil {
		next = nextRun.UTC().Format(time.RFC3339Nano)
	}
	res, err := s.db.ExecContext(ctx, `UPDATE jobs SET next_run_at = ?, updated_at = ? WHERE id = ?`,
		next, time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return fmt.Errorf("jobs: update next_run_at: %w", err)
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) SetJobLastRunStatus(ctx context.Context, id, status string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE jobs SET last_run_status = ?, updated_at = ? WHERE id = ?`,
		status, time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return fmt.Errorf("jobs: update last_run_status: %w", err)
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteJob(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM jobs WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("jobs: delete job: %w", err)
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) RecordRun(ctx context.Context, in JobRunInput) (JobRun, error) {
	runID, err := newID("run")
	if err != nil {
		return JobRun{}, err
	}
	started := in.StartedAt.UTC()
	if started.IsZero() {
		started = time.Now().UTC()
	}
	created := time.Now().UTC()
	finished := ""
	if in.FinishedAt != nil {
		finished = in.FinishedAt.UTC().Format(time.RFC3339Nano)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO job_runs(
		run_id, job_id, trigger_type, started_at, finished_at, outcome, failure_reason_class, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		runID, in.JobID, in.TriggerType, started.Format(time.RFC3339Nano), finished, in.Outcome, in.FailureReasonClass, created.Format(time.RFC3339Nano),
	)
	if err != nil {
		return JobRun{}, fmt.Errorf("jobs: insert run: %w", err)
	}
	return JobRun{
		RunID:              runID,
		JobID:              in.JobID,
		TriggerType:        in.TriggerType,
		StartedAt:          started,
		FinishedAt:         in.FinishedAt,
		Outcome:            in.Outcome,
		FailureReasonClass: in.FailureReasonClass,
		CreatedAt:          created,
	}, nil
}

func (s *Store) GetLastRun(ctx context.Context, jobID string) (*JobRun, error) {
	var (
		r                              JobRun
		started, finished, createdText string
	)
	err := s.db.QueryRowContext(ctx, `SELECT
		run_id, job_id, trigger_type, started_at, COALESCE(finished_at, ''), outcome, failure_reason_class, created_at
		FROM job_runs WHERE job_id = ? ORDER BY started_at DESC LIMIT 1`, jobID).Scan(
		&r.RunID, &r.JobID, &r.TriggerType, &started, &finished, &r.Outcome, &r.FailureReasonClass, &createdText,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("jobs: get last run: %w", err)
	}
	r.StartedAt, err = parseTS(started)
	if err != nil {
		return nil, err
	}
	r.CreatedAt, err = parseTS(createdText)
	if err != nil {
		return nil, err
	}
	if finished != "" {
		t, perr := parseTS(finished)
		if perr != nil {
			return nil, perr
		}
		r.FinishedAt = &t
	}
	return &r, nil
}

func (s *Store) ListRuns(ctx context.Context, jobID string) ([]JobRun, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT
		run_id, job_id, trigger_type, started_at, COALESCE(finished_at, ''), outcome, failure_reason_class, created_at
		FROM job_runs WHERE job_id = ? ORDER BY started_at ASC`, jobID)
	if err != nil {
		return nil, fmt.Errorf("jobs: list runs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make([]JobRun, 0)
	for rows.Next() {
		var (
			r                              JobRun
			started, finished, createdText string
		)
		if err := rows.Scan(&r.RunID, &r.JobID, &r.TriggerType, &started, &finished, &r.Outcome, &r.FailureReasonClass, &createdText); err != nil {
			return nil, fmt.Errorf("jobs: scan list runs: %w", err)
		}
		r.StartedAt, err = parseTS(started)
		if err != nil {
			return nil, err
		}
		r.CreatedAt, err = parseTS(createdText)
		if err != nil {
			return nil, err
		}
		if finished != "" {
			t, perr := parseTS(finished)
			if perr != nil {
				return nil, perr
			}
			r.FinishedAt = &t
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("jobs: list runs rows: %w", err)
	}
	return out, nil
}

func (s *Store) CreateDeleteChallenge(ctx context.Context, jobID string, requestedBy int64, ttl time.Duration) (DeleteChallenge, error) {
	token, err := newID("del")
	if err != nil {
		return DeleteChallenge{}, err
	}
	now := time.Now().UTC()
	ch := DeleteChallenge{
		Token:             token,
		JobID:             jobID,
		RequestedByUserID: requestedBy,
		ExpiresAt:         now.Add(ttl),
		CreatedAt:         now,
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO delete_challenges(token, job_id, requested_by_user_id, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`,
		ch.Token, ch.JobID, ch.RequestedByUserID, ch.ExpiresAt.Format(time.RFC3339Nano), ch.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return DeleteChallenge{}, fmt.Errorf("jobs: create delete challenge: %w", err)
	}
	return ch, nil
}

func (s *Store) GetDeleteChallenge(ctx context.Context, token string) (DeleteChallenge, error) {
	var (
		ch                       DeleteChallenge
		expiresAtText, createdAt string
	)
	err := s.db.QueryRowContext(ctx, `SELECT token, job_id, requested_by_user_id, expires_at, created_at FROM delete_challenges WHERE token = ?`, token).Scan(
		&ch.Token, &ch.JobID, &ch.RequestedByUserID, &expiresAtText, &createdAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return DeleteChallenge{}, ErrNotFound
	}
	if err != nil {
		return DeleteChallenge{}, fmt.Errorf("jobs: get delete challenge: %w", err)
	}
	ch.ExpiresAt, err = parseTS(expiresAtText)
	if err != nil {
		return DeleteChallenge{}, err
	}
	ch.CreatedAt, err = parseTS(createdAt)
	if err != nil {
		return DeleteChallenge{}, err
	}
	return ch, nil
}

func (s *Store) DeleteDeleteChallenge(ctx context.Context, token string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM delete_challenges WHERE token = ?`, token)
	if err != nil {
		return fmt.Errorf("jobs: delete challenge: %w", err)
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return ErrNotFound
	}
	return nil
}

func newID(prefix string) (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("jobs: id generation: %w", err)
	}
	return fmt.Sprintf("%s_%d_%s", prefix, time.Now().UTC().UnixNano(), hex.EncodeToString(b[:])), nil
}

func parseTS(v string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, v)
	if err != nil {
		return time.Time{}, fmt.Errorf("jobs: parse timestamp %q: %w", v, err)
	}
	return t.UTC(), nil
}
