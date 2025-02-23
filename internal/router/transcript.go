package router

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ahmethakanbesel/youtube-video-summary/internal/service"
)

type Router struct {
	service *service.Transcript
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func NewRouter(svc *service.Transcript, uiAssets embed.FS) *http.ServeMux {
	r := &Router{service: svc}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/transcripts", r.handleGetTranscripts)

	// Serve static files from the dist directory
	distFS, err := fs.Sub(uiAssets, "dist")
	if err != nil {
		panic(err)
	}
	fs := http.FileServer(http.FS(distFS))
	mux.Handle("/", fs)

	return mux
}

func (r *Router) writeJSONError(w http.ResponseWriter, errMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: errMsg,
	})
	if err != nil {
		slog.Error("Failed to encode error response", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (r *Router) handleGetTranscripts(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		r.writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	videoURL := req.URL.Query().Get("videoUrl")
	if videoURL == "" {
		r.writeJSONError(w, "Missing videoUrl parameter", http.StatusBadRequest)
		return
	}

	intervalStr := req.URL.Query().Get("interval")
	interval, err := strconv.ParseFloat(intervalStr, 64)
	if err != nil {
		interval = 0 // Will default to 10.0 in service
	}

	svcReq := service.TranscriptRequest{
		VideoURL:        videoURL,
		IntervalSeconds: interval,
	}

	resp, err := r.service.GetTranscripts(req.Context(), svcReq)
	if err != nil {
		// Map service layer errors to HTTP status codes
		switch {
		case err == service.ErrInvalidURL:
			r.writeJSONError(w, "Invalid YouTube video URL", http.StatusBadRequest)
		default:
			r.writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	if resp.Raw == nil && resp.Formatted == nil {
		r.writeJSONError(w, "No transcript available", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		r.writeJSONError(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
