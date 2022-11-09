FROM --platform=$BUILDPLATFORM golang:1.19.3-bullseye as stage

ARG TARGETARCH

LABEL org.label-schema.vcs-url="https://github.com/ortuman/jackal.git"
LABEL org.label-schema.name="jackal"
LABEL org.label-schema.vendor="Miguel Ángel Ortuño"
LABEL maintainer="Miguel Ángel Ortuño <ortuman@protonmail.com>"

ADD . /src
WORKDIR /src

RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -a -tags netgo -ldflags "-s -w" "github.com/ortuman/jackal/cmd/jackal"
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -a -tags netgo -ldflags "-s -w" "github.com/ortuman/jackal/cmd/jackalctl"
RUN chmod +x jackal jackalctl

FROM --platform=$BUILDPLATFORM debian:stable-slim

RUN apt-get update && apt-get install -y ca-certificates
RUN update-ca-certificates

WORKDIR /jackal

COPY --from=stage /src/jackal /jackal
COPY --from=stage /src/jackalctl /jackal

EXPOSE 5222

ENV PATH $PATH:/jackal

CMD ["./jackal"]
