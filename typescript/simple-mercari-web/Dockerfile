FROM node:20-alpine
WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .

RUN npm run build

RUN addgroup -S mercari && adduser -S trainee -G mercari
RUN chown -R trainee:mercari /app && chmod -R 755 /app
USER trainee

EXPOSE 3000

CMD ["npm", "start", "--", "--host", "0.0.0.0"]