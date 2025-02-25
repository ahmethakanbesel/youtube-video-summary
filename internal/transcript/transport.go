package transcript

import "github.com/ahmethakanbesel/youtube-video-summary/pkg/youtube"

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

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
