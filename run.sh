#!/bin/bash

#
# Copyright 2025-2026 Thorsten A. Knieling
#
# SPDX-License-Identifier: Apache-2.0
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#       http://www.apache.org/licenses/LICENSE-2.0
#

LOGPATH=$(pwd)/logs
export LOGPATH

rm -f ecoflow2db.log $LOGPATH/*
#go run ./cmd/ecoflow2db 
docker/ecoflow2db $*
