package main

//the task of this service will be to accept a
//music file only, parse the meta data,
//chunk the file into small chunks
//store it in this service's edge db
//tell the orchestrator that the file has been created and chunked
//then the orchestrator will pull the file and save it in mainDB
import (
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	app := pocketbase.New()

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// registers new "GET /hello" route
		se.Router.GET("/hello", func(re *core.RequestEvent) error {
			return re.String(200, "Hello world!")
		})

		se.Router.POST("/file", func(e *core.RequestEvent) error {
			return HandleUpload(e)
		})

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

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
