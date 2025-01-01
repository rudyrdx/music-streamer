package stream

import (
	"fmt"
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
	e.Response.WriteHeader(206)
	e.Response.Write(buffer)
	return nil
}

type Range struct {
	start, end int
}

type RangeHashMap struct {
	data map[Range]interface{}
}

func NewRangedMap() *RangeHashMap {
	return &RangeHashMap{
		data: make(map[Range]interface{}),
	}
}

func (r *RangeHashMap) Add(start, end int, value interface{}) error {
	if start > end {
		return fmt.Errorf("start cannot be greater than end")
	}
	r.data[Range{start, end}] = value
	return nil
}

func (r *RangeHashMap) Get(num int) (interface{}, bool) {
	for key, val := range r.data {
		if num >= key.start && num <= key.end {
			return val, true
		}
	}
	return nil, false
}

func HandleChunkedStreamer(e *core.RequestEvent, app *pocketbase.PocketBase, c *cache.Cache) error {

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

	chunks, err := helpers.LookupFromCacheOrDB(c, "ChunkedFiles_"+record.Id, func() ([]*core.Record, error) {
		return app.FindAllRecords("ChunkedFiles", dbx.HashExp{"file": record.Id})
	}, cache.DefaultExpiration)
	if err != nil {
		return e.String(500, "Failed to find chunks")
	}

	if len(chunks) < 1 {
		return e.String(500, "No chunks found")
	}

	//each chunk has startOffset and endOffset along with chunkPath
	//we need to have a datastructure that allows quick lookup of chunk based on range
	//a map of range to chunk

	var chunkMap = NewRangedMap()
	for _, chunk := range chunks {
		startOffset := int64(chunk.GetInt("start_byte_offset"))
		endOffset := int64(chunk.GetInt("end_byte_offset"))
		chunkPath := chunk.Get("chunk_path").(string)
		chunkMap.Add(int(startOffset), int(endOffset), string(chunkPath))
	}

	rangeStart, _, err := parseRange(rnge, float64(record.GetInt("file_size")))
	if err != nil {
		return e.String(416, "Invalid range")
	}

	path, t := chunkMap.Get(int(rangeStart) + 10)
	if !t {
		return e.String(500, "Failed to find chunk")
	}

	fileBytes, err := os.ReadFile(path.(string))
	if err != nil {
		return e.String(500, "Failed to read file")
	}

	e.Response.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", 0, len(fileBytes), len(fileBytes)))
	e.Response.Header().Set("Accept-Ranges", "bytes")
	e.Response.Header().Set("Content-Length", strconv.Itoa(len(fileBytes)))
	e.Response.Header().Set("Content-Type", "audio/flac")
	//add cors header
	e.Response.Header().Set("Access-Control-Allow-Origin", "*")
	e.Response.WriteHeader(206)
	e.Response.Write(fileBytes)
	//hny
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
