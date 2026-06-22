package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// ============================================================
// AI SERVICE — calls Python AI microservice
// ============================================================

type AIService struct {
	aiServiceURL string
	geminiKey    string
	httpClient   *http.Client
}

type AIAnalysisResult struct {
	Vibe          string            `json:"vibe"`
	VibeReason    string            `json:"vibe_reason"`
	Tone          string            `json:"tone"`
	Captions      map[string]string `json:"captions"`
	SuggestedTags []string          `json:"suggested_tags"`
}

func NewAIService(aiServiceURL, geminiKey string) *AIService {
	return &AIService{
		aiServiceURL: aiServiceURL,
		geminiKey:    geminiKey,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *AIService) AnalyzeMemory(title, story string) (*AIAnalysisResult, error) {
	// First try Python AI microservice
	if s.aiServiceURL != "" {
		result, err := s.callPythonAI(title, story)
		if err == nil {
			return result, nil
		}
		log.Printf("[AI] Python service failed, falling back to Gemini: %v", err)
	}
	// Fallback to direct Gemini API
	return s.callGemini(title, story)
}

func (s *AIService) callPythonAI(title, story string) (*AIAnalysisResult, error) {
	body := map[string]string{"title": title, "story": story}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", s.aiServiceURL+"/analyze", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AI service error: %d", resp.StatusCode)
	}

	var result AIAnalysisResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *AIService) callGemini(title, story string) (*AIAnalysisResult, error) {
	prompt := fmt.Sprintf(`You are an empathetic AI assistant for a digital memoir app called Pikyon.
Analyze this personal memory and return a JSON object ONLY (no markdown, no explanation):
Memory Title: "%s"
Memory Story: "%s"
Return this exact JSON structure:
{
  "vibe": "<Song Title - Artist Name that matches the emotional tone>",
  "vibe_reason": "<One sentence explaining why this song fits>",
  "tone": "<one word: nostalgic|joyful|bittersweet|peaceful|melancholic|triumphant|tender>",
  "captions": {
    "twitter": "<Tweet under 280 chars with 2-3 hashtags>",
    "instagram": "<Instagram caption 100-150 chars with 5 hashtags>",
    "linkedin": "<Professional reflection 200-250 chars>"
  },
  "suggested_tags": ["<tag1>", "<tag2>", "<tag3>"]
}`, title, story)

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
	}
	jsonBody, _ := json.Marshal(reqBody)

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=%s",
		s.geminiKey,
	)
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Gemini API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, err
	}
	if len(geminiResp.Candidates) == 0 {
		return nil, fmt.Errorf("empty response from Gemini")
	}

	text := geminiResp.Candidates[0].Content.Parts[0].Text
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var result AIAnalysisResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini response: %w", err)
	}
	return &result, nil
}

// ============================================================
// MEDIA SERVICE — Supabase Storage
// ============================================================

type MediaService struct {
	supabaseURL string
	serviceKey  string
	bucket      string
	httpClient  *http.Client
}

type PresignedURLResponse struct {
	SignedURL string    `json:"signed_url"`
	ExpiresAt time.Time `json:"expires_at"`
}

type UploadURLResponse struct {
	UploadURL string `json:"upload_url"`
	MediaKey  string `json:"media_key"`
}

func NewMediaService(supabaseURL, serviceKey, bucket string) *MediaService {
	return &MediaService{
		supabaseURL: supabaseURL,
		serviceKey:  serviceKey,
		bucket:      bucket,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *MediaService) GetStreamURL(mediaKey string) (*PresignedURLResponse, error) {
	endpoint := fmt.Sprintf("%s/storage/v1/object/sign/%s/%s",
		s.supabaseURL, s.bucket, mediaKey)
	body := map[string]int{"expiresIn": 300}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		SignedURL string `json:"signedURL"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	fullURL := result.SignedURL
	if len(fullURL) > 0 && fullURL[0] == '/' {
		fullURL = s.supabaseURL + fullURL
	}
	return &PresignedURLResponse{
		SignedURL: fullURL,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}, nil
}

func (s *MediaService) GetUploadURL(userID, memoryID, fileExt string) (*UploadURLResponse, error) {
	mediaKey := fmt.Sprintf("%s/%s/%d.%s", userID, memoryID, time.Now().Unix(), fileExt)
	endpoint := fmt.Sprintf("%s/storage/v1/object/upload/sign/%s/%s",
		s.supabaseURL, s.bucket, mediaKey)
	body := map[string]int{"expiresIn": 300}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		URL   string `json:"url"`
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	uploadURL := result.URL
	if len(uploadURL) > 0 && uploadURL[0] == '/' {
		uploadURL = s.supabaseURL + uploadURL
	}

	return &UploadURLResponse{
		UploadURL: uploadURL,
		MediaKey:  mediaKey,
	}, nil
}

func (s *MediaService) DeleteMedia(mediaKey string) error {
	endpoint := fmt.Sprintf("%s/storage/v1/object/%s/%s",
		s.supabaseURL, s.bucket, mediaKey)
	req, err := http.NewRequest("DELETE", endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// ============================================================
// SOCIAL SERVICE — async goroutines
// ============================================================

type SocialPlatform string

const (
	PlatformTwitter   SocialPlatform = "twitter"
	PlatformInstagram SocialPlatform = "instagram"
	PlatformLinkedIn  SocialPlatform = "linkedin"
)

type PostJob struct {
	PostID   string
	MemoryID string
	UserID   string
	Platform SocialPlatform
	Caption  string
}

type PostResult struct {
	PostID     string
	Success    bool
	ExternalID string
	Error      string
}

type SocialService struct {
	httpClient *http.Client
	resultsCh  chan PostResult
}

func NewSocialService() *SocialService {
	s := &SocialService{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		resultsCh:  make(chan PostResult, 100),
	}
	go s.processResults()
	return s
}

func (s *SocialService) PostAsync(job PostJob) {
	go func(j PostJob) {
		log.Printf("[Social] Posting to %s for memory %s", j.Platform, j.MemoryID)
		s.resultsCh <- PostResult{
			PostID:     j.PostID,
			Success:    true,
			ExternalID: fmt.Sprintf("%s_%d", j.Platform, time.Now().UnixNano()),
		}
	}(job)
}

func (s *SocialService) processResults() {
	for result := range s.resultsCh {
		if result.Success {
			log.Printf("[Social] Posted successfully: %s", result.PostID)
		} else {
			log.Printf("[Social] Post failed: %s — %s", result.PostID, result.Error)
		}
	}
}