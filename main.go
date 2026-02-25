package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

// Config 配置结构
type Config struct {
	APIBaseURL       string  `json:"api_base_url"`
	APIKey           string  `json:"api_key"`
	Model            string  `json:"model"`
	Language         string  `json:"language"`
	AutoDetect       bool    `json:"auto_detect"`
	OutputDir        string  `json:"output_dir"`
	MaxFileSizeMB    float64 `json:"max_file_size_mb"`
	SilenceThreshold string  `json:"silence_threshold"`
	SilenceDuration  float64 `json:"silence_duration"`
}

// TranscriptionResult 转写结果
type TranscriptionResult struct {
	Text      string    `json:"text"`
	Language  string    `json:"language"`
	Segments  []Segment `json:"segments,omitempty"`
	Duration  float64   `json:"duration,omitempty"`
}

// Segment 转写分段
type Segment struct {
	ID    int     `json:"id"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

// loadConfig 加载配置文件
func loadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	if config.Model == "" {
		config.Model = "whisper-large-v3"
	}
	if config.Language == "" {
		config.Language = "zh"
	}
	if config.OutputDir == "" {
		config.OutputDir = "./outputs"
	}
	if config.MaxFileSizeMB == 0 {
		config.MaxFileSizeMB = 20
	}
	if config.SilenceThreshold == "" {
		config.SilenceThreshold = "-30dB"
	}
	if config.SilenceDuration == 0 {
		config.SilenceDuration = 0.5
	}

	return &config, nil
}

// isVideoFile 检查是否为视频文件
func isVideoFile(filename string) bool {
	videoExts := []string{".mp4", ".avi", ".mov", ".mkv", ".flv", ".wmv", ".webm", ".m4v"}
	ext := strings.ToLower(filepath.Ext(filename))
	for _, ve := range videoExts {
		if ext == ve {
			return true
		}
	}
	return false
}

// extractAudio 使用 ffmpeg 从视频中提取音频
func extractAudio(videoPath string, verbose bool) (string, error) {
	tempDir := os.TempDir()
	audioPath := filepath.Join(tempDir, fmt.Sprintf("whisper_%d.wav", time.Now().UnixNano()))

	if verbose {
		fmt.Printf("正在提取音频: %s -> %s\n", videoPath, audioPath)
	}

	// 检查 ffmpeg 是否可用
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return "", fmt.Errorf("未找到 ffmpeg，请先安装 ffmpeg")
	}

	// 使用 ffmpeg 提取音频
	// -vn: 不处理视频
	// -acodec pcm_s16le: 使用 PCM 16位编码
	// -ar 16000: 采样率 16kHz
	// -ac 1: 单声道
	cmd := exec.Command("ffmpeg",
		"-i", videoPath,
		"-vn",
		"-acodec", "pcm_s16le",
		"-ar", "16000",
		"-ac", "1",
		"-y",
		audioPath,
	)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg 提取音频失败: %w", err)
	}

	if verbose {
		fmt.Println("音频提取完成")
	}

	return audioPath, nil
}

// transcribeAudio 调用 Whisper API 进行转写
func transcribeAudio(client *openai.Client, audioPath, model, language string, autoDetect bool, verbose bool) (*TranscriptionResult, error) {
	if verbose {
		fmt.Printf("正在转写音频: %s\n", audioPath)
	}

	ctx := context.Background()

	// 打开音频文件
	audioFile, err := os.Open(audioPath)
	if err != nil {
		return nil, fmt.Errorf("打开音频文件失败: %w", err)
	}
	defer audioFile.Close()

	// 构建请求参数
	req := openai.AudioRequest{
		Model:    model,
		FilePath: audioPath,
		Format:   openai.AudioResponseFormatVerboseJSON,
	}

	// 设置语言
	if !autoDetect && language != "" {
		req.Language = language
	}

	// 调用 API
	resp, err := client.CreateTranscription(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("API 调用失败: %w", err)
	}

	if verbose {
		fmt.Println("转写完成")
	}

	// 构建结果
	result := &TranscriptionResult{
		Text:     resp.Text,
		Language: resp.Language,
	}

	// 提取分段信息
	if len(resp.Segments) > 0 {
		for i, seg := range resp.Segments {
			result.Segments = append(result.Segments, Segment{
				ID:    i + 1,
				Start: seg.Start,
				End:   seg.End,
				Text:  seg.Text,
			})
		}
	}

	return result, nil
}

// formatSRTTime 格式化时间戳为 SRT 格式
func formatSRTTime(seconds float64) string {
	hours := int(seconds / 3600)
	minutes := int((seconds - float64(hours)*3600) / 60)
	secs := int(seconds - float64(hours)*3600 - float64(minutes)*60)
	millis := int((seconds - float64(hours)*3600 - float64(minutes)*60 - float64(secs)) * 1000)
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, secs, millis)
}

// saveTXT 保存为 TXT 格式
func saveTXT(result *TranscriptionResult, outputPath string) error {
	var txt strings.Builder

	// 如果有分段信息，按分段输出（每段一行）
	if len(result.Segments) > 0 {
		for _, seg := range result.Segments {
			txt.WriteString(seg.Text)
			txt.WriteString("\n")
		}
	} else {
		// 没有分段信息，直接输出原文
		txt.WriteString(result.Text)
	}

	return os.WriteFile(outputPath, []byte(txt.String()), 0644)
}

// saveSRT 保存为 SRT 格式
func saveSRT(result *TranscriptionResult, outputPath string) error {
	var srt strings.Builder
	for _, seg := range result.Segments {
		srt.WriteString(fmt.Sprintf("%d\n", seg.ID))
		srt.WriteString(fmt.Sprintf("%s --> %s\n", formatSRTTime(seg.Start), formatSRTTime(seg.End)))
		srt.WriteString(fmt.Sprintf("%s\n\n", seg.Text))
	}
	return os.WriteFile(outputPath, []byte(srt.String()), 0644)
}

// saveJSON 保存为 JSON 格式
func saveJSON(result *TranscriptionResult, outputPath string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0644)
}

// generateOutputPath 生成输出文件名
func generateOutputPath(inputPath, outputDir, ext string) string {
	filename := filepath.Base(inputPath)
	nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))
	timestamp := time.Now().Format("20060102_150405")
	outputFilename := fmt.Sprintf("%s_%s.%s", nameWithoutExt, timestamp, ext)
	return filepath.Join(outputDir, outputFilename)
}

// getFileSizeMB 获取文件大小（MB）
func getFileSizeMB(filePath string) (float64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	return float64(info.Size()) / (1024 * 1024), nil
}

// SilencePoint 静音点
type SilencePoint struct {
	Start float64
	End   float64
}

// detectSilence 使用 ffmpeg 检测静音点
func detectSilence(audioPath, threshold string, minDuration float64, verbose bool) ([]SilencePoint, error) {
	if verbose {
		fmt.Printf("正在检测静音点: %s\n", audioPath)
	}

	// 使用 ffmpeg silencedetect 滤镜检测静音
	cmd := exec.Command("ffmpeg",
		"-i", audioPath,
		"-af", fmt.Sprintf("silencedetect=noise=%s:d=%.2f", threshold, minDuration),
		"-f", "null",
		"-",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("静音检测失败: %w", err)
	}

	// 解析静音点
	var points []SilencePoint
	lines := strings.Split(string(output), "\n")

	var currentStart float64
	for _, line := range lines {
		if strings.Contains(line, "silence_start:") {
			// 解析静音开始时间
			parts := strings.Split(line, "silence_start:")
			if len(parts) > 1 {
				if start, err := parseSilenceTime(strings.TrimSpace(parts[1])); err == nil {
					currentStart = start
				}
			}
		} else if strings.Contains(line, "silence_end:") {
			// 解析静音结束时间
			parts := strings.Split(line, "silence_end:")
			if len(parts) > 1 {
				if end, err := parseSilenceTime(strings.TrimSpace(parts[1])); err == nil {
					points = append(points, SilencePoint{
						Start: currentStart,
						End:   end,
					})
				}
			}
		}
	}

	if verbose {
		fmt.Printf("检测到 %d 个静音点\n", len(points))
	}

	return points, nil
}

// parseSilenceTime 解析静音时间
func parseSilenceTime(s string) (float64, error) {
	// 格式可能是 "123.45" 或 "123.45 | ..."
	parts := strings.Split(s, "|")
	s = strings.TrimSpace(parts[0])
	var t float64
	_, err := fmt.Sscanf(s, "%f", &t)
	return t, err
}

// getAudioDuration 获取音频时长
func getAudioDuration(audioPath string) (float64, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		audioPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("获取音频时长失败: %w", err)
	}

	var duration float64
	_, err = fmt.Sscanf(strings.TrimSpace(string(output)), "%f", &duration)
	return duration, err
}

// AudioChunk 音频切片信息
type AudioChunk struct {
	Path        string
	StartOffset float64 // 切片在原始音频中的起始时间
}

// splitAudioBySilence 按静音点分割音频
func splitAudioBySilence(audioPath string, maxSizeMB float64, threshold string, minDuration float64, verbose bool) ([]AudioChunk, error) {
	// 获取文件大小
	sizeMB, err := getFileSizeMB(audioPath)
	if err != nil {
		return nil, fmt.Errorf("获取文件大小失败: %w", err)
	}

	// 获取音频时长
	duration, err := getAudioDuration(audioPath)
	if err != nil {
		return nil, fmt.Errorf("获取音频时长失败: %w", err)
	}

	if verbose {
		fmt.Printf("音频时长: %.2f 秒, 文件大小: %.2f MB\n", duration, sizeMB)
	}

	// 计算需要分割成多少片
	numChunks := int(sizeMB/maxSizeMB) + 1
	// 每片的理想时长
	idealChunkDuration := duration / float64(numChunks)

	if verbose {
		fmt.Printf("计划分割为 %d 片，每片约 %.2f 秒\n", numChunks, idealChunkDuration)
	}

	// 检测静音点
	silencePoints, err := detectSilence(audioPath, threshold, minDuration, verbose)
	if err != nil {
		return nil, err
	}

	// 计算切片位置（优先在静音点分割）
	splitTimes := calculateSplitTimes(duration, idealChunkDuration, silencePoints)

	if verbose {
		fmt.Printf("切片时间点: %v\n", splitTimes)
	}

	// 执行切片
	return createAudioChunks(audioPath, splitTimes, verbose)
}

// calculateSplitTimes 计算切片时间点
func calculateSplitTimes(totalDuration, idealChunkDuration float64, silencePoints []SilencePoint) []float64 {
	var splitTimes []float64
	currentTime := idealChunkDuration

	for currentTime < totalDuration {
		// 寻找最接近当前目标时间的静音点
		bestTime := currentTime
		minDiff := idealChunkDuration // 初始化为理想时长

		for _, sp := range silencePoints {
			// 静音结束点是好的分割点
			diff := sp.End - currentTime
			if diff < 0 {
				diff = -diff
			}

			// 如果静音点在合理范围内（理想时间的 50% 到 150%）
			if sp.End > currentTime*0.5 && sp.End < currentTime*1.5 && diff < minDiff {
				minDiff = diff
				bestTime = sp.End
			}
		}

		// 如果没有找到合适的静音点，使用当前时间
		if bestTime >= totalDuration {
			break
		}

		splitTimes = append(splitTimes, bestTime)
		currentTime = bestTime + idealChunkDuration
	}

	return splitTimes
}

// createAudioChunks 创建音频切片文件
func createAudioChunks(audioPath string, splitTimes []float64, verbose bool) ([]AudioChunk, error) {
	tempDir := os.TempDir()
	var chunks []AudioChunk

	// 获取音频时长
	duration, _ := getAudioDuration(audioPath)

	// 创建切片
	startTime := 0.0
	for i, endTime := range splitTimes {
		chunkPath := filepath.Join(tempDir, fmt.Sprintf("whisper_chunk_%d_%d.wav", time.Now().UnixNano(), i))

		if verbose {
			fmt.Printf("创建切片 %d: %.2f - %.2f 秒\n", i+1, startTime, endTime)
		}

		// 使用 ffmpeg 提取片段
		cmd := exec.Command("ffmpeg",
			"-i", audioPath,
			"-ss", fmt.Sprintf("%.3f", startTime),
			"-to", fmt.Sprintf("%.3f", endTime),
			"-acodec", "pcm_s16le",
			"-ar", "16000",
			"-ac", "1",
			"-y",
			chunkPath,
		)

		if err := cmd.Run(); err != nil {
			// 清理已创建的切片
			for _, c := range chunks {
				os.Remove(c.Path)
			}
			return nil, fmt.Errorf("创建切片失败: %w", err)
		}

		chunks = append(chunks, AudioChunk{
			Path:        chunkPath,
			StartOffset: startTime,
		})

		startTime = endTime
	}

	// 最后一个切片
	if startTime < duration {
		chunkPath := filepath.Join(tempDir, fmt.Sprintf("whisper_chunk_%d_%d.wav", time.Now().UnixNano(), len(splitTimes)))

		if verbose {
			fmt.Printf("创建切片 %d: %.2f - %.2f 秒\n", len(splitTimes)+1, startTime, duration)
		}

		cmd := exec.Command("ffmpeg",
			"-i", audioPath,
			"-ss", fmt.Sprintf("%.3f", startTime),
			"-acodec", "pcm_s16le",
			"-ar", "16000",
			"-ac", "1",
			"-y",
			chunkPath,
		)

		if err := cmd.Run(); err != nil {
			for _, c := range chunks {
				os.Remove(c.Path)
			}
			return nil, fmt.Errorf("创建最后切片失败: %w", err)
		}

		chunks = append(chunks, AudioChunk{
			Path:        chunkPath,
			StartOffset: startTime,
		})
	}

	return chunks, nil
}

// transcribeMultipleChunks 转写多个切片
func transcribeMultipleChunks(client *openai.Client, chunks []AudioChunk, model, language string, autoDetect, verbose bool) ([]*TranscriptionResult, error) {
	results := make([]*TranscriptionResult, len(chunks))

	for i, chunk := range chunks {
		if verbose {
			fmt.Printf("\n转写进度: %d/%d\n", i+1, len(chunks))
		}

		result, err := transcribeAudio(client, chunk.Path, model, language, autoDetect, verbose)
		if err != nil {
			return nil, fmt.Errorf("切片 %d 转写失败: %w", i+1, err)
		}

		results[i] = result
	}

	return results, nil
}

// mergeResults 合并多个转写结果并修正时间戳
func mergeResults(results []*TranscriptionResult, chunks []AudioChunk) *TranscriptionResult {
	merged := &TranscriptionResult{
		Language: "",
		Segments: []Segment{},
	}

	segmentID := 1
	var totalText strings.Builder

	for i, result := range results {
		// 设置语言（取第一个非空的）
		if merged.Language == "" && result.Language != "" {
			merged.Language = result.Language
		}

		// 合并文本
		if result.Text != "" {
			totalText.WriteString(result.Text)
			if !strings.HasSuffix(result.Text, "\n") {
				totalText.WriteString("\n")
			}
		}

		// 修正并合并分段
		offset := chunks[i].StartOffset
		for _, seg := range result.Segments {
			merged.Segments = append(merged.Segments, Segment{
				ID:    segmentID,
				Start: seg.Start + offset,
				End:   seg.End + offset,
				Text:  seg.Text,
			})
			segmentID++
		}

		// 如果没有分段信息，但有时间偏移，需要记录
		if len(result.Segments) == 0 && i > 0 {
			// 创建一个分段来标记时间偏移
			merged.Segments = append(merged.Segments, Segment{
				ID:    segmentID,
				Start: offset,
				End:   offset + 10, // 假设每段至少10秒
				Text:  result.Text,
			})
			segmentID++
		}
	}

	merged.Text = totalText.String()
	if len(merged.Segments) > 0 {
		merged.Duration = merged.Segments[len(merged.Segments)-1].End
	}

	return merged
}

// cleanupChunks 清理临时切片文件
func cleanupChunks(chunks []AudioChunk) {
	for _, chunk := range chunks {
		os.Remove(chunk.Path)
	}
}

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "./config.json", "配置文件路径")
	language := flag.String("language", "", "语言代码（如 zh, en, ja）")
	autoDetect := flag.Bool("auto-detect", false, "自动检测语言")
	model := flag.String("model", "", "Whisper 模型名称")
	outputDir := flag.String("output", "", "输出目录")
	formats := flag.String("formats", "txt,srt,json", "输出格式（逗号分隔）")
	verbose := flag.Bool("verbose", false, "显示详细输出")
	flag.Parse()

	// 检查输入文件
	if flag.NArg() < 1 {
		fmt.Println("用法: whisper-go <input-file> [options]")
		fmt.Println("选项:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	inputFile := flag.Arg(0)

	// 检查输入文件是否存在
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		log.Fatalf("输入文件不存在: %s", inputFile)
	}

	// 加载配置文件
	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 检查 API Key
	if config.APIKey == "" {
		log.Fatal("配置文件中未设置 API Key，请先在 config.json 中配置 api_key")
	}

	// 覆盖配置
	if *language != "" {
		config.Language = *language
	}
	if *model != "" {
		config.Model = *model
	}
	if *outputDir != "" {
		config.OutputDir = *outputDir
	}
	if *autoDetect {
		config.AutoDetect = true
	}

	// 解析输出格式
	formatList := strings.Split(*formats, ",")
	for i, f := range formatList {
		formatList[i] = strings.TrimSpace(strings.ToLower(f))
	}

	// 创建输出目录
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		log.Fatalf("创建输出目录失败: %v", err)
	}

	// 处理输入文件
	var audioPath string
	var cleanupAudio bool

	if isVideoFile(inputFile) {
		if *verbose {
			fmt.Printf("检测到视频文件: %s\n", inputFile)
		}

		// 提取音频
		audioPath, err = extractAudio(inputFile, *verbose)
		if err != nil {
			log.Fatalf("提取音频失败: %v", err)
		}
		cleanupAudio = true
	} else {
		audioPath = inputFile
		cleanupAudio = false
	}

	// 清理临时文件
	defer func() {
		if cleanupAudio && audioPath != "" {
			os.Remove(audioPath)
			if *verbose {
				fmt.Println("已清理临时音频文件")
			}
		}
	}()

	// 创建 OpenAI 客户端
	defaultConfig := openai.DefaultConfig(config.APIKey)
	defaultConfig.BaseURL = config.APIBaseURL
	client := openai.NewClientWithConfig(defaultConfig)

	if *verbose {
		fmt.Printf("API 配置:\n")
		fmt.Printf("  Base URL: %s\n", config.APIBaseURL)
		fmt.Printf("  Model: %s\n", config.Model)
		fmt.Printf("  Language: %s (Auto-detect: %v)\n", config.Language, config.AutoDetect)
		fmt.Printf("  Output Directory: %s\n", config.OutputDir)
		fmt.Printf("  Output Formats: %s\n", config.MaxFileSizeMB)
		fmt.Printf("  Max File Size: %.0f MB\n\n", config.MaxFileSizeMB)
	}

	// 检查文件大小，决定是否需要切片
	fileSizeMB, err := getFileSizeMB(audioPath)
	if err != nil {
		log.Fatalf("获取文件大小失败: %v", err)
	}

	var result *TranscriptionResult

	if fileSizeMB > config.MaxFileSizeMB {
		if *verbose {
			fmt.Printf("文件大小 %.2f MB 超过阈值 %.0f MB，将进行切片处理\n", fileSizeMB, config.MaxFileSizeMB)
		}

		// 切片处理
		chunks, err := splitAudioBySilence(audioPath, config.MaxFileSizeMB, config.SilenceThreshold, config.SilenceDuration, *verbose)
		if err != nil {
			log.Fatalf("音频切片失败: %v", err)
		}

		// 确保清理切片文件
		defer cleanupChunks(chunks)

		if *verbose {
			fmt.Printf("\n共创建 %d 个切片，开始转写...\n", len(chunks))
		}

		// 转写所有切片
		results, err := transcribeMultipleChunks(client, chunks, config.Model, config.Language, config.AutoDetect, *verbose)
		if err != nil {
			log.Fatalf("切片转写失败: %v", err)
		}

		// 合并结果
		result = mergeResults(results, chunks)

		if *verbose {
			fmt.Println("\n切片转写完成，结果已合并")
		}
	} else {
		// 文件大小正常，直接转写
		if *verbose {
			fmt.Printf("文件大小 %.2f MB，直接转写\n", fileSizeMB)
		}

		result, err = transcribeAudio(client, audioPath, config.Model, config.Language, config.AutoDetect, *verbose)
		if err != nil {
			log.Fatalf("转写失败: %v", err)
		}
	}

	// 保存结果
	outputFiles := []string{}
	for _, format := range formatList {
		var outputPath string

		switch format {
		case "txt":
			outputPath = generateOutputPath(inputFile, config.OutputDir, "txt")
			if err := saveTXT(result, outputPath); err != nil {
				log.Printf("保存 TXT 失败: %v", err)
				continue
			}
		case "srt":
			if len(result.Segments) == 0 {
				log.Println("警告: 没有分段信息，跳过 SRT 格式")
				continue
			}
			outputPath = generateOutputPath(inputFile, config.OutputDir, "srt")
			if err := saveSRT(result, outputPath); err != nil {
				log.Printf("保存 SRT 失败: %v", err)
				continue
			}
		case "json":
			outputPath = generateOutputPath(inputFile, config.OutputDir, "json")
			if err := saveJSON(result, outputPath); err != nil {
				log.Printf("保存 JSON 失败: %v", err)
				continue
			}
		default:
			log.Printf("不支持的格式: %s", format)
			continue
		}

		if outputPath != "" {
			outputFiles = append(outputFiles, outputPath)
			if *verbose {
				fmt.Printf("已保存: %s\n", outputPath)
			}
		}
	}

	// 输出摘要
	fmt.Println("\n=== 转写完成 ===")
	fmt.Printf("语言: %s\n", result.Language)
	fmt.Printf("文本长度: %d 字符\n", len(result.Text))
	fmt.Printf("分段数: %d\n", len(result.Segments))
	fmt.Printf("\n输出文件:\n")
	for _, file := range outputFiles {
		fmt.Printf("  - %s\n", file)
	}

	if *verbose {
		fmt.Printf("\n转写文本预览:\n%s\n", result.Text)
	}
}