# Go Whisper Client

A Go-based command-line tool for Whisper API, supporting audio and video transcription.

[中文文档](README.md)

## Features

- ✅ Support for audio and video file input
- ✅ Automatic video-to-audio conversion (using ffmpeg)
- ✅ Multiple output formats (TXT, SRT, JSON)
- ✅ Configuration management via config file
- ✅ Automatic language detection or manual specification
- ✅ Support for multiple Whisper models
- ✅ **Automatic large file chunking** - Split and transcribe large files automatically based on silence detection
- ✅ **Intelligent silence detection** - Split at speech pauses to preserve semantic integrity
- ✅ **Automatic timestamp correction** - Merged results have accurate timestamps aligned with original media

## Installation

### Prerequisites

1. Go 1.21 or higher
2. ffmpeg (for video-to-audio conversion and silence detection)

### Build

```bash
cd go-whisper-go
go build -o whisper-go.exe .
```

## Usage

### 1. Configure API

Edit the `config.json` file:

```json
{
  "api_base_url": "https://api.groq.com/openai/v1",
  "api_key": "your-api-key-here",
  "model": "whisper-large-v3",
  "language": "zh",
  "auto_detect": true,
  "output_dir": "./outputs",
  "max_file_size_mb": 10,
  "silence_threshold": "-30dB",
  "silence_duration": 0.5
}
```

### 2. Run Transcription

```bash
# Basic usage
whisper-go.exe input.mp4

# Specify language
whisper-go.exe input.mp4 --language en

# Auto-detect language
whisper-go.exe input.mp4 --auto-detect

# Specify output directory
whisper-go.exe input.mp4 --output ./my-output

# Use different model
whisper-go.exe input.mp4 --model whisper-medium

# Output specific formats only
whisper-go.exe input.mp4 --formats txt,srt

# Show verbose output
whisper-go.exe input.mp4 --verbose
```

### 3. Command Line Arguments

| Argument | Description | Default |
|----------|-------------|---------|
| `input` | Input file path | Required |
| `--config` | Configuration file path | ./config.json |
| `--language` | Language code (e.g., zh, en, ja) | Read from config |
| `--auto-detect` | Auto-detect language | Read from config |
| `--model` | Whisper model name | Read from config |
| `--output` | Output directory | Read from config |
| `--formats` | Output formats (comma-separated) | txt,srt,json |
| `--verbose` | Show verbose output | false |

## Large File Chunking

When the input file exceeds the configured `max_file_size_mb` threshold, the tool automatically performs chunking:

### Processing Flow

```
Input File → File Size Check
              │
              ├─ ≤ Threshold → Direct Transcription
              │
              └─ > Threshold → Silence Detection → Chunking → Sequential Transcription → Merge Results (Correct Timestamps)
```

### Chunking Strategy

1. **Silence Detection**: Uses ffmpeg `silencedetect` filter to identify speech pauses
2. **Smart Splitting**: Prioritizes splitting at silence points to avoid cutting off words/sentences
3. **Timestamp Correction**: Automatically adjusts timestamps when merging results to align with original media

### Example Output

```
File size 45.2 MB exceeds threshold 10 MB, chunking will be performed
Detecting silence points: audio.wav
Detected 15 silence points
Audio duration: 541.00 seconds, File size: 45.20 MB
Planning to split into 5 chunks, approximately 108.2 seconds each
Split time points: [102.5 215.3 318.7 430.2]
Creating chunk 1: 0.00 - 102.50 seconds
Creating chunk 2: 102.50 - 215.30 seconds
Creating chunk 3: 215.30 - 318.70 seconds
Creating chunk 4: 318.70 - 430.20 seconds
Creating chunk 5: 430.20 - 541.00 seconds

Created 5 chunks, starting transcription...

Transcription progress: 1/5
Transcription progress: 2/5
Transcription progress: 3/5
Transcription progress: 4/5
Transcription progress: 5/5

Chunk transcription complete, results merged
Temporary chunk files cleaned up
```

## Supported Formats

### Input Formats

- **Audio**: MP3, WAV, M4A, AAC, FLAC, OGG
- **Video**: MP4, AVI, MOV, MKV, FLV, WMV, WEBM

### Output Formats

- **TXT**: Plain text format (line-separated by segments for better readability)
- **SRT**: Subtitle format (with timestamps)
- **JSON**: Complete structured data (including segment information)

## Configuration Reference

| Field | Description | Default |
|-------|-------------|---------|
| `api_base_url` | API base URL | - |
| `api_key` | API key | - |
| `model` | Whisper model name | whisper-large-v3 |
| `language` | Language code (e.g., zh, en, ja) | zh |
| `auto_detect` | Whether to auto-detect language | true |
| `output_dir` | Output directory path | ./outputs |
| `max_file_size_mb` | File size threshold (MB) for chunking | 10 |
| `silence_threshold` | Silence detection sensitivity | -30dB |
| `silence_duration` | Minimum silence duration (seconds) | 0.5 |

### Supported Models

- `whisper-large-v3`
- `whisper-large-v2`
- `whisper-medium`
- `whisper-small`
- `whisper-base`
- `whisper-tiny`
- `whisper-1` (OpenAI format)

## Examples

### Example 1: Transcribe Video File

```bash
whisper-go.exe video.mp4
```

Output:
- `video_20240222_153020.txt` - Transcribed text
- `video_20240222_153020.srt` - Subtitle file
- `video_20240222_153020.json` - Complete data

### Example 2: Specify Language and Format

```bash
whisper-go.exe audio.wav --language en --formats txt,json
```

### Example 3: Auto-detect Language

```bash
whisper-go.exe mixed.mp4 --auto-detect
```

### Example 4: Process Large File

```bash
whisper-go.exe large_video.mp4 --verbose
```

The tool will automatically detect file size and perform chunking, showing detailed progress.

## Notes

1. First-time use requires configuring `api_key` in `config.json`
2. Ensure ffmpeg is installed and available in PATH
3. Video files are automatically converted to WAV format (16kHz mono)
4. Output filenames include timestamps to avoid overwriting
5. Large file chunking generates temporary files that are automatically cleaned up after transcription

## License

MIT License
