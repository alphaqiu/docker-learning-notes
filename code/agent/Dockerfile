FROM ubuntu:latest
MAINTAINER alpha <alphaqiu@gmail.com>


WORKDIR /opt
RUN mkdir -p /opt/scripts
COPY prerequest.sh /opt/scripts/prerequest.sh
RUN chmod +x /opt/scripts/prerequest.sh

ENTRYPOINT ["/opt/scripts/prerequest.sh"]