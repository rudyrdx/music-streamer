package main

//the task of this service will be to accept a
//music file only, parse the meta data,
//chunk the file into small chunks
//store it in this service's edge db
//tell the orchestrator that the file has been created and chunked
//then the orchestrator will pull the file and save it in mainDB
import (
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rudyrdx/music-streamer/chunker/collections"
	"github.com/rudyrdx/music-streamer/chunker/handlers"
)

func main() {
	app := pocketbase.New()

	app.OnServe().BindFunc(func(be *core.ServeEvent) error {

		collections.SetupCollections(app)

		return be.Next()
	})

	app.OnServe().BindFunc(handlers.SetupHandlers)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
