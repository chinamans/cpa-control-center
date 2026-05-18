FROM node:22-alpine AS frontend
WORKDIR /src/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.24-bookworm AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN rm -rf cmd/web/public && mkdir -p cmd/web/public
COPY --from=frontend /src/frontend/dist/ ./cmd/web/public/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/cpa-control-center ./cmd/web

FROM debian:bookworm-slim
RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates tzdata \
  && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=backend /out/cpa-control-center /app/cpa-control-center
ENV CPA_CONTROL_CENTER_ADDR=:8080 \
    CPA_CONTROL_CENTER_DATA_DIR=/data
VOLUME ["/data"]
EXPOSE 8080
ENTRYPOINT ["/app/cpa-control-center"]
