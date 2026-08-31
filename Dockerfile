FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.* ./ 
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o medley .

FROM rust:bookworm AS rust-builder

WORKDIR /app

RUN apt-get update && apt-get install -y libc6-dev jq

RUN curl -sSLkf $(curl -sSLkf https://api.github.com/repos/jim60105/bgutil-ytdlp-pot-provider-rs/releases/latest | jq -r '.tarball_url') -o bgutil-pot.tar.gz \
    && tar -xzf bgutil-pot.tar.gz --strip-components=1 \
    && rm bgutil-pot.tar.gz

RUN RUSTFLAGS="-C target-feature=+crt-static" \
    cargo build \
    -p bgutil-ytdlp-pot-provider \
    --bin bgutil-pot \
    --features ffi,vendored-openssl \
    --release \
    --target x86_64-unknown-linux-gnu

FROM python:3.12-alpine

RUN apk add --no-cache ca-certificates ffmpeg imagemagick \
 && apk add --no-cache --repository=https://dl-cdn.alpinelinux.org/alpine/edge/community deno \
 && apk add --no-cache --repository=https://dl-cdn.alpinelinux.org/alpine/edge/testing/ atomicparsley

RUN pip install --no-cache-dir --default-timeout=100 \
    yt-dlp \
    yt-dlp-ejs \
    eyeD3

COPY --from=builder /app/medley /usr/local/bin/medley
COPY --from=builder /app/LICENSE /usr/share/doc/medley/LICENSE

COPY --from=rust-builder /app/target/x86_64-unknown-linux-gnu/release/bgutil-pot /usr/local/bin/bgutil-pot
COPY --from=rust-builder /app/plugin/yt_dlp_plugins /etc/yt-dlp/plugins/bgutil-ytdlp-pot-provider/yt_dlp_plugins

WORKDIR /app

RUN chown -R nobody:nobody /app
ENV HOME=/app

USER nobody:nobody

RUN yt-dlp --version && \
    deno --version && \
    ffmpeg -version && \
    magick -version && \
    eyeD3 --version && \
    AtomicParsley -v && \
    which medley

ENTRYPOINT ["medley"]

