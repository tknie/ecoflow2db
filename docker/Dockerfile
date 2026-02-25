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

FROM alpine:latest

RUN  apk update && apk upgrade && apk add gcompat

COPY ecoflow2db /ecoflow2db/bin/
COPY docker-entrypoint.sh /

RUN chmod a+x docker-entrypoint.sh && \
   mkdir /ecoflow2db/log && chmod 777 /ecoflow2db/log

ENV LOGFILE=/ecoflow2db/log/

CMD ["/docker-entrypoint.sh"]
