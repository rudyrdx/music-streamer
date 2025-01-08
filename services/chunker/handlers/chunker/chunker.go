package chunker

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rudyrdx/music-streamer/chunker/helpers"
)

func ChunkJob(app *pocketbase.PocketBase) {

	start_time := time.Now()

	records, err := app.FindRecordsByFilter(
		"UploadedFiles",     // collection
		"processed = False", // filter
		"-created",          // sort
		4,                   // limit
		0,                   // offset
	)
	if err != nil {
		return
	}

	if len(records) < 1 {
		return
	}
	total_records := len(records)
	errors := make([]string, 0)
	chunkIds := make([]string, 0)
	collection, err := app.FindCollectionByNameOrId("ChunkedFiles")
	if err != nil {
		errors = append(errors, fmt.Sprintf("error finding collection ChunkedFiles: %v", err))
	}

	for _, record := range records {
		// Constants
		record_id := record.Id
		flacFilePath := record.Get("file_path").(string)
		const chunkSize = int(1 * 1024 * 1024) // 1 MB in bytes (as float64)
		oldDir := strings.Split(flacFilePath, "/")[1]

		if _, err := os.Stat(oldDir); os.IsNotExist(err) {
			err = os.MkdirAll(oldDir, 0755)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s error creating directory %s: %v", record_id, oldDir, err))
				continue
			}
		}

		headers, err := ReadFLACHeaders(flacFilePath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s error reading FLAC headers: %v", record_id, err))
			continue
		}

		file, err := os.Open(flacFilePath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s error opening file: %v", record_id, err))
			continue
		}

		// Skip the headers
		_, err = file.Seek(int64(len(headers)), io.SeekStart)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s error seeking past headers: %v", record_id, err))
			continue
		}

		audioData, err := io.ReadAll(file)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s error reading audio data: %v", record_id, err))
			continue
		}

		framePositions := FindFramePositions(audioData)
		if len(framePositions) == 0 {
			errors = append(errors, fmt.Sprintf("%s no frame positions found", record_id))
		}
		fmt.Printf("Found %d frame positions.\n", len(framePositions))

		currentFrameIndex := 0
		chunkOrder := 1
		for currentFrameIndex < len(framePositions) {
			r := core.NewRecord(collection)
			accumulatedSize := 0
			startFrameIndex := currentFrameIndex

			for accumulatedSize < chunkSize && currentFrameIndex < len(framePositions)-1 {
				frameStart := framePositions[currentFrameIndex]
				frameEnd := framePositions[currentFrameIndex+1]
				frameLength := frameEnd - frameStart
				accumulatedSize += frameLength
				currentFrameIndex++
			}

			// Handle the last frame
			if currentFrameIndex == len(framePositions)-1 {
				frameStart := framePositions[currentFrameIndex]
				frameEnd := len(audioData)
				frameLength := frameEnd - frameStart
				accumulatedSize += frameLength
				currentFrameIndex++
			}

			// Extract frames
			dataStart := framePositions[startFrameIndex]
			var dataEnd int
			if currentFrameIndex < len(framePositions) {
				dataEnd = framePositions[currentFrameIndex]
			} else {
				dataEnd = len(audioData)
			}
			chunkData := audioData[dataStart:dataEnd]

			// Write the chunk
			chunkFileName := filepath.Join(oldDir, fmt.Sprintf("%s.flac", helpers.GenerateULID()))
			chunkFile, err := os.Create(chunkFileName)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s error creating chunk file: %v", record_id, err))
				continue
			}

			_, err = chunkFile.Write(headers)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s error writing headers to chunk file: %v", record_id, err))
				continue
			}

			_, err = chunkFile.Write(chunkData)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s error writing chunk data to chunk file: %v", record_id, err))
				continue
			}

			chunkFile.Close()
			r.Set("file", record.Id)
			r.Set("chunk_path", chunkFileName)
			r.Set("start_byte_offset", dataStart)
			r.Set("end_byte_offset", dataEnd)
			r.Set("chunk_order", chunkOrder)
			r.Set("chunk_size", chunkSize)
			err = app.Save(r)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s error saving chunk %d: %v", record_id, chunkOrder, err))
				continue
			}
			chunkOrder++
			chunkIds = append(chunkIds, r.Id)
		}

		record.Set("processed", true)
		err = app.Save(record)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s error saving record: %v", record_id, err))
			continue
		}

		// Remove the original file
		err = file.Close()
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s error closing file: %v", record_id, err))
			continue
		}

		err = os.Remove(flacFilePath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s error removing file: %v", record_id, err))
			continue
		}
	}

	end_time := time.Since(start_time).Nanoseconds()
	if len(errors) > 0 {
		app.Logger().Error(
			"ChunkJob",
			"totalRecords", total_records,
			"totalChunks", len(records),
			"execTime", float64(end_time)/1000000,
			"chunkIds", chunkIds,
			"errors", errors,
		)
	} else {
		app.Logger().Info(
			"ChunkJob",
			"totalRecords", total_records,
			"totalChunks", len(records),
			"execTime", float64(end_time)/1000000,
			"chunkIds", chunkIds,
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
