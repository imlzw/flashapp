package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

func (s *server) asyncUpdateAppTitle(userID int, appID, prompt string) {
	// 给点时间让前端刷新
	time.Sleep(2 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	title, err := s.fetchLLMSummary(ctx, "请用 2-6 个字总结以下 H5 应用的需求意图，直接输出总结后的标题，不要标点符号和解释：\n"+prompt)
	if err != nil || title == "" {
		return
	}

	title = strings.Trim(title, " \"'“”自然‘’")
	if len([]rune(title)) > 12 {
		title = string([]rune(title)[:12])
	}

	_, _ = s.store.updateApp(userID, appID, func(app *appRecord) error {
		app.Title = title
		return nil
	})
}

func (s *server) fetchLLMSummary(ctx context.Context, prompt string) (string, error) {
	if s.cfg.UseMock {
		return "Mock 应用", nil
	}

	var urlStr string
	var payload map[string]any

	if s.cfg.LLMProvider == "gemini" {
		urlStr = s.cfg.LLMAPIURL
		if urlStr == "" || !strings.Contains(urlStr, "://") {
			model := s.cfg.LLMModel
			if model == "" {
				model = "gemini-1.5-flash"
			}
			urlStr = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", model)
		}

		if !strings.Contains(urlStr, "key=") {
			if strings.Contains(urlStr, "?") {
				urlStr += "&key=" + s.cfg.LLMAPIKey
			} else {
				urlStr += "?key=" + s.cfg.LLMAPIKey
			}
		}

		payload = map[string]any{
			"contents": []map[string]any{
				{"parts": []map[string]any{{"text": prompt}}},
			},
			"generationConfig": map[string]any{
				"maxOutputTokens": 20,
			},
		}
	} else {
		urlStr = s.cfg.LLMAPIURL
		payload = map[string]any{
			"model": s.cfg.LLMModel,
			"messages": []map[string]any{
				{"role": "user", "content": prompt},
			},
			"max_tokens": 20,
		}
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", urlStr, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if s.cfg.LLMProvider != "gemini" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.LLMAPIKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm error: %d", resp.StatusCode)
	}

	if s.cfg.LLMProvider == "gemini" {
		var res struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil || len(res.Candidates) == 0 {
			return "", err
		}
		return res.Candidates[0].Content.Parts[0].Text, nil
	} else {
		var res openAIResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil || len(res.Choices) == 0 {
			return "", err
		}
		return res.Choices[0].Message.Content, nil
	}
}

func (s *server) generateHTML(ctx context.Context, req agentRequest, writer *streamDeploymentWriter) error {
	if s.cfg.UseMock {
		return streamMockHTML(ctx, req, writer)
	}
	if s.cfg.LLMAPIKey == "" {
		return errors.New("missing FLASHAPP_LLM_API_KEY")
	}

	switch s.cfg.LLMProvider {
	case "gemini":
		return s.streamFromGemini(ctx, req, writer)
	default:
		return s.streamFromOpenAI(ctx, req, writer)
	}
}

func (s *server) streamFromOpenAI(ctx context.Context, req agentRequest, writer *streamDeploymentWriter) error {
	payload := map[string]any{
		"model":       s.cfg.LLMModel,
		"stream":      true,
		"max_tokens":  s.cfg.MaxTokens,
		"temperature": s.cfg.Temperature,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": s.cfg.SystemPrompt,
			},
			{
				"role":    "user",
				"content": composeAgentPrompt(req),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.LLMAPIURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+s.cfg.LLMAPIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return fmt.Errorf("llm upstream returned %s: %s", resp.Status, strings.TrimSpace(string(errBody)))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			return nil
		}

		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}

		text := chunk.Choices[0].Delta.Content
		if text == "" {
			continue
		}
		if err := writer.WriteChunk(text); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func (s *server) streamFromGemini(ctx context.Context, req agentRequest, writer *streamDeploymentWriter) error {
	payload := map[string]any{
		"contents": []map[string]any{
			{
				"role": "user",
				"parts": []map[string]any{
					{"text": composeAgentPrompt(req)},
				},
			},
		},
		"systemInstruction": map[string]any{
			"parts": []map[string]any{
				{"text": s.cfg.SystemPrompt},
			},
		},
		"generationConfig": map[string]any{
			"maxOutputTokens": s.cfg.MaxTokens,
			"temperature":     s.cfg.Temperature,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	apiURL := s.cfg.LLMAPIURL
	if apiURL == "" || !strings.Contains(apiURL, "://") {
		model := s.cfg.LLMModel
		if model == "" {
			model = "gemini-1.5-flash"
		}
		apiURL = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?key=%s", model, s.cfg.LLMAPIKey)
	} else if !strings.Contains(apiURL, "key=") {
		if strings.Contains(apiURL, "?") {
			apiURL += "&key=" + s.cfg.LLMAPIKey
		} else {
			apiURL += "?key=" + s.cfg.LLMAPIKey
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return fmt.Errorf("gemini upstream returned %s: %s", resp.Status, strings.TrimSpace(string(errBody)))
	}

	dec := json.NewDecoder(resp.Body)
	token, err := dec.Token()
	if err != nil {
		return fmt.Errorf("failed to read gemini response start: %v", err)
	}

	var (
		firstChunk = true
		hasPrefix  = false
	)

	if delim, ok := token.(json.Delim); ok && delim == '[' {
		for dec.More() {
			var chunk struct {
				Candidates []struct {
					Content struct {
						Parts []struct {
							Text string `json:"text"`
						} `json:"parts"`
					} `json:"content"`
					FinishReason string `json:"finishReason"`
				} `json:"candidates"`
			}
			if err := dec.Decode(&chunk); err != nil {
				continue
			}

			for _, cand := range chunk.Candidates {
				if cand.FinishReason != "" && cand.FinishReason != "STOP" {
					log.Printf("Gemini generation stopped for app %s: %s", req.AppID, cand.FinishReason)
				}
				for _, part := range cand.Content.Parts {
					text := part.Text
					if text == "" {
						continue
					}

					if firstChunk {
						text = strings.TrimLeft(text, " \n\r")
						if strings.HasPrefix(text, "```") {
							hasPrefix = true
							if idx := strings.Index(text, "\n"); idx != -1 {
								text = text[idx+1:]
							} else {
								text = ""
							}
						}
						firstChunk = false
					}

					if hasPrefix {
						if idx := strings.Index(text, "```"); idx != -1 {
							text = text[:idx]
							hasPrefix = false
						}
					}

					if text != "" {
						if err := writer.WriteChunk(text); err != nil {
							return err
						}
					}
				}
			}
		}
		_, _ = dec.Token()
	} else {
		return fmt.Errorf("unexpected gemini response format: expected array")
	}

	return nil
}

func (w *streamDeploymentWriter) WriteChunk(chunk string) error {
	if chunk == "" {
		return nil
	}
	if _, err := io.WriteString(w.file, chunk); err != nil {
		return err
	}
	if _, err := io.WriteString(w.response, chunk); err != nil {
		return err
	}
	w.started = true
	w.flusher.Flush()
	return nil
}

func composeAgentPrompt(req agentRequest) string {
	return composePromptWithTemplate(req)
}

func pickTitle(title, prompt, fallback string) string {
	title = strings.TrimSpace(title)
	if title != "" {
		return limitRunes(title, 24)
	}

	firstLine := strings.Split(prompt, "\n")[0]
	prefixes := []string{"帮我做一个", "帮我写个", "请帮我做一个", "我想做一个", "帮我生成一个", "创建一个", "生成一个", "做一个", "做个", "写个"}

	cleanTitle := firstLine
	for _, p := range prefixes {
		if strings.HasPrefix(cleanTitle, p) {
			cleanTitle = strings.TrimPrefix(cleanTitle, p)
			break
		}
	}

	cleanTitle = strings.TrimRight(cleanTitle, "。，？！.,?! ")

	if candidate := limitRunes(cleanTitle, 24); candidate != "" {
		return candidate
	}

	if candidate := limitRunes(strings.TrimSpace(fallback), 24); candidate != "" {
		return candidate
	}
	return "未命名应用"
}

func summarizePrompt(prompt string) string {
	cleaned := strings.Join(strings.Fields(strings.TrimSpace(prompt)), " ")
	if cleaned == "" {
		return "这是一个由 FlashApp 生成的轻量 H5 页面，围绕当前需求即时构造首屏、信息卡片与可演示的交互区域。"
	}
	if len([]rune(cleaned)) > 92 {
		cleaned = limitRunes(cleaned, 92) + "..."
	}
	return "围绕“" + cleaned + "”构建一个可立即部署和预览的 H5 页面，强调信息层级、移动端体验和快速验证。"
}

func promptFragments(prompt string, limit int) []string {
	replacer := strings.NewReplacer("\r", "\n", "，", "\n", ";", "\n", "。", "\n", ".", "\n", "！", "\n", ",", "\n", "？", "\n", "|", "\n")
	chunks := strings.Split(replacer.Replace(prompt), "\n")
	seen := make(map[string]struct{})
	result := make([]string, 0, limit)

	for _, item := range chunks {
		item = strings.Join(strings.Fields(strings.TrimSpace(item)), " ")
		if item == "" {
			continue
		}
		item = limitRunes(item, 24)
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
		if len(result) >= limit {
			break
		}
	}
	return result
}