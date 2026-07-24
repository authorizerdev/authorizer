FROM node:22-alpine
WORKDIR /app
COPY package.json ./
RUN npm install
COPY server.ts tsconfig.json ./
EXPOSE 4100
CMD ["npm", "start"]
