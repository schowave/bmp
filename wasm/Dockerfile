# Build-Kontext ist das Repository-Wurzelverzeichnis:
#   docker build -f wasm/Dockerfile .
# Das Bundle entsteht hier im Build, damit lokal und in CI derselbe Weg laeuft.

FROM golang:1.27-alpine AS server
WORKDIR /src
COPY wasm/server.go .
RUN CGO_ENABLED=0 go build -o server server.go

FROM alpine:3 AS bundle
RUN apk add --no-cache bash zip
WORKDIR /src
COPY bmp bmp
COPY web web
COPY VERSION VERSION
COPY wasm/build-bundle.sh wasm/dosbox.conf wasm/index.html wasm/
RUN wasm/build-bundle.sh /public

FROM alpine:3
COPY --from=server /src/server /server
COPY --from=bundle /public /public
VOLUME /data
EXPOSE 8080
CMD ["/server"]
