#!/usr/bin/env sh

# Copy the file with cookies if it is set through environment variables, to
# avoid issues with read-only mounted secrets like this:
#
# File \"/usr/bin/yt-dlp/__main__.py\", line 17, in <module>;
# ...
# with open(file, 'w' if write else 'r', encoding='utf-8')
# OSError: [Errno 30] Read-only file system: '/cookies.txt'

if [ -n "$COOKIES_FILE" ]; then
  cp "$COOKIES_FILE" /tmp/cookies.txt;
  export COOKIES_FILE="/tmp/cookies.txt";
fi;

exec "$@"
