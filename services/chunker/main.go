package main

//the task of this service will be to accept a
//music file only, parse the meta data,
//chunk the file into small chunks
//store it in this service's edge db
//tell the orchestrator that the file has been created and chunked
//then the orchestrator will pull the file and save it in mainDB
import (
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rudyrdx/music-streamer/chunker/collections"
	"github.com/rudyrdx/music-streamer/chunker/handlers"
	"github.com/rudyrdx/music-streamer/chunker/helpers"
)

func main() {
	app := pocketbase.New()

	app.OnServe().BindFunc(func(be *core.ServeEvent) error {

		collections.SetupCollections(app)

		return be.Next()
	})

	app.OnServe().BindFunc(handlers.SetupHandlers)
	
	app.Cron().MustAdd("Chunk", "*/1 * * * *", func() {

		records, err := app.FindRecordsByFilter(
			"UploadedFiles",     // collection
			"processed = False", // filter
			"-created",          // sort
			2,                   // limit
			0,                   // offset
		)
		if err != nil {
			return
		}

		if len(records) < 1 {
			return
		}

		fmt.Println("Processing records", len(records))

		for _, record := range records {
			path := record.Get("file_path").(string)
			size := record.Get("file_size").(float64)
			src, err := os.Open(path)
			if err != nil {
				fmt.Println("error opening file", err)
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
				fmt.Println("error finding collection", err)
				return
			}

			for i, chunk := range chunks {
				// fmt.Printf("Chunk %d: Start: %d, End: %d\n", i+1, chunk[0], chunk[1])
				r := core.NewRecord(collection)
				chunkName := helpers.GenerateULID()
				chunkPath := fmt.Sprintf("./%s/%s", oldDir, chunkName)

				dst, err := os.Create(chunkPath)
				if err != nil {
					fmt.Printf("Error creating chunk file %s: %v\n", chunkPath, err)
					continue
				}

				_, err = src.Seek(chunk[0], io.SeekStart)
				if err != nil {
					fmt.Printf("Error seeking in source file for chunk %d: %v\n", i+1, err)
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
				r.Set("chunk_order", i+1)
				r.Set("chunk_size", chunkSize)
				err = app.Save(r)
				if err != nil {
					fmt.Printf("Error saving chunk %d: %v\n", i+1, err)
				}
			}
			record.Set("processed", true)
			err = app.Save(record)
			if err != nil {
				fmt.Println("error saving record", err)
				continue
			}
			//remove old file
			err = src.Close()
			if err != nil {
				fmt.Println("error closing file", err)
				continue
			}
			err = os.Remove(path)
			if err != nil {
				fmt.Println("error removing file", err)
				continue
			}
		}
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
