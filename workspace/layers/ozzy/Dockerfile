FROM redhat/ubi8-minimal:latest

# TODO: why wasn't this needed before?
USER root

ARG some_arg
RUN echo ${some_arg} > /opt/arg.txt

RUN touch /file-from-first-extension.txt

ARG build_id=0
RUN echo ${build_id}

RUN microdnf --setopt=install_weak_deps=0 --setopt=tsflags=nodocs install -y java curl && microdnf clean all && rpm -q curl && java -version
