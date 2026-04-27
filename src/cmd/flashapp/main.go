package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	cfg := loadConfig()

	var store Store
	var err error
	if strings.HasSuffix(cfg.DataFile, ".db") || strings.HasSuffix(cfg.DataFile, ".sqlite") {
		store, err = openDBStore(cfg.DataFile)
	} else {
		store, err = openFileStore(cfg.DataFile)
	}
	if err != nil {
		log.Fatalf("failed to open store: %v", err)
	}

	s := &server{
		cfg:   cfg,
		store: store,
		httpClient: &http.Client{
			Timeout: 10 * time.Minute,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   30 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
	}

	if !cfg.UseMock && looksLikePlaceholderKey(cfg.LLMAPIKey) {
		log.Printf("WARNING: FLASHAPP_LLM_API_KEY is missing or looks like a placeholder. LLM features may fail.")
	}

	log.Printf("FlashApp starting on %s (Mock: %v, Provider: %s, Model: %s)", cfg.Addr, cfg.UseMock, cfg.LLMProvider, cfg.LLMModel)
	log.Fatal(http.ListenAndServe(cfg.Addr, s.routes()))
}

func loadConfig() config {
	envFile := firstNonEmpty(os.Getenv("FLASHAPP_CONFIG"), "config.json")
	fileCfg, err := loadFileConfig(envFile)
	if err != nil {
		log.Printf("Note: failed to load config file %s: %v. Using environment variables.", envFile, err)
	}

	llmProvider := strings.ToLower(firstNonEmpty(os.Getenv("FLASHAPP_LLM_PROVIDER"), fileCfg.Provider, "openai"))
	defaultAPIURL := "https://api.openai.com/v1/chat/completions"
	if llmProvider == "gemini" {
		defaultAPIURL = "" // Will be constructed in llm.go if empty
	}

	return config{
		Addr:           firstNonEmpty(os.Getenv("FLASHAPP_ADDR"), fileCfg.ServerPort, ":18080"),
		StaticDir:      firstNonEmpty(os.Getenv("FLASHAPP_STATIC"), "static"),
		DataFile:       normalizeDataFilePath(os.Getenv("FLASHAPP_DATA"), fileCfg.DBPath),
		AppRoot:        normalizeAppRoot(os.Getenv("FLASHAPP_APPS"), fileCfg.AppStoragePath),
		JWTSecret:      getenv("FLASHAPP_JWT_SECRET", "flash-secret-key-1234567890"),
		PreviewBaseURL: getenv("FLASHAPP_PREVIEW_BASE", ""),
		LLMAPIURL:      firstNonEmpty(os.Getenv("FLASHAPP_LLM_API_URL"), fileCfg.APIURL, defaultAPIURL),
		LLMAPIKey:      firstNonEmpty(os.Getenv("FLASHAPP_LLM_API_KEY"), fileCfg.APIKey),
		LLMModel:       firstNonEmpty(os.Getenv("FLASHAPP_LLM_MODEL"), fileCfg.Model, "gpt-4o"),
		LLMProvider:    llmProvider,
		SystemPrompt:   firstNonEmpty(os.Getenv("FLASHAPP_SYSTEM_PROMPT"), fileCfg.SystemPrompt, defaultSystemPrompt),
		MaxTokens:      firstPositiveInt(parseIntEnv("FLASHAPP_MAX_TOKENS"), fileCfg.MaxTokens, 4000),
		Temperature:    firstPositiveFloat(parseFloatEnv("FLASHAPP_TEMPERATURE"), fileCfg.Temperature, 0.7),
		UseMock:        parseBoolEnv("FLASHAPP_USE_MOCK", false),
	}
}

func loadFileConfig(filePath string) (fileConfig, error) {
	var cfg fileConfig
	f, err := os.Open(filePath)
	if err != nil {
		return cfg, err
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
