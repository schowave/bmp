FROM node:16-alpine

COPY jsdos-bmp/_site .
COPY jsdos-bmp/package.json .
RUN npm install

ENTRYPOINT npm run start

