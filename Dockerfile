
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.* ./ 
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o medley .

FROM python:3.12-alpine

RUN apk add --no-cache \
    ffmpeg \
    imagemagick \
    --repository=https://dl-cdn.alpinelinux.org/alpine/edge/testing/ \
    atomicparsley

RUN pip install --no-cache-dir --default-timeout=100 \
    yt-dlp \
    eyeD3

WORKDIR /app

COPY --from=builder /app/medley /usr/local/bin/medley
COPY --from=builder /app/LICENSE /usr/share/doc/medley/LICENSE

# Verify everything is accessible and working in the environment
RUN yt-dlp --version && \
    ffmpeg -version && \
    magick -version && \
    eyeD3 --version && \
    AtomicParsley -v && \
    which medley

ENTRYPOINT ["medley"]

