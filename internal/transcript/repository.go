package transcript

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/ahmethakanbesel/youtube-video-summary/pkg/youtube"
)

var (
	ErrTranscriptNotFound = errors.New("transcript not found")
	ErrInvalidTranscript  = errors.New("invalid transcript")
)

type Repository interface {
	Get(ctx context.Context, videoID string) (*youtube.TranscriptResponse, error)
	Save(ctx context.Context, videoID string, transcript *youtube.TranscriptResponse) error
	Clear(ctx context.Context) error
	Size() int
}

type MemoryRepository struct {
	logger    *slog.Logger
	cache     map[string]*youtube.TranscriptResponse
	cacheLock sync.RWMutex
}

var _ Repository = (*MemoryRepository)(nil)

func NewMemoryRepository(logger *slog.Logger) *MemoryRepository {
	if logger == nil {
		logger = slog.Default()
	}

	return &MemoryRepository{
		logger: logger,
		cache:  make(map[string]*youtube.TranscriptResponse),
	}
}

func (r *MemoryRepository) Get(ctx context.Context, videoID string) (*youtube.TranscriptResponse, error) {
	if videoID == "" {
		return nil, errors.New("video ID cannot be empty")
	}

	r.cacheLock.RLock()
	defer r.cacheLock.RUnlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		transcript, exists := r.cache[videoID]
		if !exists {
			r.logger.Debug("Cache miss", "video_id", videoID)
			return nil, ErrTranscriptNotFound
		}

		if transcript == nil {
			r.logger.Warn("Found nil transcript in cache", "video_id", videoID)
			return nil, ErrInvalidTranscript
		}

		r.logger.Debug("Cache hit", "video_id", videoID)
		// Return a copy to prevent modifications to cached data
		transcriptCopy := *transcript
		return &transcriptCopy, nil
	}
}

func (r *MemoryRepository) Save(ctx context.Context, videoID string, transcript *youtube.TranscriptResponse) error {
	if videoID == "" {
		return errors.New("video ID cannot be empty")
	}
	if transcript == nil {
		return ErrInvalidTranscript
	}

	r.cacheLock.Lock()
	defer r.cacheLock.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Make a copy of the transcript to prevent external modifications
		transcriptCopy := *transcript
		r.cache[videoID] = &transcriptCopy
		r.logger.Debug("Cached transcript",
			"video_id", videoID,
			"cache_size", len(r.cache),
		)
		return nil
	}
}

func (r *MemoryRepository) Clear(ctx context.Context) error {
	r.cacheLock.Lock()
	defer r.cacheLock.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		r.cache = make(map[string]*youtube.TranscriptResponse)
		r.logger.Info("Cache cleared")
		return nil
	}
}

func (r *MemoryRepository) Size() int {
	r.cacheLock.RLock()
	defer r.cacheLock.RUnlock()
	return len(r.cache)
}
