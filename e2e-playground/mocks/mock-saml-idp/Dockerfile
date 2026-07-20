FROM node:22-alpine
WORKDIR /app
COPY package.json ./
RUN npm install
COPY server.ts ./
COPY certs ./certs
EXPOSE 4001
CMD ["npm", "start"]
