package backend

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/text/unicode/norm"
)

type ConvertAudioRequest struct {
	InputFiles   []string `json:"input_files"`
	OutputFormat string   `json:"output_format"`
	Bitrate      string   `json:"bitrate"`
	Codec        string   `json:"codec"`
}

type ConvertAudioResult struct {
	InputFile  string `json:"input_file"`
	OutputFile string `json:"output_file"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

func ConvertAudio(req ConvertAudioRequest) ([]ConvertAudioResult, error) {
	ffmpegPath, err := GetFFmpegPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get ffmpeg path: %w", err)
	}

	if err := ValidateExecutable(ffmpegPath); err != nil {
		return nil, fmt.Errorf("invalid ffmpeg executable: %w", err)
	}

	installed, err := IsFFmpegInstalled()
	if err != nil || !installed {
		return nil, fmt.Errorf("ffmpeg is not installed")
	}

	results := make([]ConvertAudioResult, len(req.InputFiles))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, inputFile := range req.InputFiles {
		wg.Add(1)
		go func(idx int, inputFile string) {
			defer wg.Done()

			result := ConvertAudioResult{
				InputFile: inputFile,
			}

			inputExt := strings.ToLower(filepath.Ext(inputFile))
			baseName := strings.TrimSuffix(filepath.Base(inputFile), inputExt)
			inputDir := filepath.Dir(inputFile)

			outputFormatUpper := strings.ToUpper(req.OutputFormat)
			outputDir := filepath.Join(inputDir, outputFormatUpper)

			if err := os.MkdirAll(outputDir, 0755); err != nil {
				result.Error = fmt.Sprintf("failed to create output directory: %v", err)
				result.Success = false
				mu.Lock()
				results[idx] = result
				mu.Unlock()
				return
			}

			outputExt := "." + strings.ToLower(req.OutputFormat)
			outputFile := filepath.Join(outputDir, baseName+outputExt)
			outputFile = norm.NFC.String(outputFile)

			if inputExt == outputExt {
				result.Error = "Input and output formats are the same"
				result.Success = false
				mu.Lock()
				results[idx] = result
				mu.Unlock()
				return
			}

			result.OutputFile = outputFile

			var coverArtPath string
			var lyrics string
			var inputMetadata Metadata

			inputMetadata, err = ExtractFullMetadataFromFile(inputFile)
			if err != nil {
				fmt.Printf("[FFmpeg] Warning: Failed to extract metadata from %s: %v\n", inputFile, err)
			}

			inputFile = norm.NFC.String(inputFile)
			coverArtPath, err = ExtractCoverArt(inputFile)
			if err != nil {
				fmt.Printf("[FFmpeg] Warning: Failed to extract cover art from %s: %v\n", inputFile, err)
			}
			lyrics, err = ExtractLyrics(inputFile)
			if err != nil {
				fmt.Printf("[FFmpeg] Warning: Failed to extract lyrics from %s: %v\n", inputFile, err)
			} else if lyrics != "" {
				fmt.Printf("[FFmpeg] Lyrics extracted from %s: %d characters\n", inputFile, len(lyrics))
			} else {
				fmt.Printf("[FFmpeg] No lyrics found in %s\n", inputFile)
			}

			inputMetadata.Lyrics = lyrics

			args := []string{
				"-i", inputFile,
				"-y",
			}

			switch req.OutputFormat {
			case "mp3":
				args = append(args,
					"-codec:a", "libmp3lame",
					"-b:a", req.Bitrate,
					"-map", "0:a",
					"-id3v2_version", "3",
				)
			case "m4a":

				codec := req.Codec
				if codec == "" {
					codec = "aac"
				}

				if codec == "alac" {

					args = append(args,
						"-codec:a", "alac",
						"-map", "0:a",
					)
				} else {

					args = append(args,
						"-codec:a", "aac",
						"-b:a", req.Bitrate,
						"-map", "0:a",
					)
				}
			case "wav", "aiff":
				sampleFmt, rawBits := pcmSampleFormatForInput(inputFile)
				pcmCodec := "pcm_s16le"
				if req.OutputFormat == "aiff" {
					pcmCodec = "pcm_s16be"
				}
				if sampleFmt == "s32" {
					if req.OutputFormat == "aiff" {
						pcmCodec = "pcm_s24be"
					} else {
						pcmCodec = "pcm_s24le"
					}
				}
				args = append(args,
					"-codec:a", pcmCodec,
					"-map", "0:a",
				)
				if rawBits > 0 {
					args = append(args, "-bits_per_raw_sample", strconv.Itoa(rawBits))
				}
			case "opus":
				bitrate := req.Bitrate
				if bitrate == "" {
					bitrate = "192k"
				}
				args = append(args,
					"-codec:a", "libopus",
					"-b:a", bitrate,
					"-map", "0:a",
				)
			}

			args = append(args, outputFile)

			fmt.Printf("[FFmpeg] Converting: %s -> %s\n", inputFile, outputFile)

			cmd := exec.Command(ffmpegPath, args...)

			setHideWindow(cmd)
			output, err := cmd.CombinedOutput()
			if err != nil {
				result.Error = fmt.Sprintf("conversion failed: %s - %s", err.Error(), string(output))
				result.Success = false
				mu.Lock()
				results[idx] = result
				mu.Unlock()

				if coverArtPath != "" {
					os.Remove(coverArtPath)
				}
				return
			}

			if err := EmbedMetadataToConvertedFile(outputFile, inputMetadata, coverArtPath); err != nil {
				fmt.Printf("[FFmpeg] Warning: Failed to embed metadata: %v\n", err)
			} else {
				fmt.Printf("[FFmpeg] Metadata embedded successfully\n")
			}

			if lyrics != "" {
				if err := EmbedLyricsOnlyUniversal(outputFile, lyrics); err != nil {
					fmt.Printf("[FFmpeg] Warning: Failed to embed lyrics: %v\n", err)
				} else {
					fmt.Printf("[FFmpeg] Lyrics embedded successfully\n")
				}
			}

			if coverArtPath != "" {
				os.Remove(coverArtPath)
			}

			result.Success = true
			fmt.Printf("[FFmpeg] Successfully converted: %s\n", outputFile)

			mu.Lock()
			results[idx] = result
			mu.Unlock()
		}(i, inputFile)
	}

	wg.Wait()
	return results, nil
}

func pcmSampleFormatForInput(inputFile string) (sampleFmt string, rawBits int) {
	if meta, err := GetTrackMetadata(inputFile); err == nil && meta != nil && meta.BitsPerSample > 16 {
		return "s32", 24
	}
	return "s16", 0
}

type AudioFileInfo struct {
	Path     string `json:"path"`
	Filename string `json:"filename"`
	Format   string `json:"format"`
	Size     int64  `json:"size"`
}

func GetAudioFileInfo(filePath string) (*AudioFileInfo, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
	return &AudioFileInfo{
		Path:     filePath,
		Filename: filepath.Base(filePath),
		Format:   ext,
		Size:     info.Size(),
	}, nil
}
