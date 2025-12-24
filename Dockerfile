FROM node:18-alpine AS web-builder
WORKDIR /web
COPY web/package*.json ./
RUN npm ci
COPY web ./
RUN npm run build

FROM golang:1.24-alpine AS go-builder
WORKDIR /build/server
ENV GOPROXY=https://goproxy.cn,direct
ARG TARGETOS
ARG TARGETARCH
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server ./
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH
RUN go build -o /build/one-mcp ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata nodejs npm python3 py3-pip bash git curl
WORKDIR /app/server
COPY --from=go-builder /build/one-mcp ./one-mcp
COPY --from=web-builder /web/dist /app/web/dist
EXPOSE 8080
ENV GIN_MODE=release
ENTRYPOINT ["./one-mcp"]
