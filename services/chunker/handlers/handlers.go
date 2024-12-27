package handlers

import (
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	file "github.com/rudyrdx/music-streamer/chunker/handlers/File"
	stream "github.com/rudyrdx/music-streamer/chunker/handlers/Stream"
)

func SetupHandlers(se *core.ServeEvent) error {

	se.Router.Bind(apis.BodyLimit(500 << 20))

	se.Router.GET("/hello", func(re *core.RequestEvent) error {
		return re.String(200, "Hello world!")
	})

	se.Router.POST("/file", func(e *core.RequestEvent) error {
		return file.HandleUpload(e)
	})

	se.Router.GET("/stream", func(e *core.RequestEvent) error {
		return stream.HandleStreamer(e)
	})

	return se.Next()
}
