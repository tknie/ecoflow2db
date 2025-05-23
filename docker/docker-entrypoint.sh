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

# default log level set to info
ENABLE_DEBUG=${ENABLE_DEBUG:-info}
export ENABLE_DEBUG

EXITCODE=0

#
# Clean up function to start shutdown of databases
#
function clean_up {
   echo $(date +"%Y-%m-%d %H:%m:%S")" Clean up environment"
   exit ${EXITCODE}
}

trap clean_up EXIT HUP INT TERM SIGHUP SIGINT SIGTERM QUIT

/ecoflow2db/bin/ecoflow2db
EXITCODE=$?

exit ${EXITCODE}
