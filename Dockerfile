# Build stage for frontend
FROM node:16 AS frontend-builder
WORKDIR /app/frontend-react
COPY frontend-react/package.json frontend-react/yarn.lock ./
RUN yarn install
COPY frontend-react/ ./
RUN yarn build

# Build stage for Go backend
FROM golang:1.21 AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY --from=frontend-builder /app/frontend-react/build ./frontend-react/build
RUN CGO_ENABLED=0 GOOS=linux go build -o snapcast-control .

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=go-builder /app/snapcast-control .
EXPOSE 8080
CMD ["./snapcast-control"]
