FROM ubuntu

RUN apt-get update && apt-get -y install ca-certificates
RUN mkdir /opt/nalivator

COPY nalivator9000 /opt/nalivator/
COPY config.json /opt/nalivator/

