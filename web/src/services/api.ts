interface Segment {
  text: string;
  start: number;
  duration: number;
}

interface RawTranscript {
  segments: Segment[];
}

interface ApiResponse {
  raw: RawTranscript;
  formatted: string[];
  title: string;
}

interface ApiError {
  error: string;
  message: string;
}

interface Subtitle {
  time: string;
  text: string;
}

interface VideoSummary {
  title: string;
  url: string;
  videoId: string;
  thumbnailUrl: string;
  subtitles: Subtitle[];
 formatted: string[];
}

function extractVideoId(url: string): string {
  const regExp = /^.*((youtu.be\/)|(v\/)|(\/u\/\w\/)|(embed\/)|(watch\?))\??v?=?([^#&?]*).*/;
  const match = url.match(regExp);
  return (match && match[7].length === 11) ? match[7] : "";
}

function formatTime(seconds: number): string {
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = Math.floor(seconds % 60);
  return `${minutes.toString().padStart(2, '0')}:${remainingSeconds.toString().padStart(2, '0')}`;
}

const API_URL = import.meta.env.VITE_API_URL || '/api/v1';

export async function getVideoSummary(url: string): Promise<VideoSummary> {
  try {
    const response = await fetch(`${API_URL}/transcripts?videoUrl=${encodeURIComponent(url)}`);
    const data = await response.json();
    
    if (!response.ok) {
      const apiError = data as ApiError;
      throw new Error(apiError.message || 'Failed to fetch video summary');
    }

    const apiResponse = data as ApiResponse;
    const videoId = extractVideoId(url);

    // Convert raw segments to subtitles format
    const subtitles = apiResponse.raw.segments.map(segment => ({
      time: formatTime(segment.start),
      text: segment.text
    }));

    return {
      title: apiResponse.title,
      url,
      videoId,
      thumbnailUrl: `https://img.youtube.com/vi/${videoId}/maxresdefault.jpg`,
      subtitles,
      formatted: apiResponse.formatted,
    };
  } catch (error) {
    console.error('Error fetching video summary:', error);
    throw error;
  }
}