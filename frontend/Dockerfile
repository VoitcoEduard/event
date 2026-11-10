FROM node:26-alpine3.24

WORKDIR /app

COPY package*.json ./

RUN npm ci

COPY . .

RUN chown -R node:node /app

USER node

EXPOSE 8081

CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0", "--port", "8081"]
