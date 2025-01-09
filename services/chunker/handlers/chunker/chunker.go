package chunker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func ChunkJob(app *pocketbase.PocketBase, segmentTime float64, deleteOriginalFile bool) {
	startTime := time.Now()

	// Fetch records to process
	records, err := app.FindRecordsByFilter("UploadedFiles", "processed = False", "-created", 4, 0)
	if err != nil || len(records) == 0 {
		app.Logger().Info("ChunkJob", "message", "No records to process", "error", err)
		return
	}

	totalRecords := len(records)
	errors := []string{}
	chunkIDs := []string{}

	// Get chunked files collection
	collection, err := app.FindCollectionByNameOrId("ChunkedFiles")
	if err != nil {
		app.Logger().Error("ChunkJob", "message", "Failed to find ChunkedFiles collection", "error", err)
		return
	}

	// Process each record
	for _, record := range records {
		if err := processRecord(app, record, collection, segmentTime, deleteOriginalFile, &chunkIDs, &errors); err != nil {
			errors = append(errors, fmt.Sprintf("Record %s failed: %v", record.Id, err))
		}
	}

	// Log execution summary
	execTimeMs := float64(time.Since(startTime).Nanoseconds()) / 1e6
	if len(errors) > 0 {
		app.Logger().Error(
			"ChunkJob",
			"totalRecords", totalRecords,
			"execTime(ms)", execTimeMs,
			"chunkIds", chunkIDs,
			"errors", errors,
		)
	} else {
		app.Logger().Info(
			"ChunkJob",
			"totalRecords", totalRecords,
			"execTime(ms)", execTimeMs,
			"chunkIds", chunkIDs,
		)
	}
}
func processRecord(app *pocketbase.PocketBase, record *core.Record, collection *core.Collection, segmentTime float64, deleteOriginalFile bool, chunkIDs *[]string, errors *[]string) error {
	recordID := record.Id
	flacFilePath := record.Get("file_path").(string)

	// Ensure the output directory exists
	outputDir := filepath.Dir(flacFilePath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating directory %s: %w", outputDir, err)
	}

	// 1. Get the FLAC file's duration
	durationInSeconds, err := GetFlacDuration(flacFilePath)
	if err != nil {
		return fmt.Errorf("error reading FLAC duration: %w", err)
	}

	// 2. Segment the file using FFmpeg
	chunkPattern := filepath.Join(outputDir, "chunk_%03d.flac")
	ffmpegCmd := exec.Command("ffmpeg", "-i", flacFilePath, "-c", "copy", "-f", "segment", "-segment_time", fmt.Sprintf("%.2f", segmentTime), "-reset_timestamps", "1", chunkPattern)
	if err := ffmpegCmd.Run(); err != nil {
		return fmt.Errorf("error chunking via ffmpeg: %w", err)
	}

	// 3. Process each generated chunk
	chunkFiles, err := filepath.Glob(filepath.Join(outputDir, "chunk_*.flac"))
	if err != nil {
		return fmt.Errorf("error globbing chunk files: %w", err)
	}

	if len(chunkFiles) == 0 {
		return fmt.Errorf("no chunks generated for file %s", flacFilePath)
	}

	for i, chunkFilePath := range chunkFiles {
		if err := saveChunkRecord(app, collection, recordID, chunkFilePath, i, segmentTime, durationInSeconds, chunkIDs); err != nil {
			*errors = append(*errors, fmt.Sprintf("Chunk %d failed to save: %v", i+1, err))
		}
	}

	// 4. Mark the original record as processed
	record.Set("processed", true)
	if err := app.Save(record); err != nil {
		return fmt.Errorf("error saving record as processed: %w", err)
	}

	// 5. Optionally delete the original file
	if deleteOriginalFile {
		if err := os.Remove(flacFilePath); err != nil {
			*errors = append(*errors, fmt.Sprintf("Error deleting original file %s: %v", flacFilePath, err))
		}
	}

	return nil
}

func saveChunkRecord(app *pocketbase.PocketBase, collection *core.Collection, recordID, chunkFilePath string, index int, segmentTime, durationInSeconds float64, chunkIDs *[]string) error {
	// Calculate chunk start and end times
	startSec := float64(index) * segmentTime
	endSec := startSec + segmentTime
	if endSec > durationInSeconds {
		endSec = durationInSeconds
	}
	chunkDuration := endSec - startSec

	// Create a new database record for this chunk
	chunkRecord := core.NewRecord(collection)
	chunkRecord.Set("file", recordID)
	chunkRecord.Set("chunk_path", chunkFilePath)
	chunkRecord.Set("chunk_order", index+1)
	chunkRecord.Set("start_byte_offset", startSec)
	chunkRecord.Set("end_byte_offset", endSec)
	chunkRecord.Set("chunk_size", chunkDuration)

	if err := app.Save(chunkRecord); err != nil {
		return fmt.Errorf("error saving chunk record: %w", err)
	}

	*chunkIDs = append(*chunkIDs, chunkRecord.Id)
	return nil
}

// GetFlacDuration returns the duration (in seconds) of a file using `ffprobe`.
func GetFlacDuration(filePath string) (float64, error) {
	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "json",
		filePath,
	)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("error running ffprobe: %w", err)
	}

	var ffprobeOutput struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out.Bytes(), &ffprobeOutput); err != nil {
		return 0, fmt.Errorf("error parsing ffprobe output: %w", err)
	}

	duration, err := strconv.ParseFloat(ffprobeOutput.Format.Duration, 64)
	if err != nil {
		return 0, fmt.Errorf("error converting duration to float: %w", err)
	}
	return duration, nil
}
