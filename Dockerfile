FROM registry.access.redhat.com/ubi8:8.4-211

RUN yum install -y --setopt=tsflags=nodocs nodejs && \
	rpm -V nodejs && \
	yum -y clean all