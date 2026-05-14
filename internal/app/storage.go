package app

import (
	"database/sql"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const sessionSchema = `
CREATE TABLE IF NOT EXISTS sessions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  concurrency INTEGER NOT NULL,
  repeat_count INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS requests (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id INTEGER NOT NULL,
  sort_order INTEGER NOT NULL,
  name TEXT NOT NULL,
  method TEXT NOT NULL,
  url TEXT NOT NULL,
  headers TEXT NOT NULL,
  payload TEXT NOT NULL,
  FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);
`

func openSessionDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", filepath.Clean(sessionDBFile))
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(sessionSchema); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func (m model) saveSession(name string) error {
	db, err := openSessionDB()
	if err != nil {
		return err
	}
	defer func() {
		_ = db.Close()
	}()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	now := time.Now().UTC().Format(time.RFC3339)
	sessionID, err := upsertSession(tx, name, m.concurrency(), m.repeatCount(), now)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM requests WHERE session_id = ?`, sessionID); err != nil {
		return err
	}

	for i, form := range m.Forms {
		if _, err := tx.Exec(
			`INSERT INTO requests (session_id, sort_order, name, method, url, headers, payload)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			sessionID,
			i,
			form.NameInput.Value(),
			form.MethodInput.Value(),
			form.URLInput.Value(),
			form.HeadersArea.Value(),
			form.PayloadArea.Value(),
		); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func upsertSession(tx *sql.Tx, name string, concurrency int, repeatCount int, now string) (int64, error) {
	var id int64
	err := tx.QueryRow(`SELECT id FROM sessions WHERE name = ?`, name).Scan(&id)
	switch {
	case err == sql.ErrNoRows:
		result, execErr := tx.Exec(
			`INSERT INTO sessions (name, concurrency, repeat_count, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?)`,
			name,
			concurrency,
			repeatCount,
			now,
			now,
		)
		if execErr != nil {
			return 0, execErr
		}
		return result.LastInsertId()
	case err != nil:
		return 0, err
	default:
		_, execErr := tx.Exec(
			`UPDATE sessions SET concurrency = ?, repeat_count = ?, updated_at = ? WHERE id = ?`,
			concurrency,
			repeatCount,
			now,
			id,
		)
		if execErr != nil {
			return 0, execErr
		}
		return id, nil
	}
}

func listSavedSessions() ([]SavedSession, error) {
	db, err := openSessionDB()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = db.Close()
	}()

	rows, err := db.Query(`
		SELECT s.id, s.name, s.updated_at, COUNT(r.id)
		FROM sessions s
		LEFT JOIN requests r ON r.session_id = s.id
		GROUP BY s.id, s.name, s.updated_at
		ORDER BY s.updated_at DESC, s.id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	sessions := make([]SavedSession, 0)
	for rows.Next() {
		var session SavedSession
		if err := rows.Scan(&session.ID, &session.Name, &session.UpdatedAt, &session.RequestCount); err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

func loadSessionByID(sessionID int64, client *http.Client) (model, error) {
	db, err := openSessionDB()
	if err != nil {
		return model{}, err
	}
	defer func() {
		_ = db.Close()
	}()

	var (
		name        string
		concurrency int
		repeatCount int
	)
	if err := db.QueryRow(
		`SELECT name, concurrency, repeat_count FROM sessions WHERE id = ?`,
		sessionID,
	).Scan(&name, &concurrency, &repeatCount); err != nil {
		return model{}, err
	}

	rows, err := db.Query(
		`SELECT name, method, url, headers, payload
		 FROM requests
		 WHERE session_id = ?
		 ORDER BY sort_order ASC, id ASC`,
		sessionID,
	)
	if err != nil {
		return model{}, err
	}
	defer func() {
		_ = rows.Close()
	}()

	m := initialModel()
	m.Client = client
	m.Forms = nil
	m.Results = nil
	m.CurrentSession = name
	m.ConcurrencyInput.SetValue(strconv.Itoa(maxInt(concurrency, 1)))
	m.RepeatInput.SetValue(strconv.Itoa(maxInt(repeatCount, 1)))

	for rows.Next() {
		var req persistRequest
		if err := rows.Scan(&req.Name, &req.Method, &req.URL, &req.Headers, &req.Payload); err != nil {
			return model{}, err
		}
		form := newRequestForm()
		form.NameInput.SetValue(req.Name)
		form.MethodInput.SetValue(req.Method)
		form.URLInput.SetValue(req.URL)
		form.HeadersArea.SetValue(req.Headers)
		form.PayloadArea.SetValue(req.Payload)
		m.Forms = append(m.Forms, form)
	}

	if err := rows.Err(); err != nil {
		return model{}, err
	}

	if len(m.Forms) == 0 {
		m.Forms = []RequestForm{newRequestForm()}
	}

	m.FocusIndex = fieldConcurrency
	m.ActiveReq = 0
	m.SelectedResult = 0
	m.ModalMode = modalNone
	return m, nil
}

func (m model) defaultSessionName() string {
	if trimmed := strings.TrimSpace(m.CurrentSession); trimmed != "" {
		return trimmed
	}
	return fmt.Sprintf("session-%s", time.Now().Format("20060102-150405"))
}

func (m model) currentSessionLabel() string {
	if trimmed := strings.TrimSpace(m.CurrentSession); trimmed != "" {
		return trimmed
	}
	return "(unsaved)"
}

func maxInt(v int, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
}
