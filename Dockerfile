FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.* ./ 
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o medley .

# FROM denoland/deno:alpine AS deno-image

FROM python:3.12-alpine

# COPY --from=deno-image /bin/deno /usr/local/bin/deno

RUN apk add --no-cache \
    --repository=https://dl-cdn.alpinelinux.org/alpine/edge/community deno

RUN apk add --no-cache \
    ffmpeg \
    imagemagick \
    --repository=https://dl-cdn.alpinelinux.org/alpine/edge/testing/ atomicparsley

RUN pip install --no-cache-dir --default-timeout=100 \
    yt-dlp \
    yt-dlp-ejs \
    eyeD3

COPY --from=builder /app/medley /usr/local/bin/medley
COPY --from=builder /app/LICENSE /usr/share/doc/medley/LICENSE

WORKDIR /app

RUN chown -R nobody:nobody /app
ENV HOME=/app

USER nobody:nobody

# Verify everything is accessible and working as 'nobody'
RUN yt-dlp --version && \
    deno --version && \
    ffmpeg -version && \
    magick -version && \
    eyeD3 --version && \
    AtomicParsley -v && \
    which medley

ENTRYPOINT ["medley"]

