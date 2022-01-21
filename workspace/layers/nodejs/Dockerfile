FROM registry.access.redhat.com/ubi8

ARG runtime
ARG version

RUN echo ${runtime}:${version}

RUN yum -y module enable ${runtime}:${version} && \
    MODULE_DEPS="make gcc gcc-c++ libatomic_ops git openssl-devel" && \
    INSTALL_PKGS="$MODULE_DEPS nodejs npm nodejs-nodemon nss_wrapper" && \
    ln -s /usr/lib/node_modules/nodemon/bin/nodemon.js /usr/bin/nodemon && \
    ln -s /usr/libexec/platform-python /usr/bin/python3 && \
    yum install -y --setopt=tsflags=nodocs $INSTALL_PKGS && \
    rpm -V $INSTALL_PKGS && \
    yum -y clean all --enablerepo='*'