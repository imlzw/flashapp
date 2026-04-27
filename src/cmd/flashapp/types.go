package main

import (
	"net/http"
	"os"
	"time"
)

const (
	appStatusDraft = iota
	appStatusGenerating
	appStatusReady
	appStatusFailed

	defaultMemoryLimit  = 80 << 20
	maxContextBytes     = 1 << 20
	defaultSystemPrompt = `You are FlashApp, a high-end H5 engine.
Goal: Build H5 apps that are indistinguishable from native mobile applications (iOS/Android style).
Rules:
- Output ONLY a self-contained index.html (no markdown).
- Interaction: Must feel like a premium native app—fluid, responsive, and tactile.
- Games: Must provide a professional mobile game experience—full-screen, immersive, and zero-latency.
- Minimalism: Minimize text UI. Use icons instead of labels where possible. Maximize the main content/game area.
- Language: All user-facing text and code comments within the generated HTML MUST be in Chinese.
- Technical: Use a 100vh viewport-bounded layout with internal scrolling.
- Updates: Boldly refactor any subpar design to meet these native standards.`
)

type config struct {
	Addr           string
	StaticDir      string
	DataFile       string
	AppRoot        string
	JWTSecret      string
	PreviewBaseURL string
	LLMAPIURL      string
	LLMAPIKey      string
	LLMModel       string
	LLMProvider    string
	SystemPrompt   string
	MaxTokens      int
	Temperature    float64
	UseMock        bool
}

type fileConfig struct {
	APIKey         string  `json:"api_key"`
	APIURL         string  `json:"api_url"`
	Provider       string  `json:"provider"`
	Model          string  `json:"model"`
	SystemPrompt   string  `json:"system_prompt"`
	MaxTokens      int     `json:"max_tokens"`
	Temperature    float64 `json:"temperature"`
	AppStoragePath string  `json:"app_storage_path"`
	DBPath         string  `json:"db_path"`
	ServerPort     string  `json:"server_port"`
}

type server struct {
	cfg        config
	store      Store
	httpClient *http.Client
}

type Store interface {
	createUser(username, passwordHash string) (user, error)
	getUserByID(id int) (user, bool)
	findUserByUsername(username string) (user, bool)
	updateUserPassword(userID int, newHash string) error
	updateUserNickname(userID int, newNickname string) error
	updateUserUsername(userID int, newUsername string) error

	saveNewApp(record appRecord) error
	getApp(userID int, appID string) (appRecord, bool)
	listApps(userID int) []appRecord
	updateApp(userID int, appID string, mutate func(*appRecord) error) (appRecord, error)
	deleteApp(userID int, appID string) error

	savePublishedApp(record publishedApp) error
	getPublishedAppByOriginal(userID int, originalAppID string) (publishedApp, bool)
	getPublishedApp(id string) (publishedApp, bool)
	listPublishedApps(userID int) []publishedApp
	listAllPublishedApps() []publishedApp
	deletePublishedApp(userID int, pubID string) error
}

type user struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Nickname     string    `json:"nickname"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
}

type appRecord struct {
	ID          string    `json:"id"`
	UserID      int       `json:"user_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      int       `json:"status"`
	Prompt      string    `json:"prompt"`
	PreviewURL  string    `json:"preview_url"`
	Version     int       `json:"version"`
	IsPublic    bool      `json:"is_public"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type publishedApp struct {
	ID            string    `json:"id"`
	OriginalAppID string    `json:"original_app_id"`
	UserID        int       `json:"user_id"`
	AuthorName    string    `json:"author_name"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Prompt        string    `json:"prompt"`
	PreviewURL    string    `json:"preview_url"`
	ScreenshotURL string    `json:"screenshot_url"`
	Version       int       `json:"version"`
	IsPublic      bool      `json:"is_public"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type userView struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
}

type authRequest struct {
	Mode     string `json:"mode"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string   `json:"token"`
	User  userView `json:"user"`
}

type generationRequest struct {
	AppID       string `json:"app_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
}

type tokenClaims struct {
	Sub      int    `json:"sub"`
	Username string `json:"username"`
	Exp      int64  `json:"exp"`
}

type authedHandler func(http.ResponseWriter, *http.Request, *user)

type agentRequest struct {
	AppID        string
	Title        string
	Description  string
	Prompt       string
	ExistingHTML string
}

type streamDeploymentWriter struct {
	response http.ResponseWriter
	flusher  http.Flusher
	file     *os.File
	started  bool
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

type geminiStreamChunk struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}
