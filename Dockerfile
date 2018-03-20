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

ADD ./cert/key.pem /jackal/cert/key.pem
ADD ./cert/cert.pem /jackal/cert/cert.pem

ADD docker.jackal.yml /etc/jackal/jackal.yml

EXPOSE 5222

ENTRYPOINT ["/jackal/jackal"]
