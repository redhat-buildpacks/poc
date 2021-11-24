FROM registry.access.redhat.com/ubi8

LABEL runtime="nodejs"
LABEL version="1.14"

RUN yum install -y --setopt=tsflags=nodocs nodejs && \
	rpm -V nodejs && \
	yum -y clean all