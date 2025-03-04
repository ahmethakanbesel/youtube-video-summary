# YouTube AI Video Summary

This project is a tool that generates YouTube video summaries using popular LLM (Large Language Model) platforms like ChatGPT and Claude. It fetches video transcripts from YouTube and provides quick chat links to LLM platforms for summarization. The tool is self-contained and requires no external dependencies or APIs.

## Usage

### Pre-built Binary

1. Download the latest release for your operating system (Windows, macOS, or Linux) from the [releases page](https://github.com/ahmethakanbesel/youtube-video-summary/releases).
2. Extract the downloaded archive
3. Run the executable file
4. Access the web interface at `http://localhost:8080`

Default port can be changed by setting the `PORT` environment variable.

### Building from source

You can quick start it on your computer with the following command:

```bash
git clone https://github.com/ahmethakanbesel/youtube-video-summary
cd youtube-video-summary
docker compose up -d
```

If you need to build it without Docker, `bun` and `go` are required. After installing `bun`, you can run the following command:

```bash
make build
```

The command above will build the frontend first and then embed it into the Go binary.

## Preview

![Web UI Preview](./docs/preview.png)

## License

This is free and unencumbered software released into the public domain.

Anyone is free to copy, modify, publish, use, compile, sell, or
distribute this software, either in source code form or as a compiled
binary, for any purpose, commercial or non-commercial, and by any
means.

In jurisdictions that recognize copyright laws, the author or authors
of this software dedicate any and all copyright interest in the
software to the public domain. We make this dedication for the benefit
of the public at large and to the detriment of our heirs and
successors. We intend this dedication to be an overt act of
relinquishment in perpetuity of all present and future rights to this
software under copyright law.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS BE LIABLE FOR ANY CLAIM, DAMAGES OR
OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
OTHER DEALINGS IN THE SOFTWARE.

For more information, please refer to <https://unlicense.org>
