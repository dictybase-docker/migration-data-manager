FROM golang:1.5.1-onbuild
MAINTAINER Siddhartha Basu<siddhartha-basu@northwestern.edu>

# Add the prestop hook(kubernetes container lifecycle)
ADD hook.sh /
