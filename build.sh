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

VERSION=${VERSION:-v1.0.0}
PACKAGE=github.com/tknie/ecoflow2db
DATE=$(date +%d-%m-%Y'_'%H:%M:%S)

go build -ldflags "-X ${PACKAGE}.Version=${VERSION} -X ${PACKAGE}.BuildDate=${DATE}" -o docker/ecoflow2db ./cmd/ecoflow2db
