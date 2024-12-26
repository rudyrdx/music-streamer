package chunker

//the chunker will read the uploadfile db, get all the records that need processing
//lets say 10 at a time.
//then as it will process it and chunk them, it will update the record in the db and add the reference id to the
//uploaded file table so that the orchestrator can read the chunks of that file and update the maindb

