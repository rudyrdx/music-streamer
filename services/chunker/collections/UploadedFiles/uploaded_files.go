package uploadedfiles

import (
	"github.com/pocketbase/pocketbase/core"
)

func CreateCollection() *core.Collection {
	collection := core.NewBaseCollection("UploadedFiles")

	collection.Fields.Add(&core.TextField{
		Name:     "file_path",
		Required: true,
	})

	collection.Fields.Add(&core.TextField{
		Name:     "file_name",
		Required: true,
		Max:      256,
	})

	collection.Fields.Add(&core.NumberField{
		Name:     "file_size",
		Required: true,
	})

	collection.Fields.Add(&core.BoolField{
		Name:     "processed",
		Required: true,
	})

	collection.Fields.Add(&core.JSONField{
		Name:     "file_info",
		Required: true,
	})

	collection.Fields.Add(&core.AutodateField{
		Name:     "created",
		OnCreate: true,
	})

	collection.Fields.Add(&core.AutodateField{
		Name:     "updated",
		OnCreate: true,
		OnUpdate: true,
	})

	return collection
}
