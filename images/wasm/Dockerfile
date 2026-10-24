# The build context is the repository root:
#   docker build -f images/wasm/Dockerfile .
# The bundle is built here in the image build, so local and CI take the same path.

FROM golang:1.27-alpine AS server
WORKDIR /src
COPY images/wasm/go.mod images/wasm/server.go ./
RUN CGO_ENABLED=0 go build -o server .

FROM alpine:3 AS bundle
RUN apk add --no-cache bash zip
WORKDIR /src
COPY bmp bmp
COPY images/shared images/shared
COPY VERSION VERSION
COPY images/wasm/build-bundle.sh images/wasm/dosbox.conf images/wasm/index.html images/wasm/
RUN images/wasm/build-bundle.sh /public

FROM alpine:3
# Unprivileged like the VNC image, with the same uid 1000. /data must be writable
# for it, see images/wasm/docker-compose.yml.
RUN adduser -D -H -u 1000 bmp \
    && mkdir /data \
    && chown bmp:bmp /data
COPY --from=server /src/server /server
COPY --from=bundle /public /public
VOLUME /data
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -q -O /dev/null http://localhost:8080/version.txt || exit 1
USER bmp
CMD ["/server"]
