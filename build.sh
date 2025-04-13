#!/bin/bash

VERSION=${1:-1.0.0}
PACKAGE=github.com/tknie/ecoflow2db
DATE=$(date +%d-%m-%Y'_'%H:%M:%S)

go build -ldflags "-X ${PACKAGE}.Version=${VERSION} -X ${PACKAGE}.BuildDate=${DATE}" -o docker/ecoflow2db ./cmd/ecoflow2db
