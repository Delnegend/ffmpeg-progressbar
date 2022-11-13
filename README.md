# FFmpeg progress bar
## Requirements
- [FFmpeg, FFprobe](https://www.gyan.dev/ffmpeg/builds/)
- [Python 3.6+](https://www.python.org/downloads/) (for running the script)
- [Go 1.14+](https://golang.org/dl/) (for building the binary)

## Usage
There are two ways to use this script:
### Python:
- Replace `ffmpeg` with `python ffmpeg.py` in your command.
### Golang:
- Build the binary with `go build ffmpegbar.go`.
- Replace `ffmpeg` with `ffmpegbar` in your command.

## Note
- This requires input and output video to have the same framecount.
- Only works with video transcoding.