# syntax=docker/dockerfile:1

# -✂- this is the base image with python installed --------------------------------------------------------------------
FROM docker.io/library/python:3.13-slim-bookworm AS python

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
    && YT_DLP_VERSION="2025.05.22" \
    && wget -O /bin/yt-dlp "https://github.com/yt-dlp/yt-dlp/releases/download/${YT_DLP_VERSION}/yt-dlp" \
    && chmod +x /bin/yt-dlp

# -✂- this stage is used to develop and build the app -----------------------------------------------------------------
FROM python AS develop

# install Go using the official image and copy all required binaries
COPY --from=public.ecr.aws/docker/library/golang:1.24-bookworm /usr/local/go /usr/local/go
COPY --from=ffmpeg /bin/ffmpeg /bin/ffprobe /bin/
COPY --from=yt-dlp /bin/yt-dlp /bin/yt-dlp

ENV \
  # add Go binaries to the PATH
  PATH="$PATH:/go/bin:/usr/local/go/bin" \
  # use the /var/tmp/go as the GOPATH to reuse the modules cache
  GOPATH="/var/tmp/go" \
  # set path to the Go cache (think about this as a "object files cache")
  GOCACHE="/var/tmp/go/cache"

RUN set +x \
    # precompile the standard library
    && go build std \
    # allow anyone to read/write the Go cache
    && find /var/tmp/go -type d -exec chmod 0777 {} + \
    && find /var/tmp/go -type f -exec chmod 0666 {} +

RUN set -x \
    # ensure everything is installed correctly
    && go version \
    && python3 --version \
    && ffmpeg -version \
    && ffprobe -version \
    && yt-dlp --version

# -✂- this stage is used to compile the application -------------------------------------------------------------------
# later you can copy the compiled binary from this image step like this:
# COPY --from=compiler /src/video-dl-bot /bin/video-dl-bot
FROM develop AS compiler

# can be passed with any prefix (like `v1.2.3@FOO`), e.g.: `docker build --build-arg "APP_VERSION=v1.2.3@FOO" .`
ARG APP_VERSION="undefined@docker"

# copy the source code
COPY . /src
WORKDIR /src

RUN set -x \
    # build the app itself
    && go generate -skip readme ./... \
    && CGO_ENABLED=0 go build \
      -trimpath \
      -ldflags "-s -w -X gh.tarampamp.am/video-dl-bot/internal/version.version=${APP_VERSION}" \
      -o ./video-dl-bot \
      ./cmd/video-dl-bot/ \
    && ./video-dl-bot --help

# -✂- and this is the final stage -------------------------------------------------------------------------------------
FROM python AS runtime

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

COPY --from=ffmpeg /bin/ffmpeg /bin/ffprobe /bin/
COPY --from=yt-dlp /bin/yt-dlp /bin/yt-dlp
COPY --from=compiler /src/video-dl-bot /bin/video-dl-bot

# prepare the rootfs for scratch
RUN set -x \
    && echo 'video-dl-bot:x:10001:10001::/tmp:/sbin/nologin' >> /etc/passwd \
    && echo 'video-dl-bot:x:10001:' >> /etc/group

# use an unprivileged user
USER 10001:10001

WORKDIR /tmp

ENTRYPOINT ["/bin/video-dl-bot"]
