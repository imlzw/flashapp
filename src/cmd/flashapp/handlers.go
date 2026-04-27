package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/auth", s.handleAuth)
	mux.HandleFunc("/api/user/password", s.withAuth(s.handleChangePassword))
	mux.HandleFunc("/api/user/nickname", s.withAuth(s.handleChangeNickname))
	mux.HandleFunc("/api/user/username", s.withAuth(s.handleChangeUsername))
	mux.HandleFunc("/api/apps", s.withAuth(s.handleListApps))
	mux.HandleFunc("/api/apps/toggle_public", s.withAuth(s.handleTogglePublic))
	mux.HandleFunc("/api/apps/delete_published", s.withAuth(s.handleDeletePublished))
	mux.HandleFunc("/api/create", s.withAuth(s.handleCreate))
	mux.HandleFunc("/api/update", s.withAuth(s.handleUpdate))
	mux.HandleFunc("/api/delete", s.withAuth(s.handleDelete))
	mux.HandleFunc("/api/publish", s.withAuth(s.handlePublish))
	mux.HandleFunc("/api/my_published_apps", s.withAuth(s.handleMyPublishedApps))
	mux.HandleFunc("/api/plaza", s.handlePlaza)
	mux.HandleFunc("/api/fork", s.withAuth(s.handleFork))
	mux.HandleFunc("/preview/", s.handlePreview)
	mux.Handle("/", http.FileServer(http.Dir(s.cfg.StaticDir)))
	return logRequest(mux)
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"mock_agent": s.cfg.UseMock,
		"llm_model":  s.cfg.LLMModel,
	})
}

func (s *server) handleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req authRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	username, err := normalizeUsername(req.Username)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(req.Password) < 6 {
		writeAPIError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	var currentUser user
	switch strings.ToLower(strings.TrimSpace(req.Mode)) {
	case "register":
		hash, err := hashPassword(req.Password)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to hash password")
			return
		}
		currentUser, err = s.store.createUser(username, hash)
		if err != nil {
			if errors.Is(err, errAlreadyExists) {
				writeAPIError(w, http.StatusConflict, "username already exists")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "failed to create user")
			return
		}
	case "login":
		var ok bool
		currentUser, ok = s.store.findUserByUsername(username)
		if !ok || !verifyPassword(currentUser.PasswordHash, req.Password) {
			writeAPIError(w, http.StatusUnauthorized, "invalid username or password")
			return
		}
	default:
		writeAPIError(w, http.StatusBadRequest, "mode must be register or login")
		return
	}

	token, err := signToken(s.cfg.JWTSecret, tokenClaims{
		Sub:      currentUser.ID,
		Username: currentUser.Username,
		Exp:      time.Now().Add(12 * time.Hour).Unix(),
	})
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	writeJSON(w, http.StatusOK, authResponse{
		Token: token,
		User:  toUserView(currentUser),
	})
}

func (s *server) handleListApps(w http.ResponseWriter, r *http.Request, currentUser *user) {
	if r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	apps := s.store.listApps(currentUser.ID)
	// Augment apps with public status from publishedApps
	type augmentedApp struct {
		appRecord
		IsPublished     bool   `json:"is_published"`
		IsPublic        bool   `json:"is_public"`
		PublishedTitle  string `json:"published_title,omitempty"`
		PublishedPrompt string `json:"published_prompt,omitempty"`
	}
	res := make([]augmentedApp, 0, len(apps))
	for _, app := range apps {
		aug := augmentedApp{appRecord: app}
		if pub, exists := s.store.getPublishedAppByOriginal(currentUser.ID, app.ID); exists {
			aug.IsPublished = true
			aug.IsPublic = pub.IsPublic
			aug.PublishedTitle = pub.Title
			aug.PublishedPrompt = pub.Prompt
		}
		res = append(res, aug)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"apps": res,
	})
}

func (s *server) handleDeletePublished(w http.ResponseWriter, r *http.Request, currentUser *user) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		AppID string `json:"app_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	// For deletion from plaza, we use the original App ID to find and remove the record
	pubApp, exists := s.store.getPublishedAppByOriginal(currentUser.ID, req.AppID)
	if !exists {
		// Try treating req.AppID as the published ID directly (for cases where it comes from plaza grid)
		pubApp, exists = s.store.getPublishedApp(req.AppID)
		if !exists || pubApp.UserID != currentUser.ID {
			writeAPIError(w, http.StatusNotFound, "published app not found")
			return
		}
	}

	if err := s.store.deletePublishedApp(pubApp.UserID, pubApp.ID); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to delete publication")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *server) handleTogglePublic(w http.ResponseWriter, r *http.Request, currentUser *user) {
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		AppID string `json:"app_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	pubApp, exists := s.store.getPublishedAppByOriginal(currentUser.ID, req.AppID)
	if !exists {
		writeAPIError(w, http.StatusNotFound, "请先发布该应用后再设置公开状态")
		return
	}

	pubApp.IsPublic = !pubApp.IsPublic
	pubApp.UpdatedAt = time.Now().UTC()
	if err := s.store.savePublishedApp(pubApp); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to update public status")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "is_public": pubApp.IsPublic})
}

func (s *server) handleCreate(w http.ResponseWriter, r *http.Request, currentUser *user) {
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req generationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		writeAPIError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	appID, err := generateAppID()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to generate app id")
		return
	}

	title := pickTitle(req.Title, prompt, "")
	description := req.Description
	if description == "" {
		description = prompt
	}
	now := time.Now().UTC()
	record := appRecord{
		ID:          appID,
		UserID:      currentUser.ID,
		Title:       title,
		Description: description,
		Status:      appStatusGenerating,
		Prompt:      prompt,
		PreviewURL:  s.previewURL(appID),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.saveNewApp(record); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to create app")
		return
	}

	s.streamAndDeploy(w, r, currentUser, record, prompt, "")
}

func (s *server) handleUpdate(w http.ResponseWriter, r *http.Request, currentUser *user) {
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req generationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.AppID) == "" {
		writeAPIError(w, http.StatusBadRequest, "app_id is required")
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		writeAPIError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	record, ok := s.store.getApp(currentUser.ID, req.AppID)
	if !ok {
		writeAPIError(w, http.StatusNotFound, "app not found")
		return
	}

	if strings.TrimSpace(req.Title) != "" || record.Title == "" || record.Title == "未命名应用" {
		record.Title = pickTitle(req.Title, req.Prompt, record.Title)
	}
	if record.Description == "" {
		if req.Description != "" {
			record.Description = req.Description
		} else {
			record.Description = strings.TrimSpace(req.Prompt)
		}
	}
	record.Status = appStatusGenerating
	record.Prompt = strings.TrimSpace(req.Prompt)
	record.UpdatedAt = time.Now().UTC()
	if _, err := s.store.updateApp(record.UserID, record.ID, func(app *appRecord) error {
		app.Title = record.Title
		app.Description = record.Description
		app.Status = appStatusGenerating
		app.Prompt = record.Prompt
		return nil
	}); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to update app state")
		return
	}

	existingHTML, err := loadExistingHTML(filepath.Join(getAppDir(s.cfg.AppRoot, record.ID), "index.html"))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to load existing app")
		return
	}

	s.streamAndDeploy(w, r, currentUser, record, record.Prompt, existingHTML)
}

func (s *server) handleDelete(w http.ResponseWriter, r *http.Request, currentUser *user) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		AppID string `json:"app_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.AppID) == "" {
		writeAPIError(w, http.StatusBadRequest, "app_id is required")
		return
	}

	record, ok := s.store.getApp(currentUser.ID, req.AppID)
	if !ok {
		writeAPIError(w, http.StatusNotFound, "app not found")
		return
	}

	if err := s.store.deleteApp(currentUser.ID, record.ID); err != nil {
		log.Printf("failed to delete app %s from store: %v", record.ID, err)
		writeAPIError(w, http.StatusInternalServerError, "failed to delete app")
		return
	}

	appDir := getAppDir(s.cfg.AppRoot, record.ID)
	if err := os.RemoveAll(appDir); err != nil {
		log.Printf("failed to delete app directory %s: %v", appDir, err)
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *server) handlePublish(w http.ResponseWriter, r *http.Request, currentUser *user) {
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		AppID       string `json:"app_id"`
		Screenshot  string `json:"screenshot"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Prompt      string `json:"prompt"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	record, ok := s.store.getApp(currentUser.ID, req.AppID)
	if !ok {
		writeAPIError(w, http.StatusNotFound, "app not found")
		return
	}

	// Use provided title/description/prompt or fallback to original record
	title := req.Title
	if title == "" {
		title = record.Title
	}
	description := req.Description
	if description == "" {
		description = record.Description
	}
	prompt := req.Prompt
	if prompt == "" {
		prompt = record.Prompt
	}

	// Update original record as well so the sidebar/history reflects the changes
	if title != record.Title || description != record.Description || prompt != record.Prompt {
		if _, err := s.store.updateApp(currentUser.ID, record.ID, func(app *appRecord) error {
			app.Title = title
			app.Description = description
			app.Prompt = prompt
			return nil
		}); err != nil {
			log.Printf("failed to sync title/description/prompt to original app %s: %v", record.ID, err)
		}
	}

	pubApp, exists := s.store.getPublishedAppByOriginal(currentUser.ID, req.AppID)
	if exists {
		pubApp.Version++
		pubApp.UpdatedAt = time.Now().UTC()
		pubApp.AuthorName = currentUser.Username
		pubApp.Title = title
		pubApp.Description = description
		pubApp.Prompt = prompt
		// Keep existing IsPublic status when updating
	} else {
		pubID, _ := generateAppID()
		pubApp = publishedApp{
			ID:            pubID,
			OriginalAppID: req.AppID,
			UserID:        currentUser.ID,
			AuthorName:    currentUser.Username,
			Title:         title,
			Description:   description,
			Prompt:        prompt,
			Version:       1,
			IsPublic:      false,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		}
	}

	pubDir := getAppDir(s.cfg.AppRoot, pubApp.ID)
	if err := os.MkdirAll(pubDir, 0755); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to create publish directory")
		return
	}

	sessionDir := getAppDir(s.cfg.AppRoot, req.AppID)
	srcHTML := filepath.Join(sessionDir, "index.html")
	dstHTML := filepath.Join(pubDir, "index.html")

	input, err := os.ReadFile(srcHTML)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to read app file")
		return
	}
	if err := os.WriteFile(dstHTML, input, 0644); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to copy app file")
		return
	}

	if req.Screenshot != "" {
		screenshotPath := filepath.Join(pubDir, "screenshot.png")
		if err := saveBase64Image(req.Screenshot, screenshotPath); err != nil {
			log.Printf("failed to save screenshot for %s: %v", pubApp.ID, err)
		} else {
			pubApp.ScreenshotURL = "/preview/" + pubApp.ID + "/screenshot.png"
		}
	}

	pubApp.PreviewURL = "/preview/" + pubApp.ID + "/index.html"

	if err := s.store.savePublishedApp(pubApp); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to save published app")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"pub_id": pubApp.ID, "version": pubApp.Version})
}

func (s *server) handlePlaza(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	all := s.store.listAllPublishedApps()
	public := make([]publishedApp, 0)
	for _, app := range all {
		if app.IsPublic {
			public = append(public, app)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"apps": public,
	})
}

func (s *server) handleMyPublishedApps(w http.ResponseWriter, r *http.Request, currentUser *user) {
	if r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"apps": s.store.listPublishedApps(currentUser.ID),
	})
}

func (s *server) handleFork(w http.ResponseWriter, r *http.Request, currentUser *user) {
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		PubID string `json:"pub_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	sourceApp, ok := s.store.getPublishedApp(req.PubID)
	if !ok {
		writeAPIError(w, http.StatusNotFound, "published app not found")
		return
	}

	newAppID, err := generateAppID()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to generate app id")
		return
	}

	sourceDir := getAppDir(s.cfg.AppRoot, sourceApp.ID)
	newDir := getAppDir(s.cfg.AppRoot, newAppID)

	if err := copyDir(sourceDir, newDir); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to copy app files")
		return
	}

	now := time.Now().UTC()
	newRecord := appRecord{
		ID:         newAppID,
		UserID:     currentUser.ID,
		Title:      sourceApp.Title + " (Fork)",
		Status:     appStatusReady,
		Prompt:     sourceApp.Prompt,
		PreviewURL: s.previewURL(newAppID),
		Version:    0,
		IsPublic:   false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.store.saveNewApp(newRecord); err != nil {
		_ = os.RemoveAll(newDir)
		writeAPIError(w, http.StatusInternalServerError, "failed to save forked app")
		return
	}

	writeJSON(w, http.StatusOK, newRecord)
}

func (s *server) handlePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	raw := strings.TrimPrefix(r.URL.Path, "/preview/")
	if raw == "" {
		http.NotFound(w, r)
		return
	}

	parts := strings.Split(raw, "/")
	appID := parts[0]
	if !validPathSegment(appID) {
		http.NotFound(w, r)
		return
	}
	if len(parts) == 1 && !strings.HasSuffix(r.URL.Path, "/") {
		http.Redirect(w, r, "/preview/"+appID+"/", http.StatusTemporaryRedirect)
		return
	}

	filePath := path.Join(parts[1:]...)
	if filePath == "" || filePath == "." {
		filePath = "index.html"
	}

	baseDir := getAppDir(s.cfg.AppRoot, appID)
	localPath := filepath.Join(baseDir, filepath.FromSlash(filePath))
	rel, err := filepath.Rel(baseDir, localPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		http.NotFound(w, r)
		return
	}

	info, err := os.Stat(localPath)
	if err == nil && info.IsDir() {
		localPath = filepath.Join(localPath, "index.html")
	}

	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	http.ServeFile(w, r, localPath)
}

func (s *server) withAuth(next authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(header, "Bearer ") {
			writeAPIError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}

		claims, err := parseToken(s.cfg.JWTSecret, strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
		if err != nil {
			writeAPIError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		if claims.Exp <= time.Now().Unix() {
			writeAPIError(w, http.StatusUnauthorized, "token expired")
			return
		}

		currentUser, ok := s.store.getUserByID(claims.Sub)
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "user not found")
			return
		}

		next(w, r, &currentUser)
	}
}

func (s *server) streamAndDeploy(w http.ResponseWriter, r *http.Request, currentUser *user, record appRecord, prompt, existingHTML string) {
	if r.Context().Err() != nil {
		return
	}

	appDir := getAppDir(s.cfg.AppRoot, record.ID)
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to prepare app directory")
		return
	}

	tmpPath := filepath.Join(appDir, "index.html.tmp")
	finalPath := filepath.Join(appDir, "index.html")

	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to create temporary file")
		return
	}
	defer func() {
		_ = tmpFile.Close()
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeAPIError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("X-App-ID", record.ID)
	w.Header().Set("X-Preview-URL", record.PreviewURL)

	writer := &streamDeploymentWriter{
		response: w,
		flusher:  flusher,
		file:     tmpFile,
	}

	err = s.generateHTML(r.Context(), agentRequest{
		AppID:        record.ID,
		Title:        record.Title,
		Description:  record.Description,
		Prompt:       prompt,
		ExistingHTML: existingHTML,
	}, writer)
	if err != nil {
		_ = os.Remove(tmpPath)
		if _, saveErr := s.store.updateApp(currentUser.ID, record.ID, func(app *appRecord) error {
			app.Status = appStatusFailed
			return nil
		}); saveErr != nil {
			log.Printf("mark app failed %s: %v", record.ID, saveErr)
		}
		if !writer.started {
			writeAPIError(w, http.StatusBadGateway, "generation failed: "+err.Error())
		}
		log.Printf("generate app %s: %v", record.ID, err)
		return
	}

	if err := tmpFile.Sync(); err != nil {
		_ = os.Remove(tmpPath)
		s.markAppFailed(currentUser.ID, record.ID)
		s.handleStreamFailure(w, writer, http.StatusInternalServerError, "failed to sync generated file", err)
		return
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		s.markAppFailed(currentUser.ID, record.ID)
		s.handleStreamFailure(w, writer, http.StatusInternalServerError, "failed to finalize generated file", err)
		return
	}
	if err := atomicReplaceFile(tmpPath, finalPath); err != nil {
		s.markAppFailed(currentUser.ID, record.ID)
		s.handleStreamFailure(w, writer, http.StatusInternalServerError, "failed to publish generated app", err)
		return
	}

	oldTitle := record.Title
	if _, err := s.store.updateApp(currentUser.ID, record.ID, func(app *appRecord) error {
		app.Title = record.Title
		app.Status = appStatusReady
		app.Prompt = prompt
		app.PreviewURL = record.PreviewURL
		return nil
	}); err != nil {
		log.Printf("mark app ready %s: %v", record.ID, err)
	} else {
		if oldTitle == "" || oldTitle == "未命名应用" || len([]rune(oldTitle)) > 15 {
			go s.asyncUpdateAppTitle(currentUser.ID, record.ID, prompt)
		}
	}
}

func (s *server) markAppFailed(userID int, appID string) {
	if _, err := s.store.updateApp(userID, appID, func(app *appRecord) error {
		app.Status = appStatusFailed
		return nil
	}); err != nil {
		log.Printf("mark app failed %s: %v", appID, err)
	}
}

func (s *server) handleStreamFailure(w http.ResponseWriter, writer *streamDeploymentWriter, status int, message string, err error) {
	if writer != nil && writer.started {
		log.Printf("%s: %v", message, err)
		return
	}
	writeAPIError(w, status, message)
}

func (s *server) handleChangePassword(w http.ResponseWriter, r *http.Request, currentUser *user) {
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !verifyPassword(currentUser.PasswordHash, req.OldPassword) {
		writeAPIError(w, http.StatusBadRequest, "当前密码错误")
		return
	}
	if len(req.NewPassword) < 6 {
		writeAPIError(w, http.StatusBadRequest, "新密码不能少于6位")
		return
	}
	newHash, err := hashPassword(req.NewPassword)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "密码加密失败")
		return
	}
	if err := s.store.updateUserPassword(currentUser.ID, newHash); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "更新密码失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *server) handleChangeNickname(w http.ResponseWriter, r *http.Request, currentUser *user) {
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		NewNickname string `json:"new_nickname"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	nickname, err := normalizeNickname(req.NewNickname)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.store.updateUserNickname(currentUser.ID, nickname); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "更新昵称失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *server) handleChangeUsername(w http.ResponseWriter, r *http.Request, currentUser *user) {
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		NewUsername string `json:"new_username"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	username, err := normalizeUsername(req.NewUsername)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	if existing, ok := s.store.findUserByUsername(username); ok && existing.ID != currentUser.ID {
		writeAPIError(w, http.StatusConflict, "该账号已被使用")
		return
	}

	if err := s.store.updateUserUsername(currentUser.ID, username); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "更新账号失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
