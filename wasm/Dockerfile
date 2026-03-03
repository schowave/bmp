FROM golang:1.23-alpine AS build
WORKDIR /src
COPY server.go .
RUN CGO_ENABLED=0 go build -o server server.go

FROM alpine:3
COPY --from=build /src/server /server
COPY index.html bmp.jsdos version.txt favicon.png /public/
VOLUME /data
EXPOSE 8080
CMD ["/server"]
