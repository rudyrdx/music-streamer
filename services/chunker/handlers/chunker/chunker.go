package chunker

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rudyrdx/music-streamer/chunker/helpers"
)

func ChunkJob(app *pocketbase.PocketBase, segmentSize int64, deleteOriginalFile bool) {
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
		if err := processRecord(app, record, collection, segmentSize, deleteOriginalFile, &chunkIDs, &errors); err != nil {
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

func processRecord(app *pocketbase.PocketBase, record *core.Record, collection *core.Collection, segmentSize int64, deleteOriginalFile bool, chunkIDs *[]string, errors *[]string) error {
	recordID := record.Id
	flacFilePath := record.Get("file_path").(string)

	// Ensure the output directory exists
	outputDir := filepath.Join("output_chunks")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating directory %s: %w", outputDir, err)
	}

	// Open the original file
	file, err := os.Open(flacFilePath)
	if err != nil {
		return fmt.Errorf("error opening file %s: %w", flacFilePath, err)
	}

	// Get the file size
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("error getting file stats: %w", err)
	}
	fileSize := stat.Size()

	// Initialize variables
	var startByte int64 = 0
	var segmentIndex int = 0

	// Read and process chunks
	for startByte < fileSize {
		// Determine the size of the current chunk
		chunkSize := segmentSize
		if startByte+chunkSize > fileSize {
			chunkSize = fileSize - startByte
		}

		// Generate a unique name for the chunk
		chunkFilename := fmt.Sprintf("%s.bin", helpers.GenerateULID())
		chunkFilePath := filepath.Join(outputDir, chunkFilename)

		// Create and write to the chunk file
		chunkFile, err := os.Create(chunkFilePath)
		if err != nil {
			*errors = append(*errors, fmt.Sprintf("Chunk %d failed to create: %v", segmentIndex, err))
			break
		}
		func() {
			defer chunkFile.Close()

			// Copy the chunk data from the original file
			if _, err := file.Seek(startByte, io.SeekStart); err != nil {
				*errors = append(*errors, fmt.Sprintf("Chunk %d failed to seek: %v", segmentIndex, err))
				return
			}

			if _, err := io.CopyN(chunkFile, file, chunkSize); err != nil {
				*errors = append(*errors, fmt.Sprintf("Chunk %d failed to copy: %v", segmentIndex, err))
				return
			}
		}()

		// Save chunk metadata to database
		endByte := startByte + chunkSize - 1
		if err := saveChunkRecord(app, collection, recordID, chunkFilePath, segmentIndex, startByte, endByte, chunkSize, chunkIDs, fileSize); err != nil {
			*errors = append(*errors, fmt.Sprintf("Chunk %d failed to save: %v", segmentIndex, err))
		}
		// Update for the next chunk
		startByte += chunkSize
		segmentIndex++
	}

	// Mark the original record as processed
	record.Set("processed", true)
	if err := app.Save(record); err != nil {
		return fmt.Errorf("error saving record as processed: %w", err)
	}

	// Optionally delete the original file
	if deleteOriginalFile {
		if err := file.Close(); err != nil {
			*errors = append(*errors, fmt.Sprintf("Error closing original file %s: %v", flacFilePath, err))
		}
		if err := os.Remove(flacFilePath); err != nil {
			*errors = append(*errors, fmt.Sprintf("Error deleting original file %s: %v", flacFilePath, err))
		}
	}

	return nil
}

func saveChunkRecord(app *pocketbase.PocketBase, collection *core.Collection, recordID, chunkFilePath string, index int, startByte, endByte, chunkSize int64, chunkIDs *[]string, fileSize int64) error {
	// Create a new database record for this chunk
	chunkRecord := core.NewRecord(collection)
	chunkRecord.Set("file", recordID)
	chunkRecord.Set("chunk_path", chunkFilePath)
	chunkRecord.Set("chunk_order", index+1)
	chunkRecord.Set("start_byte_offset", startByte)
	chunkRecord.Set("end_byte_offset", endByte)
	chunkRecord.Set("chunk_size", chunkSize)
	chunkRecord.Set("file_size", fileSize)

	if err := app.Save(chunkRecord); err != nil {
		return fmt.Errorf("error saving chunk record: %w", err)
	}

	*chunkIDs = append(*chunkIDs, chunkRecord.Id)
	return nil
}
