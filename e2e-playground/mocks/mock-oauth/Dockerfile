FROM node:22-alpine
WORKDIR /app
COPY package.json ./
RUN npm install
COPY server.ts ./
EXPOSE 4000
CMD ["npm", "start"]
