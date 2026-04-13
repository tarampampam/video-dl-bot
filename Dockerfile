# Dockerfile

# ... Other stages ...

# yt-dlp stage
FROM alpine:latest as ytdlp

# Set the shell to be POSIX compliant
SHELL ["/bin/sh", "-eux"]

# Extract the version from requirements file
YT_DLP_VERSION=$(grep -E '^[^#]*yt-dlp' /tmp/requirements-ytdlp.txt | sed 's/.*==//')

# Fail-fast: check for valid YT_DLP_VERSION
if [ -z "$YT_DLP_VERSION" ] || ! echo "$YT_DLP_VERSION" | grep -qE '^[0-9A-Za-z._-]+$'; then
  echo 'Error: Invalid YT_DLP_VERSION extracted';
  exit 1;
fi

# Install wget and optionally ca-certificates
apk add --no-cache wget ca-certificates

# Download yt-dlp
wget https://github.com/yt-dlp/yt-dlp/releases/download/${YT_DLP_VERSION}/yt-dlp -O /bin/yt-dlp && chmod +x /bin/yt-dlp

# ... Rest of the Dockerfile ...