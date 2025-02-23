<script lang="ts">
  import { onMount } from 'svelte';
  import { getVideoSummary } from './services/api';

  let videoUrl = '';
  let loading = false;
  let error = '';
  let videoData: {
    title: string;
    url: string;
    videoId: string;
    thumbnailUrl: string;
    subtitles: Array<{ time: string; text: string }>;
    formatted: string[];
  } | null = null;
  let showSubtitles = false;
  let timeout: NodeJS.Timeout;

  const isValidYoutubeUrl = (url: string): boolean => {
    const pattern = /^(https?:\/\/)?(www\.)?(youtube\.com|youtu\.be)\/.+$/;
    return pattern.test(url);
  };

  const fetchSummary = async (url: string) => {
    if (!isValidYoutubeUrl(url)) {
      error = 'Please enter a valid YouTube URL';
      return;
    }

    try {
      loading = true;
      error = '';
      videoData = await getVideoSummary(url);
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to fetch video summary';
      videoData = null;
    } finally {
      loading = false;
    }
  };

  const getAiPrompt = () => {
    if (!videoData) return '';
    return encodeURIComponent(`Please summarize the video below\n\n${videoData.formatted.join('\n')}`);
  };

  $: if (videoUrl) {
    clearTimeout(timeout);
    timeout = setTimeout(() => {
      fetchSummary(videoUrl);
    }, 500);
  }

  onMount(() => {
    return () => {
      clearTimeout(timeout);
    };
  });
</script>

<main class="container">
  <div class="content-wrapper">
    <h1 class="text-4xl font-bold mb-8" aria-level="1">
      YouTube Video Summary
    </h1>

    <div class="mb-8 input-container">
      <label for="videoUrl" class="block mb-2">
        Enter YouTube Video URL
      </label>
      <input
        type="url"
        id="videoUrl"
        bind:value={videoUrl}
        placeholder="https://youtube.com/watch?v=..."
        class="neo-brutal-input"
        aria-label="YouTube video URL input"
        aria-invalid={error ? 'true' : 'false'}
        aria-describedby={error ? 'urlError' : undefined}
      />
      {#if error}
        <div id="urlError" class="neo-brutal-error" role="alert">
          <p>{error}</p>
        </div>
      {/if}
    </div>

    {#if loading}
      <div class="neo-brutal-card" role="status">
        <p>Loading summary...</p>
      </div>
    {/if}

    {#if videoData && !loading}
      <div class="neo-brutal-card">
        <div class="video-info mb-6">
          <img
            src={videoData.thumbnailUrl}
            alt={`Thumbnail for ${videoData.title}`}
            class="video-thumbnail"
            loading="lazy"
          />
          <h2 class="text-2xl font-bold mt-4">{videoData.title}</h2>
        </div>

        <div class="mb-6">
          <h3 class="text-xl mb-4">Generate AI Summary</h3>
          <div class="summary-buttons">
            <a
              href={`https://chatgpt.com/?q=${getAiPrompt()}`}
              target="_blank"
              rel="noopener noreferrer"
              class="neo-brutal-button chatgpt"
            >
              ChatGPT
            </a>
            <a
              href={`https://claude.ai/new?q=${getAiPrompt()}`}
              target="_blank"
              rel="noopener noreferrer"
              class="neo-brutal-button claude"
            >
              Claude
            </a>
            <a
              href={`https://www.perplexity.ai/search?q=${getAiPrompt()}`}
              target="_blank"
              rel="noopener noreferrer"
              class="neo-brutal-button gemini"
            >
              Perplexity
            </a>
          </div>
        </div>

        <div class="subtitles-section">
          <button
            class="collapsible w-full text-left"
            on:click={() => showSubtitles = !showSubtitles}
            aria-expanded={showSubtitles}
            aria-controls="subtitles"
          >
            {showSubtitles ? 'Hide' : 'Show'} Subtitles
          </button>

          {#if showSubtitles}
            <div
              id="subtitles"
              class="mt-4 p-4 bg-white border-2 border-black"
              role="region"
              aria-label="Video subtitles"
            >
              {#each videoData.subtitles as subtitle}
                <div class="subtitle-entry">
                  <span class="time">{subtitle.time}</span>
                  <span class="text">{subtitle.text}</span>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    {/if}
  </div>
</main>

<style>
  .container {
    min-height: 100vh;
    width: 100%;
    max-width: 800px;
    margin: 0 auto;
    padding: 2rem;
    display: flex;
    flex-direction: column;
  }

  .content-wrapper {
    flex: 1;
    width: 100%;
  }

  .input-container {
    width: 100%;
  }

  .video-thumbnail {
    width: 100%;
    height: auto;
    border: 3px solid var(--text);
    box-shadow: var(--shadow);
    border-radius: 4px;
  }

  .subtitle-entry {
    display: flex;
    gap: 1rem;
    margin-bottom: 0.5rem;
    padding: 0.5rem;
    border-bottom: 1px solid #eee;
  }

  .time {
    font-weight: bold;
    min-width: 60px;
  }

  .summary-buttons {
    display: flex;
    flex-wrap: wrap;
    gap: 1.5rem;
  }

  .neo-brutal-button {
    display: inline-block;
    padding: 0.75rem 1.5rem;
    font-weight: bold;
    text-decoration: none;
    border: 3px solid var(--text);
    box-shadow: var(--shadow);
    transition: all 0.2s ease;
    min-width: 140px;
    text-align: center;
  }

  .neo-brutal-button:hover {
    transform: translate(-2px, -2px);
    box-shadow: 6px 6px 0 rgba(0, 0, 0, 0.9);
  }

  .neo-brutal-error {
    margin-top: 0.5rem;
    padding: 1rem;
    background: #fee2e2;
    border: 3px solid var(--text);
    box-shadow: var(--shadow);
    color: #991b1b;
    font-weight: bold;
  }

  .chatgpt {
    background-color: #19c37d;
    color: white;
  }

  .claude {
    background-color: #6b40ef;
    color: white;
  }

  .gemini {
    background-color: #1a73e8;
    color: white;
  }

  @media (max-width: 640px) {
    .container {
      padding: 1rem;
    }

    h1 {
      font-size: 2rem;
    }

    .neo-brutal-button {
      width: 100%;
      text-align: center;
      margin-bottom: 0.5rem;
    }

    .summary-buttons {
      gap: 1rem;
    }
  }
</style>