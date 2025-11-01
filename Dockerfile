# syntax=docker/dockerfile:1

# -✂- this stage is used to download the latest version of FFmpeg -----------------------------------------------------
# later you can copy FFmpeg binaries from this image step like this (glibc is required):
# COPY --from=ffmpeg /bin/ffmpeg /bin/ffprobe /bin/
FROM docker.io/library/alpine:latest AS ffmpeg

RUN set -x \
    && APK_ARCH="$(apk --print-arch)" \
    # https://github.com/yt-dlp/FFmpeg-Builds/releases/tag/latest
    && case "${APK_ARCH}" in \
        x86_64)  FFMPEG_FILE_NAME="ffmpeg-master-latest-linux64-gpl.tar.xz" ;; \
        aarch64) FFMPEG_FILE_NAME="ffmpeg-master-latest-linuxarm64-gpl.tar.xz" ;; \
        *) echo >&2 "error: unsupported architecture: ${APK_ARCH}"; exit 1 ;; \
    esac \
    && wget -O /tmp/ffmpeg.tar.xz "https://github.com/yt-dlp/FFmpeg-Builds/releases/download/latest/${FFMPEG_FILE_NAME}" \
    && mkdir -p /tmp/ffmpeg \
    && tar -xf /tmp/ffmpeg.tar.xz -C /tmp/ffmpeg --strip-components=1 \
    && mv /tmp/ffmpeg/bin/ffmpeg /tmp/ffmpeg/bin/ffprobe /bin/ \
    && chmod +x /bin/ffmpeg /bin/ffprobe \
    && chown root:root /bin/ffmpeg /bin/ffprobe \
    && rm -rf /tmp/ffmpeg /tmp/ffmpeg.tar.xz

# -✂- this stage is used to download the required version of yt-dlp ---------------------------------------------------
# later you can copy yt-dlp binary from this image step like this (installed python 3.9+ is required):
# COPY --from=yt-dlp /bin/yt-dlp /bin/yt-dlp
FROM docker.io/library/alpine:latest AS yt-dlp

RUN set -x \
    # renovate: source=github-tags name=yt-dlp/yt-dlp
    && YT_DLP_VERSION="2025.10.22" \
    && wget -O /bin/yt-dlp "https://github.com/yt-dlp/yt-dlp/releases/download/${YT_DLP_VERSION}/yt-dlp" \
    && chmod +x /bin/yt-dlp

# -✂- this stage is used to compile the application -------------------------------------------------------------------
# later you can copy the compiled binary (with all the required files) from this image step like this:
FROM docker.io/library/golang:1.25-alpine AS compiler

# copy the source code
COPY . /src
WORKDIR /src

# can be passed with any prefix (like `v1.2.3@FOO`), e.g.: `docker build --build-arg "APP_VERSION=v1.2.3@FOO" .`
ARG APP_VERSION="undefined@docker"

RUN set -x \
    # build the app
    && go generate -skip readme ./... \
    && CGO_ENABLED=0 go build \
      -trimpath \
      -ldflags "-s -w -X gh.tarampamp.am/video-dl-bot/internal/version.version=${APP_VERSION}" \
      -o ./video-dl-bot \
      ./cmd/video-dl-bot/ \
    && ./video-dl-bot --help \
    # prepare rootfs for runtime
    && mkdir -p /tmp/rootfs \
    && cd /tmp/rootfs \
    && mkdir -p ./etc ./bin \
    && echo 'video-dl-bot:x:10001:10001::/nonexistent:/sbin/nologin' > ./etc/passwd \
    && echo 'video-dl-bot:x:10001:' > ./etc/group \
    && mv /src/video-dl-bot ./bin/video-dl-bot

# -✂- and this is the final stage -------------------------------------------------------------------------------------
FROM docker.io/library/python:3.14-slim AS runtime

COPY --from=ffmpeg /bin/ffmpeg /bin/ffprobe /bin/
COPY --from=yt-dlp /bin/yt-dlp /bin/yt-dlp
COPY --from=compiler /tmp/rootfs/bin/video-dl-bot /bin/video-dl-bot
COPY --from=compiler /tmp/rootfs/etc/passwd /tmp/rootfs/etc/group /etc/

ARG APP_VERSION="undefined@docker"

LABEL \
    # Docs: <https://github.com/opencontainers/image-spec/blob/master/annotations.md>
    org.opencontainers.image.title="video-dl-bot" \
    org.opencontainers.image.description="Telegram bot for downloading videos" \
    org.opencontainers.image.url="https://github.com/tarampampam/video-dl-bot" \
    org.opencontainers.image.source="https://github.com/tarampampam/video-dl-bot" \
    org.opencontainers.image.vendor="tarampampam" \
    org.opencontainers.version="$APP_VERSION" \
    org.opencontainers.image.licenses="MIT"

USER 10001:10001
WORKDIR /tmp
ENV HOME=/tmp LOG_FORMAT=json LOG_LEVEL=info PID_FILE=/tmp/video-dl-bot.pid
ENTRYPOINT ["/bin/video-dl-bot"]
