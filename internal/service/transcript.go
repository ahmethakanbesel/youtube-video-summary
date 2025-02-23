package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/ahmethakanbesel/youtube-video-summary/internal/repository"
	"github.com/ahmethakanbesel/youtube-video-summary/pkg/youtube"
)

var (
	ErrNoTranscript   = errors.New("no transcript available")
	ErrFailedToGet    = errors.New("failed to get transcript")
	ErrFailedToFormat = errors.New("failed to format transcript")
	ErrInvalidURL     = errors.New("invalid YouTube video URL")
)

type Transcript struct {
	client *youtube.Client
	repo   repository.Transcript
}

func NewTranscript(client *youtube.Client, repo repository.Transcript) *Transcript {
	return &Transcript{
		client: client,
		repo:   repo,
	}
}

type TranscriptRequest struct {
	VideoURL        string
	VideoID         string
	IntervalSeconds float64
}

type TranscriptResponse struct {
	Title     string              `json:"title"`
	Raw       *youtube.Transcript `json:"raw"`
	Formatted []string            `json:"formatted"`
}

func (s *Transcript) GetTranscripts(ctx context.Context, req TranscriptRequest) (TranscriptResponse, error) {
	interval := req.IntervalSeconds
	if interval <= 0 {
		interval = 10.0
	}

	// Validate video URL
	if req.VideoURL == "" || !s.IsValidUrl(req.VideoURL) {
		return TranscriptResponse{}, ErrInvalidURL
	}

	// Extract video ID from URL if not provided
	if req.VideoID == "" {
		req.VideoID = s.ExtractVideoId(req.VideoURL)
		if req.VideoID == "" {
			return TranscriptResponse{}, ErrInvalidURL
		}
	}

	var youtubeResp *youtube.TranscriptResponse
	var err error

	// Try to get from cache first
	youtubeResp, err = s.repo.Get(ctx, req.VideoID)
	if err != nil {
		if !errors.Is(err, repository.ErrTranscriptNotFound) {
			s.client.Logger().Error("Failed to get transcript from repository", "video_id", req.VideoID, "error", err)
		}

		// If not in cache or error, fetch from YouTube
		youtubeResp, err = s.client.GetTranscript(ctx, req.VideoID)
		if err != nil {
			s.client.Logger().Error("Failed to fetch raw transcript", "video_id", req.VideoID, "error", err)
			return TranscriptResponse{}, fmt.Errorf("%w: %v", ErrFailedToGet, err)
		}

		// Validate YouTube response
		if youtubeResp == nil || youtubeResp.Raw == nil || len(youtubeResp.Raw.Segments) == 0 {
			s.client.Logger().Warn("No transcript available", "video_id", req.VideoID)
			return TranscriptResponse{}, ErrNoTranscript
		}

		// Cache the successful response
		if err := s.repo.Save(ctx, req.VideoID, youtubeResp); err != nil {
			s.client.Logger().Error("Failed to cache transcript", "video_id", req.VideoID, "error", err)
			// Continue despite cache error
		}
	}

	// Create response
	resp := TranscriptResponse{
		Title: youtubeResp.Title,
		Raw:   youtubeResp.Raw,
	}

	// Format the transcript
	formatted, err := s.client.FormatTranscript(ctx, youtubeResp.Raw, interval)
	if err != nil {
		s.client.Logger().Error("Failed to format transcript", "video_id", req.VideoID, "error", err)
		return TranscriptResponse{}, fmt.Errorf("%w: %v", ErrFailedToFormat, err)
	}
	resp.Formatted = formatted

	return resp, nil
}

// ExtractVideoId attempts to extract a YouTube video ID from a string.
// It can handle both direct 11-character IDs and various URL formats.
// Returns empty string if no valid video ID is found.
func (s *Transcript) ExtractVideoId(str string) string {
	// Check if the string is exactly 11 characters (direct video ID)
	if len(str) == 11 {
		return str
	}

	// Regular expression to match YouTube video ID in various URL formats
	pattern := `(?:\/|%3D|v=|vi=)([a-zA-Z0-9_-]{11})(?:[%#?&\/]|$)`
	regex := regexp.MustCompile(pattern)
	matches := regex.FindStringSubmatch(str)

	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// IsValidUrl checks if the provided URL has a valid YouTube domain.
// It handles domains: youtu.be, youtube.com, m.youtube.com, with or without www.
// Returns true if the domain is a valid YouTube domain, false otherwise.
func (s *Transcript) IsValidUrl(urlStr string) bool {
	// Parse the URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	// Extract the host
	host := strings.ToLower(parsedURL.Host)

	// Remove 'www.' if present
	host = strings.TrimPrefix(host, "www.")

	// List of valid YouTube domains
	validDomains := []string{
		"youtube.com",
		"youtu.be",
		"m.youtube.com",
	}

	// Check if the host matches any valid domain
	for _, domain := range validDomains {
		if host == domain {
			return true
		}
	}

	return false
}
