<p align="center">
  <a href="https://github.com/tarampampam/video-dl-bot#readme">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="https://socialify.git.ci/tarampampam/video-dl-bot/image?description=1&font=Raleway&forks=1&issues=1&owner=1&pulls=1&pattern=Solid&stargazers=1&theme=Dark">
      <img align="center" src="https://socialify.git.ci/tarampampam/video-dl-bot/image?description=1&font=Raleway&forks=1&issues=1&owner=1&pulls=1&pattern=Solid&stargazers=1&theme=Light">
    </picture>
  </a>
</p>

# video-dl-bot

A Telegram bot for downloading videos from various platforms directly within Telegram. Built with Go and
powered by `yt-dlp`, it supports downloading from any platform that `yt-dlp` supports.

## 🔥 Features

- **Universal Video Download**: Download videos from any platform supported by `yt-dlp`
  (YouTube, TikTok, Instagram, Twitter, and [hundreds more][yt-dlp-supported-sites])
- **Smart File Handling**:
  - Videos under 50 MB are sent directly in chat
  - Larger files are automatically uploaded to [filebin.net](https://filebin.net) with a direct download link
- **Link Extraction**: Simply send or forward a message with a video link - no commands needed
- **Visual Feedback**: The bot uses message reactions and status updates (e.g., "recording video") to show progress
- **Concurrent Download Limiting**: Prevents resource overuse with configurable parallel download limits
- **Cookie Support**: Authenticate with services like YouTube to bypass rate limits and access restricted content

[yt-dlp-supported-sites]: https://github.com/yt-dlp/yt-dlp/blob/master/supportedsites.md

## 👍 Usage

Once your bot is running, simply send it a message containing a video link. The bot will:

- Extract the URL from your message (works with forwarded messages too)
- Show progress through message reactions
- Update its status to show what it's doing (e.g., "recording video")
- Either:
  - Send the video directly in chat (if under 50 MB)
  - Upload to `filebin.net` and provide a download link (if over 50 MB)

No special commands are needed - just send the link!

## 🐋 Docker image

| Registry                          | Image                              |
|-----------------------------------|------------------------------------|
| [GitHub Container Registry][ghcr] | `ghcr.io/tarampampam/video-dl-bot` |

[ghcr]:https://github.com/users/tarampampam/packages/container/package/video-dl-bot

> [!IMPORTANT]
> Using the `latest` tag for the Docker image is highly discouraged due to potential backward-incompatible changes
> during **major** upgrades. Please use tags in the `X.Y.Z` format.

The following platforms for this image are available:

```shell
$ docker run --rm mplatform/mquery ghcr.io/tarampampam/video-dl-bot:latest
Image: ghcr.io/tarampampam/video-dl-bot:latest
 * Manifest List: Yes (Image type: application/vnd.oci.image.index.v1+json)
 * Supported platforms:
   - linux/amd64
   - linux/arm64
```

## 🔌 Quick Start

### Using Docker

```bash
docker run -it --rm \
  -e BOT_TOKEN='your-telegram-bot-token' \
  ghcr.io/tarampampam/video-dl-bot
```

With cookies file for YouTube (and other services) authentication:

```bash
docker run -it --rm \
  -e BOT_TOKEN='your-telegram-bot-token' \
  -v $(pwd)/cookies.txt:/tmp/cookies.txt:ro \
  ghcr.io/tarampampam/video-dl-bot:latest \
  --cookies-file=/tmp/cookies.txt
```

### Using Kubernetes (Helm)

Add the Helm repository:

```bash
helm repo add video-dl-bot https://tarampampam.github.io/video-dl-bot/helm-charts/
helm repo update
```

Install the chart:

```bash
helm install video-dl-bot video-dl-bot/video-dl-bot --set config.botToken.plain='your-telegram-bot-token'
```

With cookies from a Kubernetes secret:

```bash
# Create a secret with your cookies file
kubectl create secret generic bot-cookies --from-file=cookies.txt

# Install with cookies mounted
helm install video-dl-bot video-dl-bot/video-dl-bot \
  --set config.botToken.plain='your-telegram-bot-token' \
  --set config.cookiesFile='/tmp/cookies.txt' \
  --set deployment.volumes[0].name='cookies' \
  --set deployment.volumes[0].secret.secretName='bot-cookies' \
  --set deployment.volumeMounts[0].name='cookies' \
  --set deployment.volumeMounts[0].mountPath='/tmp/cookies.txt' \
  --set deployment.volumeMounts[0].subPath='cookies.txt' \
  --set deployment.volumeMounts[0].readOnly=true
```

For more configuration options, see the [Helm chart documentation][artifacthub].

[artifacthub]: https://artifacthub.io/packages/helm/video-dl-bot/video-dl-bot/

## 🔑 Getting a Telegram Bot Token

1. Open Telegram and search for [@BotFather](https://t.me/BotFather)
2. Send `/newbot` command
3. Follow the prompts to choose a name and username for your bot
4. BotFather will provide you with a token in the format `123456789:ABCdefGHIjklMNOpqrsTUVwxyz`
5. Use this token with the `BOT_TOKEN` environment variable or `--bot-token` flag

## ⚙ Configuration

### Environment Variables

| Variable                   | Description                                     | Default   |
|----------------------------|-------------------------------------------------|-----------|
| `BOT_TOKEN`                | Telegram bot token (required)                   | -         |
| `COOKIES_FILE`             | Path to cookies file in Netscape format         | -         |
| `MAX_CONCURRENT_DOWNLOADS` | Maximum number of parallel downloads            | `5`       |
| `LOG_LEVEL`                | Logging level: `debug`, `info`, `warn`, `error` | `info`    |
| `LOG_FORMAT`               | Logging format: `console`, `json`               | `console` |
| `PID_FILE`                 | Path to PID file for healthchecks               | -         |

<!--GENERATED:APP_README-->

## 💻 Command line interface

```
Description:
   This is a video download bot that allows you to download videos not leaving Telegram.

Usage:
   video-dl-bot

Version:
   0.0.0@undefined

Options:
   --log-level="…"                         Logging level (debug/info/warn/error) (default: info) [$LOG_LEVEL]
   --log-format="…"                        Logging format (console/json) (default: console) [$LOG_FORMAT]
   --bot-token="…", -t="…"                 Telegram bot token [$BOT_TOKEN]
   --cookies-file="…", -c="…"              Path to the file with cookies (netscape-formatted) for the bot (optional) [$COOKIES_FILE]
   --max-concurrent-downloads="…", -m="…"  Maximum number of concurrent downloads (default: 5) [$MAX_CONCURRENT_DOWNLOADS]
   --pid-file="…"                          Path to the file where the process ID will be stored [$PID_FILE]
   --healthcheck                           Check the health of the bot (useful for Docker/K8s healthcheck; pid file must be set) and exit
   --help, -h                              Show help
   --version, -v                           Print the version
```

<!--/GENERATED:APP_README-->

## 🍪 Using Cookies for Authentication

Many platforms, especially YouTube, require authentication to avoid rate limiting and download restrictions. Without
cookies, you may only be able to download a few videos before encountering errors.

For more details on authentication and cookies, see the [yt-dlp FAQ](https://github.com/yt-dlp/yt-dlp/wiki/FAQ).

## 👾 Support

[![Issues][badge-issues]][issues]
[![Issues][badge-prs]][prs]

If you encounter any bugs in the project, please [create an issue][new-issue] in this repository.

[badge-issues]:https://img.shields.io/github/issues/tarampampam/video-dl-bot.svg?maxAge=45
[badge-prs]:https://img.shields.io/github/issues-pr/tarampampam/video-dl-bot.svg?maxAge=45
[issues]:https://github.com/tarampampam/video-dl-bot/issues
[prs]:https://github.com/tarampampam/video-dl-bot/pulls
[new-issue]:https://github.com/tarampampam/video-dl-bot/issues/new/choose

## 📖 License

This is open-sourced software licensed under the [MIT License][license].

[license]:https://github.com/tarampampam/video-dl-bot/blob/master/LICENSE
