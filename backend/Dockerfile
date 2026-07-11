# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.23-alpine AS build
WORKDIR /src
# Cache modules first (no external deps yet, but keeps the layer stable).
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# ---- runtime stage ----
# alpine (not scratch) so ffmpeg/ffprobe is available for Milestone 2 media analysis.
FROM alpine:3.20
RUN apk add --no-cache ffmpeg ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/server /app/server
EXPOSE 8080
ENTRYPOINT ["/app/server"]
