FROM ubuntu:latest

RUN  apt-get update && apt-get -y upgrade

COPY ecoflow2db /ecoflow2db/bin/
COPY docker-entrypoint.sh /

RUN chmod a+x docker-entrypoint.sh && \
   mkdir /ecoflow2db/log && chmod 777 /ecoflow2db/log

ENV LOGFILE=/ecoflow2db/log/

CMD ["/docker-entrypoint.sh"]
