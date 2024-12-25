package collections

import (
	"fmt"

	"github.com/pocketbase/pocketbase"
	uploadedfiles "github.com/rudyrdx/music-streamer/chunker/collections/UploadedFiles"
)

func SetupCollections(AppInstance *pocketbase.PocketBase) {

	//uploadFiles table
	_, err := AppInstance.FindCollectionByNameOrId("UploadedFiles")
	if err != nil {
		uploadedfiles.CreateCollection()
	} else {
		fmt.Println("Collection UploadedFiles exists ✓")
	}
}
