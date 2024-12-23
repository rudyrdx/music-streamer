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