#!/bin/sh

#
# Copyright 2025 Thorsten A. Knieling
#
# SPDX-License-Identifier: Apache-2.0
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#       http://www.apache.org/licenses/LICENSE-2.0
#

LOGFILE=/ecoflow2db/log/
export LOGFILE

/ecoflow2db/bin/ecoflow2db
