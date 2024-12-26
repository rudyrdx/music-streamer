package file

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/core"
)

// once the file is uploaded, it will be added to the database, then it will be sent to processing
// in processing, there will be a service running that will take care of the file chunking,

// this flac data will be provided from the frontend using the musicmetadata library
type FileData struct {
}

func HandleUpload(re *core.RequestEvent) error {

	check_muiltipart := strings.Split(re.Request.Header.Get("Content-Type"), ";")[0] == "multipart/form-data"

	//the tmp file path is in the root folder of this main.go file
	tmp_dir := "./tmp"
	if !check_muiltipart {
		return re.String(400, "Invalid request")
	}

	files, err := re.FindUploadedFiles("file")
	if err != nil {
		return re.String(400, "Invalid request")
	}

	file_len := len(files)

	if file_len < 1 {
		return re.String(400, "Invalid request")
	}

	collection, err := re.App.FindCollectionByNameOrId("UploadedFiles")
	if err != nil {
		return re.String(500, "Internal server error")
	}

	failures := make([]string, 0)
	for _, file := range files {
		oName := file.OriginalName
		uu_id := uuid.New().String()
		size := file.Size
		path := fmt.Sprintf("%s/%s.flac", tmp_dir, uu_id)
		record := core.NewRecord(collection)
		record.Set("file_path", path)
		record.Set("file_name", oName)
		record.Set("file_size", size)
		record.Set("processed", "false")
		record.Set("file_info", map[string]interface{}{"key": "value"})
		err = re.App.Save(record)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}

		fo, err := os.Create(path)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		defer fo.Close() // Ensure the file is closed properly after writing

		src, err := file.Reader.Open()
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		defer src.Close() // Ensure the source file is closed properly

		// Copy the source file directly to the destination file in chunks
		if _, err := io.Copy(fo, src); err != nil {
			failures = append(failures, err.Error())
			continue
		}
	}

	if len(failures) > 0 {
		return re.String(500, fmt.Sprintf("Failed to save files: %v", failures))
	}

	return nil
}

//any incoming requests to this chunker service will be expected to have
//the totp token in header, ensuring that the request is coming from a valid source
//or, if we are using server side rendering, why dont we use rpc ?
