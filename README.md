# Ecoflow2db application

## Introduction

This application named Ecoflow2db is used to query Ecoflow API getting current configuration and Solar data information and generate corresponding database tables in the Postgres database.

Ecoflow2db queries periodically data from all devices quota API calls and store the data. The data is display in Grafana as an example.

Current Ecoflow devices are in use at the moment:

- Ecoflow Powerstream inverter
- Ecoflow Delta 2

In advance the application can listen for MQTT events. Unfortunately the MQTT is only active during usage of the Ecoflow App on mobile phone or smartphone device. The corresponding handling is not implemented yet.

## Environment variables

Variables | Default | Description
---------|----------|---------
 ECOFLOW_USER |  | C1
 ECOFLOW_PASSWORD |  | C2
 ECOFLOW_DEVICE_SN |  | C3
 ECOFLOW_ACCESS_KEY |  | C3
 ECOFLOW_SECRET_KEY |  | C3
 ECOFLOW_DB_URL |  | C3
 ECOFLOW_DB_USER |  | C3
 ECOFLOW_DB_PASS |  | C3
 LOGPATH |  | C3
 ECOFLOW2DB_WAIT_SECONDS | 30 | C3

## Docker environment

The Ecoflow2db application and corresponding Postgres database is running in an Raspberry Pi.

Docker images are on Docker hub at

```docker
docker pull thknie/ecoflow2db:tagname
```

See the example script showing how to start the service with podman. Located is the script in this repostiory at [docker/podstart.sh](docker/podstart.sh).

______________________
These tools are provided as-is and without warranty or support. Users are free to use, fork and modify them, subject to the license agreement.
