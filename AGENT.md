# Project - spawn-qdrant

## introduction

This is a simple project to spawn a Qdrant instance using Docker. The project is written in Go and is a simple wrapper around the Qdrant Docker image. The application can be used to spawn N instances of qdrant in a linux machine using docker. 

## operations

The command will be called `spawn-qdrant` and takes the following arguments from the .env file
- REST_PORT
- GRPC_PORT

if the .env file is not present, the application will use the default values
- REST_PORT=6333
- GRPC_PORT=6334
for the starting ports for the first instance. 

### spawn sub command

The number of instances to spawn is passed as an argument to the spawn sub command and is mandatory. this is called `instance_count`. This will be provided as a command line argument to the spawn sub command.

For example 

``` bash 
spawn-qdrant spawn 2
```

This will spawn two qdrant instances with the following ports and storage locations

- Container 1 Named qdrant-01 and uses ports 6333 and 6334 and storage location ~/.qdrant_storage01
- Container 2 Named qdrant-02 and uses ports 6335 and 6336 and storage location ~/.qdrant_storage02

The ports are the starting port number for the qdrant instance. The actual ports used will be REST_PORT + 2*(instance_count - 1) and GRPC_PORT + 2*(instance_count - 1) if instance_count > 1. If Instance count is 1, then the ports used will be REST_PORT and GRPC_PORT.

The storage for each qdrant instance will be stored in ~/.qdrant_storage{instance_count}.

For example if instance_count is 1, then the ports used will be 6333 and 6334 and the storage will be stored in ~/.qdrant_storage01 and container will be named qdrant-01.

If instance_count is 2, then the application spawns two container with the following REST, GPRC ports and storage location 

- Container 1 Named qdrant-01 and uses ports 6333 and 6334 and storage location ~/.qdrant_storage01
- Container 2 Named qdrant-02 and uses ports 6335 = 6333 + 2*(2-1) and 6336 = 6334 + 2*(2 - 1) and storage location ~/.qdrant_storage02

### stop sub command

The stop sub command will stop all the qdrant instances that were spawned using the spawn sub command. The application can be used as follows  

#### stop all sub command

This will stop all the qdrant instances that were spawned using the spawn sub command.

``` bash 
spawn-qdrant stop all 
```

#### stop n sub command

This will stop the n-th qdrant instance that was spawned using the spawn sub command.

``` bash 
spawn-qdrant stop n
```

### clean sub command 