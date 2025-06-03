# Video Downloader Bot

Important note: Since the chart is released together with the app under the same version (i.e., the chart version
matches the app version), its versioning is not compatible with semantic versioning (SemVer). I will do my best to
avoid non-backward-compatible changes in the chart, but due to Murphy's Law, I cannot guarantee that they will
never occur.

## Usage

```shell
helm repo add video-dl-bot https://tarampampam.github.io/video-dl-bot/helm-charts
helm repo update

helm install my-video-dl-bot video-dl-bot/video-dl-bot --version <version_here>
```

Alternatively, add the following lines to your `Chart.yaml`:

```yaml
dependencies:
  - name: video-dl-bot
    version: <version_here>
    repository: https://tarampampam.github.io/video-dl-bot/helm-charts
```

And override the default values in your `values.yaml`:

```yaml
video-dl-bot:
  config:
    botToken:
      plain: "<telegram-bot-token>"
```
