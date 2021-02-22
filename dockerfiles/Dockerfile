FROM debian:stretch-slim

WORKDIR /jackal

LABEL org.label-schema.vcs-url="https://github.com/ortuman/jackal.git"
LABEL org.label-schema.name="jackal"
LABEL org.label-schema.vendor="Miguel Ángel Ortuño"
LABEL maintainer="Miguel Ángel Ortuño <ortuman@protonmail.com>"

ADD build/jackal /jackal
ADD build/jackalctl /jackal

EXPOSE 5222

ENV PATH $PATH:/jackal

CMD ["./jackal"]
