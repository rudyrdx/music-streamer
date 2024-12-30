package chunker

import (
	"fmt"
	"io"
	"math"
	"os"
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

	for _, record := range records {
		record_id := record.Id
		path := record.Get("file_path").(string)
		size := record.Get("file_size").(float64)
		src, err := os.Open(path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s error opening file %s: %v", record_id, path, err))
			continue
		}
		// Constants
		const chunkSize = float64(1 * 1024 * 1024) // 1 MB in bytes (as float64)

		// Calculate number of chunks
		numChunks := int(math.Ceil(size / chunkSize)) // Use math.Ceil to round up to the next whole number

		// Map chunks to offsets
		chunks := make([][2]int64, numChunks) // Each chunk: [startOffset, endOffset]
		for i := 0; i < numChunks; i++ {
			startOffset := int64(float64(i) * chunkSize)
			endOffset := int64(math.Min(float64(startOffset)+chunkSize-1, size-1)) // Ensure we don't exceed file size
			chunks[i] = [2]int64{startOffset, endOffset}
		}

		oldDir := strings.Split(path, "/")[1]

		collection, err := app.FindCollectionByNameOrId("ChunkedFiles")
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s error finding collection ChunkedFiles: %v", record_id, err))
			continue
		}

		for i, chunk := range chunks {
			// fmt.Printf("Chunk %d: Start: %d, End: %d\n", i+1, chunk[0], chunk[1])
			r := core.NewRecord(collection)
			chunkName := helpers.GenerateULID()
			chunkPath := fmt.Sprintf("./%s/%s", oldDir, chunkName)

			dst, err := os.Create(chunkPath)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s error creating chunk %d: %v", record_id, i+1, err))
				continue
			}

			_, err = src.Seek(chunk[0], io.SeekStart)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s error seeking to start of chunk %d: %v", record_id, i+1, err))
				dst.Close()
				continue
			}

			chunkSize := chunk[1] - chunk[0] + 1
			_, err = io.CopyN(dst, src, chunkSize)
			if err != nil && err != io.EOF {
				fmt.Printf("Error writing chunk %d: %v\n", i+1, err)
			}

			// Close the chunk file
			dst.Close()
			r.Set("file", record.Id)
			r.Set("chunk_path", chunkPath)
			r.Set("start_byte_offset", chunk[0]+1)
			r.Set("end_byte_offset", chunk[1])
			r.Set("chunk_order", i+1)
			r.Set("chunk_size", chunkSize)
			err = app.Save(r)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s error saving chunk %d: %v", record_id, i+1, err))
				continue
			}
			chunkIds = append(chunkIds, r.Id)
		}
		record.Set("processed", true)
		err = app.Save(record)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s error saving record: %v", record_id, err))
			continue
		}
		//remove old file
		err = src.Close()
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s error closing file: %v", record_id, err))
			continue
		}
		err = os.Remove(path)
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
