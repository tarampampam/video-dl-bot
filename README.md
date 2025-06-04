# Video Downloader Bot

## TODO:

- Add health check for the bot (and add it to the Helm chart)

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
