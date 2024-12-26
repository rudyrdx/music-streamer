package collections

import (
	"github.com/pocketbase/pocketbase"
	chunkedfiles "github.com/rudyrdx/music-streamer/chunker/collections/ChunkedFiles"
	uploadedfiles "github.com/rudyrdx/music-streamer/chunker/collections/UploadedFiles"
)

func SetupCollections(AppInstance *pocketbase.PocketBase) error {

	_, err := AppInstance.FindCollectionByNameOrId("UploadedFiles")
	if err != nil {
		err := AppInstance.Save(uploadedfiles.CreateCollection())
		if err != nil {
			// fmt.Println("Error saving collection UploadedFiles")
			return err
		}
	}

	_, err = AppInstance.FindCollectionByNameOrId("ChunkedFiles")
	if err != nil {
		err := AppInstance.Save(chunkedfiles.CreateCollection())
		if err != nil {
			// fmt.Println("Error saving collection UploadedFiles")
			return err
		}
	}

	return nil
}
