FROM node:20-alpine

WORKDIR /app

COPY package*.json ./

RUN npm install

# Install dependencies for web support
RUN npx expo install react-dom react-native-web @expo/metro-runtime

COPY . .

EXPOSE 8081

# Start Expo for web
CMD ["npx", "expo", "start", "--web", "--host", "lan"]
