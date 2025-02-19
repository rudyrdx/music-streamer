package main

//the task of this service will be to accept a
//music file only, parse the meta data,
//chunk the file into small chunks
//store it in this service's edge db
//tell the orchestrator that the file has been created and chunked
//then the orchestrator will pull the file and save it in mainDB
import (
	"log"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rudyrdx/music-streamer/chunker/collections"
	"github.com/rudyrdx/music-streamer/chunker/handlers"
	"github.com/rudyrdx/music-streamer/chunker/handlers/chunker"
)

func main() {
	app := pocketbase.New()

	c := cache.New(5*time.Minute, 10*time.Minute)

	app.OnServe().BindFunc(func(be *core.ServeEvent) error {
		collections.SetupCollections(app)
		return be.Next()
	})

	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		handlers.SetupHandlers(e, app, c)
		return e.Next()
	})

	app.Cron().MustAdd("Chunk", "*/1 * * * *", func() {
		mb_2 := 1024 * 1024 * 1
		chunker.ChunkJob(app, int64(mb_2), true)
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
