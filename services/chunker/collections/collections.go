package collections

import (
	"fmt"

	"github.com/pocketbase/pocketbase"
	uploadedfiles "github.com/rudyrdx/music-streamer/chunker/collections/UploadedFiles"
)

func SetupCollections(AppInstance *pocketbase.PocketBase) error {

	//uploadFiles table
	_, err := AppInstance.FindCollectionByNameOrId("UploadedFiles")
	if err != nil {
		err := AppInstance.Save(uploadedfiles.CreateCollection())
		if err != nil {
			// fmt.Println("Error saving collection UploadedFiles")
			return err
		}
	} else {
		fmt.Println("Collection UploadedFiles exists")
	}

	
	return nil
}
