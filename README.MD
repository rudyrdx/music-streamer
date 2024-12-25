The task of this app will be to stream music.
each task related to music streaming will be divided as services
connected to a database for orchestrating everything.

we can provide each service its own version of database like storage where big data synchronizations can happen dynamically or slowly.
these edge databases can act like cache layers for these services.

the services are mostly music upload where chunking, audio metadata storing record in database (main & edge) will happen. 

then there should be a service for client app, (backend and frontend) having edge db and main db. edge db can be a pocketbase type

![alt text](<Pasted image 20241222222236.png>)

here service can be anything from frontend to music chunker or chunk streamer.

working at alchemus, i believe that a main central database can be vertically large with no issues. we can push data to it whenever we feel free. For the main DB we have to setup a redudnant backup db for Disaster management and recovery.

Main db of choice: POstgresql
Edge db of choice: pocketbase (provides admin ui and embedded sql server)

![alt text](<Pasted image 20241222223804.png>)

Orchestrators will be responsible for issuing commands and syncing data to and fro the services.
Main db will be the point of storage of all data with DR DB
Services will each have an edge database. Each service will be aware of the other type of services running close to it (no same type)
music upload or stream commands can come from the frontend and then frontend will invoke the music chunker or streamer services to do the job through it.

now regarding regional scalability:
lets say in a region there will be only one service of each type.
but if the load increases, we can deduplicate the that region's services and bind them behind a load balancer layer to distribute load.

## Plans
User has to login inorder to listen to music.
UI where the user can listen to music, manage playlist
listen to music means:
request a song, so that it can be streamed.
manage playlist means C R U D playlists of that user

how will the music streamer know whether the music request is from an authenticated user?

~~on receiving a request from a user we would need to check first, the user is authenticated, but as our streaming service is running separately from the frontend service, in order to check whether this request is from a genuine and authenticated user, on receiving the request~~ 
- ~~the streamer service periodically sends requests to the frontend service to confirm this user's status~~
- ~~or the streamer service will check the current status of the user but that will require a database call to the frontend service which defeats the whole purpose~~ 
- ~~is there a way to be established between the frontend service and streamer service that the request b/w the services is genuine preventing malicious attempts?~~

we can use the TOTP approach with a single use policy to prevent missuses of our streamer and only the frontend has the logic to reproduce it.

![alt text](<Pasted image 20241223111905.png>)

Database design:
chunker
UploadFiles table where all files uploaded will go and processed status will be saved
after processing, the file's chunks ids will be saved in the chunk table and reference will be added to the UploadFiles table

streamer
table will store reference ...
i think the streamer and chunker have to be one. the chunker will be responsible for chunking and streaming the music.

or ill have to setup the orchestrator to take in the data from chunker and provide it to the
streamer service's database or the streamer service can directly request and cache it in its edge database