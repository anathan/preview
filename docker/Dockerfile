# Dockerfile to install preview
# VERSION 1 - EDITION 1
FROM ubuntu
MAINTAINER Chris Antenesse, chris@antenesse.net

RUN apt-get install -y wget unzip

RUN mkdir -p /usr/local/preview/bin /usr/local/preview/conf

RUN wget https://github.com/ngerakines/preview/releases/download/v0.1.0/preview-0.1.0-linux_amd64.zip -O /usr/local/preview/bin/preview.zip
 
RUN cd /usr/local/preview/bin && unzip /usr/local/preview/bin/preview.zip && rm -f /usr/local/preview/bin/preview.zip

RUN useradd preview

RUN su preview -c '/usr/local/preview/bin/preview' &
