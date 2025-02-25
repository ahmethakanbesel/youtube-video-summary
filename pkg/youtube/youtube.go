package youtube

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Client represents the YouTube API client
type Client struct {
	httpClient *http.Client
	apiKey     string
	logger     *slog.Logger
}

// NewClient creates a new YouTube client
func NewClient(apiKey string, insecureSkipVerify bool, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}

	httpTransport := &http.Transport{
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	if insecureSkipVerify {
		httpTransport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second, Transport: httpTransport},
		apiKey:     apiKey,
		logger:     logger,
	}
}

// Logger returns the client's logger
func (c *Client) Logger() *slog.Logger {
	return c.logger
}

// TranscriptSegment represents a single segment of the transcript
type TranscriptSegment struct {
	Text      string  `json:"text"`
	StartTime float64 `json:"start"`
	Duration  float64 `json:"duration"`
}

// Transcript represents the full transcript
type Transcript struct {
	Segments []TranscriptSegment `json:"segments"`
}

// TranscriptResponse combines raw and formatted transcripts
type TranscriptResponse struct {
	Title     string      `json:"title"`
	Raw       *Transcript `json:"raw"`
	Formatted []string    `json:"formatted"`
}

// GetTranscript fetches the raw transcript and title from YouTube
func (c *Client) GetTranscript(ctx context.Context, videoID string) (*TranscriptResponse, error) {
	playerResp, err := c.getPlayerResponse(ctx, videoID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get player response")
	}

	// Extract video title from player response
	title := playerResp.VideoDetails.Title
	if title == "" {
		c.logger.Warn("No title found in player response")
	}

	captionTracks := c.extractCaptionTracks(playerResp)
	c.logger.Info("Found caption tracks", "count", len(captionTracks))
	if len(captionTracks) == 0 {
		return nil, errors.New("no caption tracks available")
	}

	var captionURL string
	for _, track := range captionTracks {
		c.logger.Debug("Caption track details", "VssID", track.VssID, "LanguageCode", track.LanguageCode, "URL", track.BaseURL)
		if strings.HasPrefix(track.VssID, ".en") || track.LanguageCode == "en" {
			captionURL = track.BaseURL
			break
		}
	}
	if captionURL == "" {
		captionURL = captionTracks[0].BaseURL
		c.logger.Debug("No English captions found, using default", "url", captionURL)
	}

	ttmlURL := fmt.Sprintf("%s&fmt=ttml", captionURL)
	resp, err := c.httpClient.Get(ttmlURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch transcript")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read TTML response")
	}
	c.logger.Debug("TTML response", "length", len(bodyBytes), "snippet", string(bodyBytes[:min(500, len(bodyBytes))]))

	segments, err := parseTTMLTranscript(bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TTML transcript")
	}
	c.logger.Info("Parsed segments", "count", len(segments))

	return &TranscriptResponse{
		Title: title,
		Raw:   &Transcript{Segments: segments},
	}, nil
}

// GetFormattedTranscript fetches and formats the transcript with title
func (c *Client) GetFormattedTranscript(ctx context.Context, videoID string, intervalSeconds float64) (*TranscriptResponse, error) {
	transcriptResp, err := c.GetTranscript(ctx, videoID)
	if err != nil {
		return nil, err
	}
	formatted, err := c.FormatTranscript(ctx, transcriptResp.Raw, intervalSeconds)
	if err != nil {
		return nil, err
	}
	transcriptResp.Formatted = formatted
	return transcriptResp, nil
}

// FormatTranscript generates a formatted transcript from an existing raw transcript
func (c *Client) FormatTranscript(ctx context.Context, transcript *Transcript, intervalSeconds float64) ([]string, error) {
	if transcript == nil || len(transcript.Segments) == 0 {
		c.logger.Warn("No segments found in transcript")
		return nil, nil
	}

	var formatted []string
	currentStart := transcript.Segments[0].StartTime
	var groupText strings.Builder

	for _, segment := range transcript.Segments {
		if segment.StartTime-currentStart >= intervalSeconds && groupText.Len() > 0 {
			formatted = append(formatted, formatTimeText(currentStart, groupText.String()))
			currentStart = segment.StartTime
			groupText.Reset()
		}
		if groupText.Len() > 0 {
			groupText.WriteString(" ")
		}
		groupText.WriteString(segment.Text)
	}

	if groupText.Len() > 0 {
		formatted = append(formatted, formatTimeText(currentStart, groupText.String()))
	}

	c.logger.Info("Formatted transcript", "groups", len(formatted))
	return formatted, nil
}

func formatTimeText(startTime float64, text string) string {
	hours := int(startTime / 3600)
	minutes := int((startTime - float64(hours*3600)) / 60)
	seconds := int(startTime - float64(hours*3600+minutes*60))
	if hours > 0 {
		return fmt.Sprintf("(%02d:%02d:%02d) %s", hours, minutes, seconds, text)
	}
	return fmt.Sprintf("(%02d:%02d) %s", minutes, seconds, text)
}

type playerResponse struct {
	Captions struct {
		PlayerCaptionsTracklistRenderer struct {
			CaptionTracks []struct {
				BaseURL      string `json:"baseUrl"`
				VssID        string `json:"vssId"`
				LanguageCode string `json:"languageCode"`
			} `json:"captionTracks"`
		} `json:"playerCaptionsTracklistRenderer"`
	} `json:"captions"`
	VideoDetails struct {
		Title string `json:"title"`
	} `json:"videoDetails"`
}

func (c *Client) getPlayerResponse(ctx context.Context, videoID string) (*playerResponse, error) {
	endpoint := "https://www.youtube.com/youtubei/v1/player"
	data := map[string]interface{}{
		"context": map[string]interface{}{
			"client": map[string]interface{}{
				"clientName":    "WEB",
				"clientVersion": "2.20241126.01.00",
				"hl":            "en",
			},
		},
		"videoId": videoID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request data")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		q := req.URL.Query()
		q.Add("key", c.apiKey)
		req.URL.RawQuery = q.Encode()
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to perform request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var playerResp playerResponse
	if err := json.NewDecoder(resp.Body).Decode(&playerResp); err != nil {
		return nil, errors.Wrap(err, "failed to decode player response")
	}

	return &playerResp, nil
}

func (c *Client) extractCaptionTracks(resp *playerResponse) []struct {
	BaseURL      string `json:"baseUrl"`
	VssID        string `json:"vssId"`
	LanguageCode string `json:"languageCode"`
} {
	return resp.Captions.PlayerCaptionsTracklistRenderer.CaptionTracks
}

type ttmlTranscript struct {
	XMLName xml.Name `xml:"tt"`
	Body    struct {
		Div struct {
			Paragraphs []struct {
				Begin string `xml:"begin,attr"`
				End   string `xml:"end,attr"`
				Text  string `xml:",chardata"`
			} `xml:"p"`
		} `xml:"div"`
	} `xml:"body"`
}

func parseTTMLTranscript(body io.Reader) ([]TranscriptSegment, error) {
	var ttml ttmlTranscript
	decoder := xml.NewDecoder(body)
	if err := decoder.Decode(&ttml); err != nil {
		return nil, errors.Wrap(err, "failed to decode TTML XML")
	}

	segments := make([]TranscriptSegment, 0, len(ttml.Body.Div.Paragraphs))
	for _, p := range ttml.Body.Div.Paragraphs {
		startTime, err := parseTime(p.Begin)
		if err != nil {
			slog.Warn("Failed to parse begin time", "time", p.Begin, "error", err)
			continue
		}
		endTime, err := parseTime(p.End)
		if err != nil {
			slog.Warn("Failed to parse end time", "time", p.End, "error", err)
			continue
		}
		segment := TranscriptSegment{
			Text:      strings.TrimSpace(p.Text),
			StartTime: startTime,
			Duration:  endTime - startTime,
		}
		if segment.Text != "" {
			segments = append(segments, segment)
		}
	}
	return segments, nil
}

func parseTime(timeStr string) (float64, error) {
	if strings.HasSuffix(timeStr, "s") {
		timeStr = strings.TrimSuffix(timeStr, "s")
		return strconv.ParseFloat(timeStr, 64)
	}
	parts := strings.Split(timeStr, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid time format: %s", timeStr)
	}
	hours, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, err
	}
	minutes, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, err
	}
	seconds, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return 0, err
	}
	return hours*3600 + minutes*60 + seconds, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
