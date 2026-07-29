FROM node:22-alpine
WORKDIR /app
COPY package.json ./
RUN npm install
COPY server.ts tsconfig.json ./
COPY certs ./certs
EXPOSE 4001
CMD ["npm", "start"]
