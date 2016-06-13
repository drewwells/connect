FROM ubuntu:xenial

RUN apt-get update && \
    apt-get install -y openconnect openssh-server netcat-traditional ocproxy dnsutils && \
    apt-get clean && \
    rm -rf /var/cache/apt/* && \
    rm -rf /var/lib/apt/lists/*

ENV PASS=mustspecifypass
ENV USER=mustspecifyuser
ENV SERVER=mustspecifyvpnserver.com

ENTRYPOINT echo "Logging in as $USER"; echo $PASS | openconnect --non-inter $SERVER --user=$USER --passwd-on-stdin
