package handlers

import (
	"github.com/patrickmn/go-cache"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	file "github.com/rudyrdx/music-streamer/chunker/handlers/File"
	stream "github.com/rudyrdx/music-streamer/chunker/handlers/Stream"
)

func SetupHandlers(se *core.ServeEvent, app *pocketbase.PocketBase, c *cache.Cache) error {

	se.Router.Bind(apis.BodyLimit(500 << 20))

	se.Router.GET("/hello", func(re *core.RequestEvent) error {
		return re.String(200, "Hello world!")
	})

	se.Router.POST("/file", func(e *core.RequestEvent) error {
		return file.HandleUpload(e)
	})

	se.Router.GET("/stream", func(e *core.RequestEvent) error {
		return stream.Stream(e, app, c)
	})

	// se.Router.GET("/metadata", func(e *core.RequestEvent) error {
	// 	return stream.GetChunkData(e, app, c)
	// })
	// se.Router.GET("/chunk", func(e *core.RequestEvent) error {
	// 	return stream.HandleChunkRequest(e, app, c)
	// })

	return se.Next()
}
