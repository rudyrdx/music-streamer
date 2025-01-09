package chunker

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func ChunkJob(app *pocketbase.PocketBase) {
	startTime := time.Now()

	records, err := app.FindRecordsByFilter(
		"UploadedFiles",
		"processed = False",
		"-created",
		4,
		0,
	)
	if err != nil || len(records) == 0 {
		return
	}

	totalRecords := len(records)
	errors := make([]string, 0)
	chunkIDs := make([]string, 0)

	collection, err := app.FindCollectionByNameOrId("ChunkedFiles")
	if err != nil {
		errors = append(errors, fmt.Sprintf(
			"error finding collection ChunkedFiles: %v",
			err,
		))
	}

	for _, record := range records {
		recordID := record.Id
		flacFilePath := record.Get("file_path").(string)

		// Create a directory for the chunks if needed
		oldDir := strings.Split(flacFilePath, "/")[1]
		if _, err := os.Stat(oldDir); os.IsNotExist(err) {
			if mkErr := os.MkdirAll(oldDir, 0755); mkErr != nil {
				errors = append(errors, fmt.Sprintf(
					"%s error creating directory %s: %v",
					recordID, oldDir, mkErr,
				))
				continue
			}
		}

		// 1) Read total duration from FLAC
		durationInSeconds, err := GetFlacDuration(flacFilePath)
		if err != nil {
			errors = append(errors,
				fmt.Sprintf("%s error reading FLAC duration: %v",
					recordID, err,
				),
			)
			continue
		}

		// 2) Decide how long each chunk should be in seconds
		//    (Customize this value or accept it as a parameter.)
		segmentTime := 30.0 // e.g., 30 seconds

		// 3) Use ffmpeg to segment by time, preserving FLAC data (no re‐encode).
		//    -c copy means no quality loss.
		//    -segment_time controls chunk length in seconds.
		chunkPattern := filepath.Join(oldDir, "chunk_%03d.flac")

		ffmpegCmd := exec.Command(
			"ffmpeg",
			"-i", flacFilePath,
			"-c", "copy",
			"-f", "segment",
			"-segment_time", fmt.Sprintf("%.2f", segmentTime),
			"-reset_timestamps", "1",
			chunkPattern,
		)
		if runErr := ffmpegCmd.Run(); runErr != nil {
			errors = append(errors, fmt.Sprintf(
				"%s error chunking via ffmpeg: %v",
				recordID, runErr,
			))
			continue
		}

		// 4) Gather the generated chunk files from oldDir
		files, err := filepath.Glob(filepath.Join(oldDir, "chunk_*.flac"))
		if err != nil {
			errors = append(errors, fmt.Sprintf(
				"%s error globbing chunk files: %v",
				recordID, err,
			))
			continue
		}

		// 5) For each chunk file, figure out its theoretical start/end times
		//    based on the chunk index, then store them in your DB.
		//    “start_byte_offset” -> chunk start time
		//    “end_byte_offset”   -> chunk end time
		//    “chunk_size”        -> chunk duration
		chunkOrder := 1
		totalChunks := len(files)

		for i, chunkFilePath := range files {
			// Theoretical start time in seconds
			startSec := float64(i) * segmentTime

			// Theoretical end time (may be less than segmentTime for the last chunk)
			endSec := float64(i+1) * segmentTime
			if endSec > durationInSeconds {
				endSec = durationInSeconds
			}

			chunkDuration := endSec - startSec

			r := core.NewRecord(collection)
			r.Set("file", record.Id)
			r.Set("chunk_path", chunkFilePath)
			r.Set("chunk_order", chunkOrder)
			// Save times in “start_byte_offset” and “end_byte_offset”
			r.Set("start_byte_offset", startSec)
			r.Set("end_byte_offset", endSec)

			// Save the chunk’s duration under “chunk_size”
			r.Set("chunk_size", chunkDuration)

			if err := app.Save(r); err != nil {
				errors = append(errors, fmt.Sprintf(
					"%s error saving chunk %d: %v",
					recordID, chunkOrder, err,
				))
				continue
			}
			chunkOrder++
			chunkIDs = append(chunkIDs, r.Id)
		}

		// 6) Mark record processed
		record.Set("processed", true)
		if err := app.Save(record); err != nil {
			errors = append(errors, fmt.Sprintf(
				"%s error saving record: %v",
				recordID, err,
			))
			continue
		}

		// 7) (Optional) Remove original file if all is good
		if err := os.Remove(flacFilePath); err != nil {
			errors = append(errors, fmt.Sprintf(
				"%s error removing file: %v",
				recordID, err,
			))
			continue
		}
	}

	endTime := time.Since(startTime).Nanoseconds()
	if len(errors) > 0 {
		app.Logger().Error(
			"ChunkJob",
			"totalRecords", totalRecords,
			"execTime(ms)", float64(endTime)/1e6,
			"chunkIds", chunkIDs,
			"errors", errors,
		)
	} else {
		app.Logger().Info(
			"ChunkJob",
			"totalRecords", totalRecords,
			"execTime(ms)", float64(endTime)/1e6,
			"chunkIds", chunkIDs,
		)
	}
}
func FindFramePositions(data []byte) []int {
	var framePositions []int
	dataLen := len(data)

	for i := 0; i < dataLen-1; {
		word := (uint16(data[i]) << 8) | uint16(data[i+1])
		sync := (word >> 2) & 0x3FFF // Get top 14 bits
		if sync == 0x3FFE {
			// Found a frame sync code
			framePositions = append(framePositions, i)
			i += 2 // Move past the sync code
		} else {
			i++
		}
	}

	return framePositions
}

func ReadFLACHeaders(flacFilePath string) ([]byte, error) {
	file, err := os.Open(flacFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Read the "fLaC" marker
	signature := make([]byte, 4)
	_, err = file.Read(signature)
	if err != nil || string(signature) != "fLaC" {
		return nil, fmt.Errorf("not a valid FLAC file")
	}

	headers := signature
	isLastMetadataBlock := false

	// Read metadata blocks
	for !isLastMetadataBlock {
		headerByte := make([]byte, 1)
		_, err = file.Read(headerByte)
		if err != nil {
			return nil, fmt.Errorf("unexpected end of file while reading metadata headers")
		}

		headerInt := headerByte[0]
		isLastMetadataBlock = (headerInt & 0x80) != 0 // Check if the last bit is set
		lengthBytes := make([]byte, 3)
		_, err = file.Read(lengthBytes)
		if err != nil {
			return nil, fmt.Errorf("unexpected end of file while reading metadata block length")
		}

		blockLength := int(binary.BigEndian.Uint32(append([]byte{0}, lengthBytes...)))

		// Correctly handle cases where blockLength is larger than the remaining file size
		blockData := make([]byte, blockLength)
		bytesRead, err := file.Read(blockData)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("error reading metadata block data: %v", err)
		}

		if bytesRead < blockLength {
			// If we read fewer bytes than blockLength, adjust blockData accordingly
			blockData = blockData[:bytesRead]
		}

		headers = append(headers, append(headerByte, append(lengthBytes, blockData...)...)...)

		if err == io.EOF {
			break // Exit loop if we've reached the end of the file
		}
	}

	return headers, nil
}

// GetFlacDuration returns duration (in seconds) for a file using ffprobe.
func GetFlacDuration(filePath string) (float64, error) {
	cmd := exec.Command("ffprobe",
	"-v", "error",
	"-show_entries", "format=duration,size",
	"-of", "json",
	filePath,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
	return 0, err
}
