FROM golang:1.10

LABEL org.label-schema.version=latest
LABEL org.label-schema.vcs-url="https://github.com/ortuman/jackal.git"
LABEL org.label-schema.name="jackal"
LABEL org.label-schema.vendor="Miguel Ángel Ortuño"
LABEL maintainer="Miguel Ángel Ortuño <ortuman@protonmail.com>"

WORKDIR /jackal

RUN apt-get update
RUN apt-get install -y --no-install-recommends libidn11-dev

RUN go get -u github.com/ortuman/jackal
RUN go build github.com/ortuman/jackal

RUN openssl genrsa -out server.key 2048
RUN openssl req -new -x509 -key server.key -out server.crt -days 365 -subj "/C=CN/ST=Madrid/L=Madrid/O=Me/OU=Me/CN=localhost"

ADD docker.jackal.yml /etc/jackal/jackal.yml

EXPOSE 5222

ENTRYPOINT ["/jackal/jackal"]
