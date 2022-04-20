FROM node:16-alpine

RUN mkdir bmp
WORKDIR bmp
COPY jsdos-bmp/_site .
COPY jsdos-bmp/package.json ./package.json
RUN npm install

ENV PORT 8080
EXPOSE $PORT
ENTRYPOINT npm run start -- -p $PORT
