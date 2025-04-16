FROM alpine:latest

RUN  apk update && apk upgrade && apk add gcompat

COPY ecoflow2db /ecoflow2db/bin/
COPY docker-entrypoint.sh /

RUN chmod a+x docker-entrypoint.sh && \
   mkdir /ecoflow2db/log && chmod 777 /ecoflow2db/log

ENV LOGFILE=/ecoflow2db/log/

CMD ["/docker-entrypoint.sh"]
