# Ecsbeat

Welcome to Ecsbeat.

Ensure env variables GOPATH and GOROOT are set properly, and this folder is at the following location:
`${GOPATH}/github.com/yangb8`

## Getting Started with Ecsbeat

### Requirements

* [Golang](https://golang.org/dl/) 1.7
* [glide](https://github.com/Masterminds/glide)

### Init Project
To get running with Ecsbeat and also install the
dependencies, run the following command:

```
make setup
```

It will create a clean git history for each major step. Note that you can always rewrite the history if you wish before pushing your changes.

To push Ecsbeat in the git repository, run the following commands:

```
git remote set-url origin https://github.com/yangb8/ecsbeat
git push origin master
```

For further development, check out the [beat developer guide](https://www.elastic.co/guide/en/beats/libbeat/current/new-beat.html).

### Build

To build the binary for Ecsbeat run the command below. This will generate a binary
in the same directory with the name ecsbeat.

```
make
```

### Run

To run Ecsbeat with debugging output enabled, run:

```
modify ecsbeat.yml based on your ecs env
./ecsbeat -c ecsbeat.yml -e -d "*"
```


### Test

To test Ecsbeat, run the following command:

```
make testsuite
```

alternatively:
```
make unit-tests
make system-tests
make integration-tests
make coverage-report
```

The test coverage is reported in the folder `./build/coverage/`

### Update

Each beat has a template for the mapping in elasticsearch and a documentation for the fields
which is automatically generated based on `etc/fields.yml`.
To generate etc/ecsbeat.template.json and etc/ecsbeat.asciidoc

```
make update
```


### Cleanup

To clean  Ecsbeat source code, run the following commands:

```
make fmt
make simplify
```

To clean up the build directory and generated artifacts, run:

```
make clean
```


### Clone

To clone Ecsbeat from the git repository, run the following commands:

```
mkdir -p ${GOPATH}/github.com/yangb8
cd ${GOPATH}/github.com/yangb8
git clone https://github.com/yangb8/ecsbeat
```


For further development, check out the [beat developer guide](https://www.elastic.co/guide/en/beats/libbeat/current/new-beat.html).


## Docker

after build is done
```make docker-image```


## Packaging

The beat frameworks provides tools to crosscompile and package your beat for different platforms. This requires [docker](https://www.docker.com/) and vendoring as described above. To build packages of your beat, run the following command:

```
make package
```

This will fetch and create all images required for the build process. The hole process to finish can take several minutes.

## Example of Common Fields in Output
```
  "ecs-customer": "EMC",
  "ecs-event-type": "disks",
  "ecs-node-ip": "10.1.83.51",                  # only if the event is on node level
  "ecs-node-name": "ecs-obj-1-1.plylab.local",  # only if the event is on node level
  "ecs-vdc-cfgname": "VDC1",
  "ecs-vdc-id": "urn:storageos:VirtualDataCenterData:407b6b6c-bda4-4ba4-89f7-220ac3d9c044",
  "ecs-vdc-name": "plylab",
  "ecs-version": "3.0.0.0.86239.1c9e5ec",
```
