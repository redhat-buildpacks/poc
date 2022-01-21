ARG base_image
FROM ${base_image}

RUN apt-get update && apt-get install -y openssl ca-certificates

RUN openssl genrsa -des3 -passout pass:"foobarpwd" -out server.pass.key 2048 && \
    openssl rsa -passin pass:"foobarpwd" -in server.pass.key -out server.key && \
    rm server.pass.key && \
    openssl req -new -key server.key -out server.csr -subj "/C=BE/ST=Namur/L=Florennes/O=CNCF/OU=Buildpack/CN=buildpack.io" && \
    openssl x509 -req -days 365 -in server.csr -signkey server.key -out server.crt

RUN cp server.crt /usr/local/share/ca-certificates

RUN update-ca-certificates