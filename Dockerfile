FROM golang:1.5.1-onbuild
MAINTAINER Siddhartha Basu<siddhartha-basu@northwestern.edu>

# install curl
RUN apt-get update \
    && apt-get -y install curl \
    && rm -rf /var/lib/apt/lists/* 

# Add the prestop hook(kubernetes container lifecycle)
ADD hook.sh /
