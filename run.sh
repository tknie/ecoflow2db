#!/bin/bash

LOGPATH=$(pwd)/logs
export LOGPATH

rm -f ecoflow2db.log $LOGPATH/*
go run ./cmd/ecoflow2db 
