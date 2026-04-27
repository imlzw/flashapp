package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type fileStore struct {
	mu   sync.Mutex
	path string
	data storeData
}

type storeData struct {
	NextUserID    int            `json:"next_user_id"`
	Users         []user         `json:"users"`
	Apps          []appRecord    `json:"apps"`
	PublishedApps []publishedApp `json:"published_apps"`
}

func openFileStore(filePath string) (*fileStore, error) {
	store := &fileStore{
		path: filePath,
		data: storeData{
			NextUserID: 1,
		},
	}
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return nil, err
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := store.saveLocked(); err != nil {
				return nil, err
			}
			return store, nil
		}
		return nil, err
	}
	if len(bytes.TrimSpace(content)) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(content, &store.data); err != nil {
		return nil, err
	}
	if store.data.NextUserID == 0 {
		store.data.NextUserID = len(store.data.Users) + 1
	}
	return store, nil
}

var errAlreadyExists = errors.New("already exists")

func (s *fileStore) createUser(username, passwordHash string) (user, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, item := range s.data.Users {
		if item.Username == username {
			return user{}, errAlreadyExists
		}
	}

	now := time.Now().UTC()
	item := user{
		ID:           s.data.NextUserID,
		Username:     username,
		Nickname:     username,
		PasswordHash: passwordHash,
		CreatedAt:    now,
	}
	s.data.NextUserID++
	s.data.Users = append(s.data.Users, item)
	if err := s.saveLocked(); err != nil {
		return user{}, err
	}
	return item, nil
}

func (s *fileStore) getUserByID(id int) (user, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.data.Users {
		if item.ID == id {
			return item, true
		}
	}
	return user{}, false
}

func (s *fileStore) findUserByUsername(username string) (user, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.data.Users {
		if item.Username == username {
			return item, true
		}
	}
	return user{}, false
}

func (s *fileStore) updateUserPassword(userID int, newHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.data.Users {
		if s.data.Users[i].ID == userID {
			s.data.Users[i].PasswordHash = newHash
			return s.saveLocked()
		}
	}
	return os.ErrNotExist
}

func (s *fileStore) updateUserNickname(userID int, newNickname string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.data.Users {
		if s.data.Users[i].ID == userID {
			s.data.Users[i].Nickname = newNickname
			return s.saveLocked()
		}
	}
	return os.ErrNotExist
}

func (s *fileStore) updateUserUsername(userID int, newUsername string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.data.Users {
		if s.data.Users[i].ID == userID {
			s.data.Users[i].Username = newUsername
			return s.saveLocked()
		}
	}
	return os.ErrNotExist
}

func (s *fileStore) saveNewApp(record appRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Apps = append(s.data.Apps, record)
	return s.saveLocked()
}

func (s *fileStore) getApp(userID int, appID string) (appRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.data.Apps {
		if item.UserID == userID && item.ID == appID {
			return item, true
		}
	}
	return appRecord{}, false
}

func (s *fileStore) listApps(userID int) []appRecord {
	s.mu.Lock()
	defer s.mu.Unlock()

	apps := make([]appRecord, 0)
	for _, item := range s.data.Apps {
		if item.UserID == userID {
			apps = append(apps, item)
		}
	}
	sort.Slice(apps, func(i, j int) bool {
		return apps[i].UpdatedAt.After(apps[j].UpdatedAt)
	})
	return apps
}

func (s *fileStore) savePublishedApp(record publishedApp) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	found := false
	for i := range s.data.PublishedApps {
		if s.data.PublishedApps[i].ID == record.ID {
			s.data.PublishedApps[i] = record
			found = true
			break
		}
	}
	if !found {
		s.data.PublishedApps = append(s.data.PublishedApps, record)
	}
	return s.saveLocked()
}

func (s *fileStore) getPublishedAppByOriginal(userID int, originalAppID string) (publishedApp, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.data.PublishedApps {
		if item.UserID == userID && item.OriginalAppID == originalAppID {
			return item, true
		}
	}
	return publishedApp{}, false
}

func (s *fileStore) getPublishedApp(id string) (publishedApp, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.data.PublishedApps {
		if item.ID == id {
			return item, true
		}
	}
	return publishedApp{}, false
}

func (s *fileStore) listPublishedApps(userID int) []publishedApp {
	s.mu.Lock()
	defer s.mu.Unlock()
	res := make([]publishedApp, 0)
	for _, item := range s.data.PublishedApps {
		if item.UserID == userID {
			res = append(res, item)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].UpdatedAt.After(res[j].UpdatedAt)
	})
	return res
}

func (s *fileStore) listAllPublishedApps() []publishedApp {
	s.mu.Lock()
	defer s.mu.Unlock()
	res := make([]publishedApp, len(s.data.PublishedApps))
	copy(res, s.data.PublishedApps)
	sort.Slice(res, func(i, j int) bool {
		return res[i].UpdatedAt.After(res[j].UpdatedAt)
	})
	return res
}

func (s *fileStore) updateApp(userID int, appID string, mutate func(*appRecord) error) (appRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.data.Apps {
		if s.data.Apps[i].UserID != userID || s.data.Apps[i].ID != appID {
			continue
		}
		if err := mutate(&s.data.Apps[i]); err != nil {
			return appRecord{}, err
		}
		s.data.Apps[i].UpdatedAt = time.Now().UTC()
		if err := s.saveLocked(); err != nil {
			return appRecord{}, err
		}
		return s.data.Apps[i], nil
	}
	return appRecord{}, os.ErrNotExist
}

func (s *fileStore) deleteApp(userID int, appID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.data.Apps {
		if s.data.Apps[i].UserID == userID && s.data.Apps[i].ID == appID {
			s.data.Apps = append(s.data.Apps[:i], s.data.Apps[i+1:]...)
			return s.saveLocked()
		}
	}
	return os.ErrNotExist
}

func (s *fileStore) deletePublishedApp(userID int, pubID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	found := -1
	for i, app := range s.data.PublishedApps {
		if app.ID == pubID && app.UserID == userID {
			found = i
			break
		}
	}

	if found == -1 {
		return errors.New("published app not found or access denied")
	}

	s.data.PublishedApps = append(s.data.PublishedApps[:found], s.data.PublishedApps[found+1:]...)
	return s.saveLocked()
}

func (s *fileStore) saveLocked() error {
	payload, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return err
	}
	return atomicReplaceFile(tmpPath, s.path)
}
