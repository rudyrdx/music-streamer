package file

import (
	"strings"

	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/core"
)

// once the file is uploaded, it will be added to the database, then it will be sent to processing
// in processing, there will be a service running that will take care of the file chunking,
func HandleUpload(re *core.RequestEvent) error {

	check_muiltipart := strings.Split(re.Request.Header.Get("Content-Type"), ";")[0] == "multipart/form-data"

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

	for _, file := range files {
		name := file.Name
		uu_id := uuid.New().String()
		println(name)
		println(uu_id)
	}

	return nil
}

//any incoming requests to this chunker service will be expected to have
//the totp token in header, ensuring that the request is coming from a valid source
//or, if we are using server side rendering, why dont we use rpc ?
