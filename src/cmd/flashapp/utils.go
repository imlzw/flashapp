package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func saveBase64Image(data string, path string) error {
	if !strings.HasPrefix(data, "data:image/") {
		return errors.New("invalid image data")
	}
	parts := strings.Split(data, ",")
	if len(parts) != 2 {
		return errors.New("invalid base64 format")
	}
	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return err
	}
	return os.WriteFile(path, decoded, 0644)
}

func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseBoolEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func parseIntEnv(key string) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

func parseFloatEnv(key string) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstPositiveInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func firstPositiveFloat(values ...float64) float64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func looksLikePlaceholderKey(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return lower == "" || strings.Contains(lower, "your-real-key") || strings.Contains(lower, "xxxx")
}

func normalizeDataFilePath(envPath, filePath string) string {
	selected := firstNonEmpty(envPath, filePath, "./runtime/data.json")
	return filepath.Clean(selected)
}

func normalizeAppRoot(envPath, filePath string) string {
	if strings.TrimSpace(envPath) != "" {
		return filepath.Clean(envPath)
	}
	if runtime.GOOS == "windows" && strings.HasPrefix(filePath, "/") {
		return filepath.Clean("./apps")
	}
	return filepath.Clean(firstNonEmpty(filePath, "./apps"))
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func generateAppID() (string, error) {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func limitRunes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(value))
	if len(runes) == 0 {
		return ""
	}
	if len(runes) <= limit {
		return string(runes)
	}
	return strings.TrimSpace(string(runes[:limit]))
}

func atomicReplaceFile(tmpPath, finalPath string) error {
	if runtime.GOOS == "windows" {
		_ = os.Remove(finalPath)
	}
	return os.Rename(tmpPath, finalPath)
}

func loadExistingHTML(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	if len(content) > maxContextBytes {
		content = content[:maxContextBytes]
	}
	return string(content), nil
}

func (s *server) previewURL(appID string) string {
	if s.cfg.PreviewBaseURL == "" {
		return "/preview/" + appID + "/"
	}
	if strings.Contains(s.cfg.PreviewBaseURL, "%s") {
		return fmt.Sprintf(s.cfg.PreviewBaseURL, appID)
	}
	return strings.TrimRight(s.cfg.PreviewBaseURL, "/") + "/" + appID + "/"
}

func extractTitleFromHTML(document string) string {
	lower := strings.ToLower(document)
	start := strings.Index(lower, "<title>")
	end := strings.Index(lower, "</title>")
	if start == -1 || end == -1 || end <= start+7 {
		return ""
	}
	return strings.TrimSpace(document[start+7 : end])
}

func getAppDir(root string, id string) string {
	if len(id) < 9 { // safety fallback for very short ids
		return filepath.Join(root, id)
	}
	// split id into chunks of 3 characters
	var parts []string
	parts = append(parts, root)
	for i := 0; i < len(id); i += 3 {
		end := i + 3
		if end > len(id) {
			end = len(id)
		}
		parts = append(parts, id[i:end])
	}
	nestedPath := filepath.Join(parts...)

	_, errNested := os.Stat(nestedPath)
	if os.IsNotExist(errNested) {
		oldFlatPath := filepath.Join(root, id)
		if _, errOld := os.Stat(oldFlatPath); errOld == nil {
			// old path exists, nested doesn't, rename
			_ = os.MkdirAll(filepath.Dir(nestedPath), 0755)
			_ = os.Rename(oldFlatPath, nestedPath)
		}
	}
	return nestedPath
}
