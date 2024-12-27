package stream

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

func HandleStreamer(e *core.RequestEvent) error {

	rnge := e.Request.Header.Get("Range")
	if rnge == "" {
		return e.String(400, "Range header not found")
	}

	fmt.Println("Range: ", rnge)

	return nil
}
