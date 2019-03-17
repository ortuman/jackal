FROM debian:stretch-slim

LABEL org.label-schema.version=latest
LABEL org.label-schema.vcs-url="https://github.com/ortuman/jackal.git"
LABEL org.label-schema.name="jackal"
LABEL org.label-schema.vendor="Miguel Ángel Ortuño"
LABEL maintainer="Miguel Ángel Ortuño <ortuman@protonmail.com>"

ADD dockerfiles/jackal.yml /etc/jackal/jackal.yml
ADD jackal /
EXPOSE 5222
CMD ["./jackal"]
