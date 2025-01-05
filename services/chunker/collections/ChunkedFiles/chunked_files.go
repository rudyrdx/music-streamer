package chunkedfiles

import (
	"github.com/pocketbase/pocketbase/core"
)

func CreateCollection() *core.Collection {
	collection := core.NewBaseCollection("ChunkedFiles")
	collection.Id = "CFTable123"

	collection.Fields.Add(&core.RelationField{
		Name:          "file",
		Required:      true,
		CascadeDelete: true,
		CollectionId:  "UFTable123",
	})

	collection.Fields.Add(&core.TextField{
		Name:     "chunk_path",
		Required: true,
	})

	collection.Fields.Add(&core.NumberField{
		Name: "start_byte_offset",
	})

	collection.Fields.Add(&core.NumberField{
		Name: "end_byte_offset",
	})

	collection.Fields.Add(&core.NumberField{
		Name:     "chunk_order",
		Required: true,
	})

	collection.Fields.Add(&core.NumberField{
		Name:     "chunk_size",
		Required: true,
	})

	collection.Fields.Add()

	collection.Fields.Add(&core.AutodateField{
		Name:     "created",
		OnCreate: true,
	})

	return collection
}
