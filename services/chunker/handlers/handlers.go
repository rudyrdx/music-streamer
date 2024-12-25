package handlers

import "github.com/pocketbase/pocketbase/core"

func SetupHandlers(se *core.ServeEvent) error {

	se.Router.GET("/hello", func(re *core.RequestEvent) error {
		return re.String(200, "Hello world!")
	})

	// se.Router.POST("/file", func(e *core.RequestEvent) error {
	// 	return HandleUpload(e)
	// })

	return se.Next()
}
