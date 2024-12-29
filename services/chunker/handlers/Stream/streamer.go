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

// we have stored the chunks in database, when a music with that id is requested,
//we will have to lookup all the chunks and then for whatever range the client requests,
//we will have to stream the chunks to the client.
//question is instead of lookup the db for each chunk, i was thinking to have a in memory
//datastructure that allows quick retreival of chunk based on range
//whenever i pass in the range, i will get the chunk with that valid range
//so key is range, for that range awnd that chunk.
//like a case statement, if range is 0-100, then chunk is 1, if range is 101-200, then chunk is 2
//but in a datastructure that allows quick lookup like in constant time