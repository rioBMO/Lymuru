package backend

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// ResampleRequest defines a batch resampling operation.
type ResampleRequest struct {
	InputFiles []string `json:"input_files"`
	SampleRate string   `json:"sample_rate"` // e.g. "44100", "48000", "96000"
	BitDepth   string   `json:"bit_depth"`   // e.g. "16", "24"
}

// ResampleResult holds the outcome for a single input file.
type ResampleResult struct {
	InputFile  string `json:"input_file"`
	OutputFile string `json:"output_file"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// ResampleAudio converts one or more audio files to the requested sample rate / bit depth.
// Outputs are written to a subfolder named after the target format (e.g. "44.1kHz 16bit").
func ResampleAudio(req ResampleRequest) ([]ResampleResult, error) {
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

	if req.SampleRate == "" && req.BitDepth == "" {
		return nil, fmt.Errorf("at least one of sample rate or bit depth must be specified")
	}

	results := make([]ResampleResult, len(req.InputFiles))
	var wg sync.WaitGroup
	var mu sync.Mutex

	folderLabel := buildFolderLabel(req.SampleRate, req.BitDepth)

	for i, inputFile := range req.InputFiles {
		wg.Add(1)
		go func(idx int, inputFile string) {
			defer wg.Done()

			result := ResampleResult{InputFile: inputFile}

			inputExt := strings.ToLower(filepath.Ext(inputFile))
			baseName := strings.TrimSuffix(filepath.Base(inputFile), inputExt)
			inputDir := filepath.Dir(inputFile)

			outputDir := filepath.Join(inputDir, folderLabel)
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				result.Error = fmt.Sprintf("failed to create output directory: %v", err)
				result.Success = false
				mu.Lock()
				results[idx] = result
				mu.Unlock()
				return
			}

			outputFile := filepath.Join(outputDir, baseName+".flac")
			result.OutputFile = outputFile

			args := []string{"-i", inputFile, "-y"}

			if req.BitDepth != "" {
				switch req.BitDepth {
				case "16":
					args = append(args, "-c:a", "flac", "-sample_fmt", "s16")
				case "24":
					args = append(args, "-c:a", "flac", "-sample_fmt", "s32", "-bits_per_raw_sample", "24")
				default:
					args = append(args, "-c:a", "flac")
				}
			} else {
				args = append(args, "-c:a", "flac")
			}

			if req.SampleRate != "" {
				args = append(args, "-ar", req.SampleRate)
			}

			args = append(args, "-map_metadata", "0")
			args = append(args, outputFile)

			LogInfo("[Resample] %s -> %s", inputFile, outputFile)

			cmd := exec.Command(ffmpegPath, args...)
			setHideWindow(cmd)
			output, err := cmd.CombinedOutput()
			if err != nil {
				result.Error = fmt.Sprintf("resampling failed: %s - %s", err.Error(), string(output))
				result.Success = false
				mu.Lock()
				results[idx] = result
				mu.Unlock()
				return
			}

			result.Success = true
			LogInfo("[Resample] Done: %s", outputFile)
			mu.Lock()
			results[idx] = result
			mu.Unlock()
		}(i, inputFile)
	}

	wg.Wait()
	return results, nil
}

// buildFolderLabel creates a human-readable folder name from the target format.
func buildFolderLabel(sampleRate, bitDepth string) string {
	var parts []string

	if bitDepth != "" {
		parts = append(parts, bitDepth+"bit")
	}

	switch sampleRate {
	case "44100":
		parts = append(parts, "44.1kHz")
	case "48000":
		parts = append(parts, "48kHz")
	case "96000":
		parts = append(parts, "96kHz")
	case "192000":
		parts = append(parts, "192kHz")
	default:
		if sampleRate != "" {
			parts = append(parts, sampleRate+"Hz")
		}
	}

	if len(parts) == 0 {
		return "Resampled"
	}
	return strings.Join(parts, " ")
}
