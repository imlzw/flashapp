package main

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type dbStore struct {
	db *sql.DB
}

func openDBStore(dsn string) (*dbStore, error) {
	if err := os.MkdirAll(filepath.Dir(dsn), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	if err := initSchema(db); err != nil {
		return nil, err
	}

	return &dbStore{db: db}, nil
}

func initSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		nickname TEXT NOT NULL,
		password_hash TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS apps (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		status INTEGER NOT NULL,
		prompt TEXT NOT NULL,
		preview_url TEXT NOT NULL,
		version INTEGER NOT NULL DEFAULT 0,
		is_public BOOLEAN NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS published_apps (
		id TEXT PRIMARY KEY,
		original_app_id TEXT NOT NULL,
		user_id INTEGER NOT NULL,
		author_name TEXT NOT NULL,
		title TEXT NOT NULL,
		prompt TEXT NOT NULL,
		preview_url TEXT NOT NULL,
		screenshot_url TEXT NOT NULL,
		version INTEGER NOT NULL,
		is_public BOOLEAN NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);
	`
	_, err := db.Exec(schema)
	return err
}

func (s *dbStore) createUser(username, passwordHash string) (user, error) {
	now := time.Now().UTC()
	res, err := s.db.Exec("INSERT INTO users (username, nickname, password_hash, created_at) VALUES (?, ?, ?, ?)", username, username, passwordHash, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return user{}, errAlreadyExists
		}
		return user{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return user{}, err
	}
	return user{
		ID:           int(id),
		Username:     username,
		Nickname:     username,
		PasswordHash: passwordHash,
		CreatedAt:    now,
	}, nil
}

func (s *dbStore) getUserByID(id int) (user, bool) {
	row := s.db.QueryRow("SELECT id, username, nickname, password_hash, created_at FROM users WHERE id = ?", id)
	var u user
	err := row.Scan(&u.ID, &u.Username, &u.Nickname, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return user{}, false
	}
	return u, true
}

func (s *dbStore) findUserByUsername(username string) (user, bool) {
	row := s.db.QueryRow("SELECT id, username, nickname, password_hash, created_at FROM users WHERE username = ?", username)
	var u user
	err := row.Scan(&u.ID, &u.Username, &u.Nickname, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return user{}, false
	}
	return u, true
}

func (s *dbStore) updateUserPassword(userID int, newHash string) error {
	res, err := s.db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", newHash, userID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return os.ErrNotExist
	}
	return nil
}

func (s *dbStore) updateUserNickname(userID int, newNickname string) error {
	res, err := s.db.Exec("UPDATE users SET nickname = ? WHERE id = ?", newNickname, userID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return os.ErrNotExist
	}
	return nil
}

func (s *dbStore) updateUserUsername(userID int, newUsername string) error {
	res, err := s.db.Exec("UPDATE users SET username = ? WHERE id = ?", newUsername, userID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return os.ErrNotExist
	}
	return nil
}

func (s *dbStore) saveNewApp(record appRecord) error {
	_, err := s.db.Exec(`INSERT INTO apps (id, user_id, title, status, prompt, preview_url, version, is_public, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.ID, record.UserID, record.Title, record.Status, record.Prompt, record.PreviewURL, record.Version, record.IsPublic, record.CreatedAt, record.UpdatedAt)
	return err
}

func (s *dbStore) getApp(userID int, appID string) (appRecord, bool) {
	row := s.db.QueryRow("SELECT id, user_id, title, status, prompt, preview_url, version, is_public, created_at, updated_at FROM apps WHERE user_id = ? AND id = ?", userID, appID)
	var a appRecord
	err := row.Scan(&a.ID, &a.UserID, &a.Title, &a.Status, &a.Prompt, &a.PreviewURL, &a.Version, &a.IsPublic, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return appRecord{}, false
	}
	return a, true
}

func (s *dbStore) listApps(userID int) []appRecord {
	rows, err := s.db.Query("SELECT id, user_id, title, status, prompt, preview_url, version, is_public, created_at, updated_at FROM apps WHERE user_id = ? ORDER BY updated_at DESC", userID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var apps []appRecord
	for rows.Next() {
		var a appRecord
		if err := rows.Scan(&a.ID, &a.UserID, &a.Title, &a.Status, &a.Prompt, &a.PreviewURL, &a.Version, &a.IsPublic, &a.CreatedAt, &a.UpdatedAt); err == nil {
			apps = append(apps, a)
		}
	}
	if apps == nil {
		apps = []appRecord{}
	}
	return apps
}

func (s *dbStore) updateApp(userID int, appID string, mutate func(*appRecord) error) (appRecord, error) {
	a, ok := s.getApp(userID, appID)
	if !ok {
		return appRecord{}, os.ErrNotExist
	}
	if err := mutate(&a); err != nil {
		return appRecord{}, err
	}
	a.UpdatedAt = time.Now().UTC()
	res, err := s.db.Exec(`UPDATE apps SET title = ?, status = ?, prompt = ?, preview_url = ?, version = ?, is_public = ?, updated_at = ? WHERE id = ? AND user_id = ?`,
		a.Title, a.Status, a.Prompt, a.PreviewURL, a.Version, a.IsPublic, a.UpdatedAt, a.ID, a.UserID)
	if err != nil {
		return appRecord{}, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return appRecord{}, os.ErrNotExist
	}
	return a, nil
}

func (s *dbStore) deleteApp(userID int, appID string) error {
	res, err := s.db.Exec("DELETE FROM apps WHERE id = ? AND user_id = ?", appID, userID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return os.ErrNotExist
	}
	return nil
}

func (s *dbStore) savePublishedApp(record publishedApp) error {
	_, err := s.db.Exec(`INSERT INTO published_apps (id, original_app_id, user_id, author_name, title, prompt, preview_url, screenshot_url, version, is_public, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET 
			author_name=excluded.author_name,
			title=excluded.title,
			prompt=excluded.prompt,
			preview_url=excluded.preview_url,
			screenshot_url=excluded.screenshot_url,
			version=excluded.version,
			is_public=excluded.is_public,
			updated_at=excluded.updated_at`,
		record.ID, record.OriginalAppID, record.UserID, record.AuthorName, record.Title, record.Prompt, record.PreviewURL, record.ScreenshotURL, record.Version, record.IsPublic, record.CreatedAt, record.UpdatedAt)
	return err
}

func (s *dbStore) getPublishedAppByOriginal(userID int, originalAppID string) (publishedApp, bool) {
	row := s.db.QueryRow("SELECT id, original_app_id, user_id, author_name, title, prompt, preview_url, screenshot_url, version, is_public, created_at, updated_at FROM published_apps WHERE user_id = ? AND original_app_id = ?", userID, originalAppID)
	return scanPublishedApp(row)
}

func (s *dbStore) getPublishedApp(id string) (publishedApp, bool) {
	row := s.db.QueryRow("SELECT id, original_app_id, user_id, author_name, title, prompt, preview_url, screenshot_url, version, is_public, created_at, updated_at FROM published_apps WHERE id = ?", id)
	return scanPublishedApp(row)
}

func scanPublishedApp(row *sql.Row) (publishedApp, bool) {
	var a publishedApp
	err := row.Scan(&a.ID, &a.OriginalAppID, &a.UserID, &a.AuthorName, &a.Title, &a.Prompt, &a.PreviewURL, &a.ScreenshotURL, &a.Version, &a.IsPublic, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return publishedApp{}, false
	}
	return a, true
}

func (s *dbStore) listPublishedApps(userID int) []publishedApp {
	rows, err := s.db.Query("SELECT id, original_app_id, user_id, author_name, title, prompt, preview_url, screenshot_url, version, is_public, created_at, updated_at FROM published_apps WHERE user_id = ? ORDER BY updated_at DESC", userID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanPublishedApps(rows)
}

func (s *dbStore) listAllPublishedApps() []publishedApp {
	rows, err := s.db.Query("SELECT id, original_app_id, user_id, author_name, title, prompt, preview_url, screenshot_url, version, is_public, created_at, updated_at FROM published_apps ORDER BY updated_at DESC")
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanPublishedApps(rows)
}

func scanPublishedApps(rows *sql.Rows) []publishedApp {
	var apps []publishedApp
	for rows.Next() {
		var a publishedApp
		if err := rows.Scan(&a.ID, &a.OriginalAppID, &a.UserID, &a.AuthorName, &a.Title, &a.Prompt, &a.PreviewURL, &a.ScreenshotURL, &a.Version, &a.IsPublic, &a.CreatedAt, &a.UpdatedAt); err == nil {
			apps = append(apps, a)
		}
	}
	if apps == nil {
		apps = []publishedApp{}
	}
	return apps
}

func (s *dbStore) deletePublishedApp(userID int, pubID string) error {
	res, err := s.db.Exec("DELETE FROM published_apps WHERE id = ? AND user_id = ?", pubID, userID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return errors.New("published app not found or access denied")
	}
	return nil
}
