# Project - spawn-qdrant

## Introduction

This is a simple project to spawn a Qdrant instance using Docker. The project is written in Go and is a simple wrapper around the Qdrant Docker image. The application can be used to spawn N instances of qdrant in a linux machine using docker. 

## Application usage

The application is called `spawn-qdrant` and takes the following arguments from the .env file
- REST_PORT
- GRPC_PORT

if the .env file is not present, the application will use the default values
- REST_PORT=6333
- GRPC_PORT=6334
for the starting ports for the first instance. 

At the start of the application it will check if docker or podman is installed and use docker if it is installed and podman if it is not installed. If none of them are installed, the application will exit with an error message and ask the user to install docker or podman.

### `spawn` sub command

The number of instances to spawn is passed as an argument to the spawn sub command and is mandatory. this is called `instance_count`. This will be provided as a command line argument to the spawn sub command.

The spawn suub command will also create a docker network called `qdrant_network` if it does not exist and all the containers will be connected to this network.

For example 

``` bash 
spawn-qdrant spawn 2
```

This will spawn two qdrant instances with the following ports and storage locations

- Container 1 Named qdrant-01 and uses ports 6333 and 6334 and storage location ~/.qdrant_storage01 in the network qdrant_network 
- Container 2 Named qdrant-02 and uses ports 6335 and 6336 and storage location ~/.qdrant_storage02 in the network qdrant_network

The ports are the starting port number for the qdrant instance. The actual ports used will be REST_PORT + 2*(instance_count - 1) and GRPC_PORT + 2*(instance_count - 1) if instance_count > 1. If Instance count is 1, then the ports used will be REST_PORT and GRPC_PORT.

The storage for each qdrant instance will be stored in ~/.qdrant_storage{instance_count}.

For example if instance_count is 1, then the ports used will be 6333 and 6334 and the storage will be stored in ~/.qdrant_storage01 and container will be named qdrant-01.

If instance_count is 2, then the application spawns two container with the following REST, GPRC ports and storage location 

- Container 1 Named qdrant-01 and uses ports 6333 and 6334 and storage location ~/.qdrant_storage01
- Container 2 Named qdrant-02 and uses ports 6335 = 6333 + 2*(2-1) and 6336 = 6334 + 2*(2 - 1) and storage location ~/.qdrant_storage02

### `stop` sub command

The stop sub command will stop all the qdrant instances that were spawned using the spawn sub command. The application can be used as follows  

#### `stop all` sub command

This will stop all the qdrant instances that were spawned using the spawn sub command and remove qdrant_network if there are no other containers in the network.

``` bash 
spawn-qdrant stop all 
```

#### `stop n` sub command

This will stop the n-th qdrant instance that was spawned using the spawn sub command and remove qdrant_network if there are no other containers in the network.

``` bash 
spawn-qdrant stop n
```

### `clean` sub command 

- This operation will stop first the qdrant instances that were spawned using the spawn sub command, by internally using the `stop all` sub command. 
- The application will then gzip the storage locations ~/.qdrant_storage{instance_count} into a single tar.gz file into a backup location called ~/.qdrant_backup with date time stamp. 
- The application will then delete the storage locations, ~/.qdrant_storage{instance_count},  of all the qdrant instances that were spawned using the spawn sub command. 
- Also the gzip and delete command will use sudo elavation to delete the storage locations as docker containers and storage locations are owned by root.

``` bash 
spawn-qdrant clean 
```
