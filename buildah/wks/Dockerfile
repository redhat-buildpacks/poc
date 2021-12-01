FROM registry.access.redhat.com/ubi8

LABEL runtime="nodejs"
LABEL version="1.14"

RUN dnf install -y --setopt=tsflags=nodocs nodejs && \
	rpm -V nodejs && \
	dnf -y clean all