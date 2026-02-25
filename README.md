# Go Whisper Client

基于Go语言的Whisper API命令行工具，支持音频和视频文件的文本转写。

[English Documentation](README_EN.md)

## 功能特性

- ✅ 支持音频和视频文件输入
- ✅ 自动将视频文件转换为音频（使用ffmpeg）
- ✅ 支持多种输出格式（TXT、SRT、JSON）
- ✅ 通过配置文件管理API配置
- ✅ 自动检测语言或手动指定
- ✅ 支持多种Whisper模型
- ✅ **大文件自动切片处理** - 超过阈值自动按静音切片转写
- ✅ **智能静音检测** - 在语音停顿处分割，保证语义完整
- ✅ **时间戳自动修正** - 切片合并后时间戳准确对应原始音视频

## 安装

### 前置要求

1. Go 1.21 或更高版本
2. ffmpeg（用于视频转音频和静音检测）

### 编译

```bash
cd go-whisper-go
go build -o whisper-go.exe .
```

## 使用方法

### 1. 配置API

编辑 `config.json` 文件：

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

### 2. 运行转写

```bash
# 基本用法
whisper-go.exe input.mp4

# 指定语言
whisper-go.exe input.mp4 --language en

# 自动检测语言
whisper-go.exe input.mp4 --auto-detect

# 指定输出目录
whisper-go.exe input.mp4 --output ./my-output

# 使用不同模型
whisper-go.exe input.mp4 --model whisper-medium

# 只输出特定格式
whisper-go.exe input.mp4 --formats txt,srt

# 显示详细输出
whisper-go.exe input.mp4 --verbose
```

### 3. 命令行参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `input` | 输入文件路径 | 必填 |
| `--config` | 配置文件路径 | ./config.json |
| `--language` | 语言代码（如 zh, en, ja） | 从配置文件读取 |
| `--auto-detect` | 自动检测语言 | 从配置文件读取 |
| `--model` | Whisper 模型名称 | 从配置文件读取 |
| `--output` | 输出目录 | 从配置文件读取 |
| `--formats` | 输出格式（逗号分隔） | txt,srt,json |
| `--verbose` | 显示详细输出 | false |

## 大文件切片处理

当输入文件超过配置的 `max_file_size_mb` 阈值时，工具会自动进行切片处理：

### 处理流程

```
输入文件 → 文件大小检查
              │
              ├─ ≤ 阈值 → 直接转写
              │
              └─ > 阈值 → 静音检测 → 切片 → 逐个转写 → 合并结果（修正时间戳）
```

### 切片策略

1. **静音检测**：使用 ffmpeg `silencedetect` 滤镜检测语音停顿点
2. **智能分割**：优先在静音处分割，避免截断词语/句子
3. **时间戳修正**：合并结果时自动调整时间戳，确保与原始音视频对应

### 示例输出

```
文件大小 45.2 MB 超过阈值 10 MB，将进行切片处理
正在检测静音点: audio.wav
检测到 15 个静音点
音频时长: 541.00 秒, 文件大小: 45.20 MB
计划分割为 5 片，每片约 108.2 秒
切片时间点: [102.5 215.3 318.7 430.2]
创建切片 1: 0.00 - 102.50 秒
创建切片 2: 102.50 - 215.30 秒
创建切片 3: 215.30 - 318.70 秒
创建切片 4: 318.70 - 430.20 秒
创建切片 5: 430.20 - 541.00 秒

共创建 5 个切片，开始转写...

转写进度: 1/5
转写进度: 2/5
转写进度: 3/5
转写进度: 4/5
转写进度: 5/5

切片转写完成，结果已合并
已清理临时切片文件
```

## 支持的格式

### 输入格式

- **音频**: MP3, WAV, M4A, AAC, FLAC, OGG
- **视频**: MP4, AVI, MOV, MKV, FLV, WMV, WEBM

### 输出格式

- **TXT**: 纯文本格式（按分段分行，便于阅读）
- **SRT**: 字幕格式（带时间戳）
- **JSON**: 完整结构化数据（包含分段信息）

## 配置文件说明

| 字段 | 说明 | 默认值 |
|------|------|--------|
| `api_base_url` | API 基础 URL | - |
| `api_key` | API 密钥 | - |
| `model` | Whisper 模型名称 | whisper-large-v3 |
| `language` | 语言代码（如 zh, en, ja） | zh |
| `auto_detect` | 是否自动检测语言 | true |
| `output_dir` | 输出目录路径 | ./outputs |
| `max_file_size_mb` | 文件大小阈值（MB），超过则切片 | 10 |
| `silence_threshold` | 静音检测灵敏度 | -30dB |
| `silence_duration` | 静音最小时长（秒） | 0.5 |

### 支持的模型

- `whisper-large-v3`
- `whisper-large-v2`
- `whisper-medium`
- `whisper-small`
- `whisper-base`
- `whisper-tiny`
- `whisper-1`（OpenAI 格式）

## 示例

### 示例 1：转写视频文件

```bash
whisper-go.exe video.mp4
```

输出：
- `video_20240222_153020.txt` - 转写文本
- `video_20240222_153020.srt` - 字幕文件
- `video_20240222_153020.json` - 完整数据

### 示例 2：指定语言和格式

```bash
whisper-go.exe audio.wav --language en --formats txt,json
```

### 示例 3：自动检测语言

```bash
whisper-go.exe mixed.mp4 --auto-detect
```

### 示例 4：处理大文件

```bash
whisper-go.exe large_video.mp4 --verbose
```

工具会自动检测文件大小并进行切片处理，显示详细进度。

## 注意事项

1. 首次使用需要配置 `config.json` 中的 `api_key`
2. 确保系统已安装 ffmpeg 并在 PATH 中
3. 视频文件会自动转换为 WAV 格式（16kHz 单声道）
4. 输出文件名包含时间戳以避免覆盖
5. 大文件切片处理会生成临时文件，转写完成后自动清理

## 许可证

MIT License
