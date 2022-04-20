FROM node:17-alpine

WORKDIR site

COPY jsdos-bmp/_site .
COPY jsdos-bmp/package.json .
RUN npm install

ENTRYPOINT npm run start

