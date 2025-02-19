package stream

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/patrickmn/go-cache"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rudyrdx/music-streamer/chunker/helpers"
)

func HandleStreamerCached(e *core.RequestEvent, app *pocketbase.PocketBase, c *cache.Cache) error {

	rnge := e.Request.Header.Get("Range")
	param := e.Request.URL.Query().Get("id")
	if rnge == "" || param == "" {
		return e.String(400, "Invalid request")
	}

	// s := time.Now()
	col, err := helpers.LookupFromCacheOrDB(c, "UploadedFiles", func() (*core.Collection, error) {
		return app.FindCollectionByNameOrId("UploadedFiles")
	}, cache.DefaultExpiration)
	if err != nil {
		return e.String(500, "Failed to find collection")
	}

	record, err := helpers.LookupFromCacheOrDB(c, param, func() (*core.Record, error) {
		return app.FindRecordById(col, param)
	}, cache.DefaultExpiration)
	if err != nil {
		return e.String(400, "Invalid request")
	}

	// end := time.Since(s).Nanoseconds()
	// secs := float64(end) / 1000000000
	// fmt.Println("Time taken to stream", secs)

	file_path := record.Get("file_path").(string)
	file_size := record.Get("file_size").(float64)

	fileCacheKey := param + "file"
	filePointerInterface, found := c.Get(fileCacheKey)
	var filePointer *os.File
	if !found {
		// Open the file
		filePointer, err = os.Open(file_path)
		if err != nil {
			return e.String(500, "Failed to open file")
		}
		// Cache the file pointer
		c.Set(fileCacheKey, filePointer, cache.DefaultExpiration)
	} else {
		filePointer = filePointerInterface.(*os.File)
	}

	// Seek to the correct position based on the range
	rangeStart, rangeEnd, err := parseRange(rnge, file_size)
	if err != nil {
		return e.String(416, "Invalid range")
	}

	// Move the file pointer to the start of the range
	_, err = filePointer.Seek(rangeStart, 0)
	if err != nil {
		return e.String(500, "Failed to seek to range")
	}

	// Read the bytes for the requested range
	buffer := make([]byte, rangeEnd-rangeStart)
	_, err = filePointer.Read(buffer)
	if err != nil {
		return e.String(500, "Failed to read file")
	}

	e.Response.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", rangeStart, rangeEnd, int(file_size)))
	e.Response.Header().Set("Accept-Ranges", "bytes")
	e.Response.Header().Set("Content-Length", strconv.Itoa(len(buffer)))
	e.Response.Header().Set("Content-Type", "audio/flac")
	e.Response.Header().Set("Access-Control-Allow-Origin", "*")
	e.Response.WriteHeader(206)
	e.Response.Write(buffer)

	return nil
}

func HandleStreamer(e *core.RequestEvent, app *pocketbase.PocketBase) error {

	rnge := e.Request.Header.Get("Range")
	param := e.Request.URL.Query().Get("id")
	if rnge == "" || param == "" {
		return e.String(400, "Invalid request")
	}

	// s := time.Now()
	col, err := app.FindCollectionByNameOrId("UploadedFiles")
	if err != nil {
		return e.String(500, "Failed to find collection")
	}

	record, err := app.FindRecordById(col, param)
	if err != nil {
		return e.String(400, "Invalid request")
	}
	// end := time.Since(s).Nanoseconds()
	// secs := float64(end) / 1000000000
	// fmt.Println("Time taken to stream", secs)

	file_path := record.Get("file_path").(string)
	file_size := record.Get("file_size").(float64)

	filePointer, err := os.Open(file_path)
	if err != nil {
		return e.String(500, "Failed to open file")
	}

	// Seek to the correct position based on the range
	rangeStart, rangeEnd, err := parseRange(rnge, file_size)
	if err != nil {
		return e.String(416, "Invalid range")
	}

	// Move the file pointer to the start of the range
	_, err = filePointer.Seek(rangeStart, 0)
	if err != nil {
		return e.String(500, "Failed to seek to range")
	}

	// Read the bytes for the requested range
	buffer := make([]byte, rangeEnd-rangeStart)
	_, err = filePointer.Read(buffer)
	if err != nil {
		return e.String(500, "Failed to read file")
	}

	e.Response.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", rangeStart, rangeEnd, int(file_size)))
	e.Response.Header().Set("Accept-Ranges", "bytes")
	e.Response.Header().Set("Content-Length", strconv.Itoa(len(buffer)))
	e.Response.Header().Set("Content-Type", "audio/flac")
	e.Response.Header().Set("Access-Control-Allow-Origin", "*")
	e.Response.WriteHeader(206)
	e.Response.Write(buffer)
	return nil
}

func parseRange(rnge string, fileSize float64) (int64, int64, error) {
	// Example: "bytes=100-200"
	if !strings.HasPrefix(rnge, "bytes=") {
		return 0, 0, fmt.Errorf("invalid range format")
	}

	// Remove the "bytes=" prefix and split the range
	rangeParts := strings.TrimPrefix(rnge, "bytes=")
	ranges := strings.Split(rangeParts, "-")
	if len(ranges) != 2 {
		return 0, 0, fmt.Errorf("invalid range format")
	}

	// Parse the start and end values
	start, err := strconv.ParseInt(ranges[0], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid start value")
	}

	// If the end value is empty, it means we are requesting from the start to the end of the file
	var end int64
	if ranges[1] == "" {
		end = int64(fileSize) - 1
	} else {
		end, err = strconv.ParseInt(ranges[1], 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid end value")
		}
	}

	// Validate the range
	if start > end || start < 0 || end >= int64(fileSize) {
		return 0, 0, fmt.Errorf("requested range out of bounds")
	}

	return start, end, nil
}

func GetChunkData(e *core.RequestEvent, app *pocketbase.PocketBase, c *cache.Cache) error {
	param := e.Request.URL.Query().Get("id")
	if param == "" {
		return e.String(400, "Invalid request")
	}

	col, err := helpers.LookupFromCacheOrDB(c, "UploadedFiles", func() (*core.Collection, error) {
		return app.FindCollectionByNameOrId("UploadedFiles")
	}, cache.DefaultExpiration)
	if err != nil {
		return e.String(500, "Failed to find collection")
	}

	record, err := helpers.LookupFromCacheOrDB(c, param, func() (*core.Record, error) {
		return app.FindRecordById(col, param)
	}, cache.DefaultExpiration)
	if err != nil {
		return e.String(400, "Invalid request")
	}

	chunks, err := helpers.LookupFromCacheOrDB(c, "ChunkedFiles_"+record.Id, func() ([]*core.Record, error) {
		return app.FindAllRecords("ChunkedFiles", dbx.HashExp{"file": record.Id})
	}, cache.DefaultExpiration)
	if err != nil {
		return e.String(500, "Failed to find chunks")
	}

	metadataChunks := make(map[int]interface{})
	for _, chunk := range chunks {
		order := chunk.GetInt("chunk_order")
		size := chunk.GetInt("chunk_size")
		sOffset := float64(chunk.GetInt("start_byte_offset"))
		eOffset := float64(chunk.GetInt("end_byte_offset"))
		metadataChunks[order] = map[string]interface{}{
			"id":          chunk.Id,
			"size":        size,
			"startOffset": sOffset,
			"endOffset":   eOffset,
		}
	}

	metadata := map[string]interface{}{
		"fileSize": record.Get("file_size"),
		"chunks":   metadataChunks,
	}

	e.Response.Header().Set("Access-Control-Allow-Origin", "*")
	return e.JSON(200, metadata)
}

func HandleChunkRequest(e *core.RequestEvent, app *pocketbase.PocketBase, c *cache.Cache) error {
	param := e.Request.URL.Query().Get("id")

	if param == "" {
		return e.String(400, "Invalid request")
	}

	record, err := app.FindRecordById("ChunkedFiles", param)
	if err != nil {
		return e.String(400, "Invalid request")
	}

	file, err := os.Open(record.Get("chunk_path").(string))
	if err != nil {
		return e.String(500, "Failed to open file")
	}
	defer file.Close()

	e.Response.Header().Set("Content-Type", "audio/flac")
	e.Response.Header().Set("Access-Control-Allow-Origin", "*")
	e.Response.WriteHeader(206)

	// Stream FFmpeg output directly to the client
	if _, err := io.Copy(e.Response, file); err != nil {
		fmt.Println("Streaming error:", err)
		e.String(500, "Failed to stream audio")
	}
	//hny
	return nil
}

func Stream(e *core.RequestEvent, app *pocketbase.PocketBase, c *cache.Cache) error {
	// Parse the Range header
	_range := e.Request.Header.Get("Range")
	_id := e.Request.URL.Query().Get("id")

	if _range == "" || _id == "" {
		return e.String(400, "Invalid request")
	}

	startPos, err := getRange(_range)
	if err != nil {
		return e.String(400, "Invalid range")
	}

	// Retrieve the records for the requested file
	var Records []*core.Record
	recordsInterface, found := c.Get(_id)
	if !found {
		records, err := app.FindAllRecords("ChunkedFiles", dbx.HashExp{"file": _id})
		if err != nil {
			return e.String(500, "Failed to find records")
		}
		c.Set(_id, records, cache.DefaultExpiration)
		Records = records
	} else {
		Records = recordsInterface.([]*core.Record)
	}

	if len(Records) == 0 {
		return e.String(400, "Invalid request")
	}

	// Find the chunk containing the requested byte
	var record *core.Record
	for _, r := range Records {
		start := int64(r.GetInt("start_byte_offset"))
		end := int64(r.GetInt("end_byte_offset"))
		if startPos >= start && startPos <= end {
			record = r
			break
		}
	}

	if record == nil {
		return e.String(416, "Range Not Satisfiable")
	}

	// Get the file details
	fileSize := record.GetInt("file_size")
	chunkPath := record.Get("chunk_path").(string)
	rangeStart := int64(record.GetInt("start_byte_offset"))
	rangeEnd := int64(record.GetInt("end_byte_offset"))

	// Open the file and stream data
	file, err := os.Open(chunkPath)
	if err != nil {
		return e.String(500, "Failed to open file")
	}
	defer file.Close()

	// Set response headers
	e.Response.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", rangeStart, rangeEnd, fileSize))
	e.Response.Header().Set("Accept-Ranges", "bytes")
	e.Response.Header().Set("Content-Length", strconv.Itoa(int(rangeEnd-rangeStart+1)))
	e.Response.Header().Set("Content-Type", "audio/flac")
	e.Response.Header().Set("Access-Control-Allow-Origin", "*")
	e.Response.WriteHeader(206)

	// Stream the chunk data
	if _, err := io.Copy(e.Response, file); err != nil {
		fmt.Println("Streaming error:", err)
		return e.String(500, "Failed to stream audio")
	}

	return nil
}

func getRange(r string) (int64, error) {
	if !strings.HasPrefix(r, "bytes=") {
		return 0, fmt.Errorf("invalid range format")
	}

	rangeParts := strings.TrimPrefix(r, "bytes=")
	ranges := strings.Split(rangeParts, "-")
	if len(ranges) != 2 {
		return 0, fmt.Errorf("invalid range format")
	}

	start, err := strconv.ParseInt(ranges[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid start value")
	}

	return start, nil
}

//ok 1 approach that i can think is, we prepare a hashmap for the chunk ranges and ids, and we send the json
//to the client based on which the clinet will request the chunk of that range.

//326kb worth of data on spotify premium
// we have stored the chunks in database, when a music with that id is requested,
//we will have to lookup all the chunks and then for whatever range the client requests,
//we will have to stream the chunks to the client.
//question is instead of lookup the db for each chunk, i was thinking to have a in memory
//datastructure that allows quick retreival of chunk based on range
//whenever i pass in the range, i will get the chunk with that valid range
//so key is range, for that range awnd that chunk.
//like a case statement, if range is 0-100, then chunk is 1, if range is 101-200, then chunk is 2
//but in a datastructure that allows quick lookup like in constant time
