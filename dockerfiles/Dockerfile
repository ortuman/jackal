FROM golang:1.11 as buildimage

LABEL org.label-schema.version=latest
LABEL org.label-schema.vcs-url="https://github.com/ortuman/jackal.git"
LABEL org.label-schema.name="jackal"
LABEL org.label-schema.vendor="Miguel Ángel Ortuño"
LABEL maintainer="Miguel Ángel Ortuño <ortuman@protonmail.com>"

WORKDIR /jackal

RUN go get -d github.com/ortuman/jackal

RUN export CGO_ENABLED=0
RUN export GOOS=linux
RUN export GOARCH=amd64
RUN export GO111MODULE=on
RUN go build -ldflags="-s -w" github.com/ortuman/jackal

FROM debian:stretch-slim
COPY --from=buildimage /jackal/jackal /
ADD docker.jackal.yml /etc/jackal/jackal.yml
EXPOSE 5222
CMD ["./jackal"]
