FROM node:16-alpine

COPY jsdos-bmp/_site .
COPY jsdos-bmp/package.json .
RUN npm install

ENV PORT 8080
ENTRYPOINT npm run start -- -p $PORT
