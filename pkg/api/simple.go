package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"xeet/pkg/config"

	"github.com/dghubble/oauth1"
	"golang.org/x/time/rate"
)

type Client struct {
	httpClient  *http.Client
	rateLimiter *rate.Limiter
	config      *config.Config
	oauthConfig *oauth1.Config
	token       *oauth1.Token
}

type TweetRequest struct {
	Text  string       `json:"text"`
	Media *MediaUpload `json:"media,omitempty"`
}

type MediaUpload struct {
	MediaIDs []string `json:"media_ids"`
}

type MediaUploadResponse struct {
	MediaID          int64  `json:"media_id"`
	MediaIDString    string `json:"media_id_string"`
	Size             int    `json:"size"`
	ExpiresAfterSecs int    `json:"expires_after_secs"`
	Image            struct {
		ImageType string `json:"image_type"`
		W         int    `json:"w"`
		H         int    `json:"h"`
	} `json:"image"`
}

type TweetResponse struct {
	Data struct {
		ID   string `json:"id"`
		Text string `json:"text"`
	} `json:"data"`
}

func NewClient(cfg *config.Config) *Client {
	oauthConfig := oauth1.NewConfig(cfg.APIKey, cfg.APISecret)
	token := oauth1.NewToken(cfg.AccessToken, cfg.AccessTokenSecret)

	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimiter: rate.NewLimiter(rate.Every(15*time.Second), 1),
		config:      cfg,
		oauthConfig: oauthConfig,
		token:       token,
	}
}

func (c *Client) PostTweet(ctx context.Context, text string) error {
	return c.PostTweetWithMedia(ctx, text, nil)
}

func (c *Client) PostTweetWithMedia(ctx context.Context, text string, imageData []byte) error {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit: %w", err)
	}

	tweetReq := TweetRequest{Text: text}

	// if imageData is not nil, upload the image
	if imageData != nil {
		mediaID, err := c.uploadMedia(ctx, imageData)
		if err != nil {
			return fmt.Errorf("media upload: %w", err)
		}
		tweetReq.Media = &MediaUpload{MediaIDs: []string{mediaID}}
	}

	jsonData, err := json.Marshal(tweetReq)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.x.com/2/tweets", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	oauthClient := c.oauthConfig.Client(ctx, c.token)

	resp, err := oauthClient.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) uploadMedia(ctx context.Context, imageData []byte) (string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("media", "image.png")
	if err != nil {
		return "", err
	}

	_, err = part.Write(imageData)
	if err != nil {
		return "", err
	}

	err = writer.Close()
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://upload.twitter.com/1.1/media/upload.json", &buf)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	oauthClient := c.oauthConfig.Client(ctx, c.token)

	resp, err := oauthClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("media upload failed %d: %s", resp.StatusCode, string(body))
	}

	var uploadResp MediaUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return "", err
	}

	return uploadResp.MediaIDString, nil
}

func (c *Client) VerifyCredentials(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.x.com/2/users/me", nil)
	if err != nil {
		return err
	}

	oauthClient := c.oauthConfig.Client(ctx, c.token)
	resp, err := oauthClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("auth failed %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
