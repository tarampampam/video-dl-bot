package bot

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	tele "gopkg.in/telebot.v4"

	"gh.tarampamp.am/video-dl-bot/internal/filestorage"
	ytdlp "gh.tarampamp.am/video-dl-bot/internal/yt-dlp"
)

// Emojis used for user interaction feedback.
const (
	emojiBadRequest  = "💩" // emoji to react with when the user provided a bad request
	emojiDownloading = "🫡" // emoji to react with while downloading
	emojiUploading   = "🚀" // emoji to react with while uploading
)

// Chat actions to simulate activity status.
const (
	actDownloading = tele.RecordingVideo
	actUploading   = tele.UploadingVideo
)

type (
	// Bot wraps the Telegram bot client.
	Bot struct {
		cookiesFile            string // path to the cookies file (if any)
		jsRuntimes             string // JavaScript runtimes for yt-dlp (e.g., "node", "bun", "deno", "quickjs")
		maxConcurrentDownloads uint   // maximum number of concurrent downloads allowed

		log    *slog.Logger
		client *tele.Bot
	}

	// Option defines a functional option type for customizing the Bot.
	Option func(*Bot)
)

// WithLogger sets a custom logger for the Bot instance.
func WithLogger(log *slog.Logger) Option { return func(b *Bot) { b.log = log } }

// WithCookiesFile sets the path to a cookies file, used by yt-dlp for authenticated downloads.
func WithCookiesFile(path string) Option { return func(b *Bot) { b.cookiesFile = path } }

// WithJSRuntimes configures the JavaScript runtimes for yt-dlp, allowing support for sites that require JS execution.
func WithJSRuntimes(runtimes string) Option { return func(b *Bot) { b.jsRuntimes = runtimes } }

// WithMaxConcurrentDownloads limits the number of concurrent downloads the bot can handle.
func WithMaxConcurrentDownloads(n uint) Option {
	return func(b *Bot) { b.maxConcurrentDownloads = max(1, min(100, n)) } //nolint:mnd
}

// NewBot creates and returns a new instance of Bot.
func NewBot(ctx context.Context, token string, opts ...Option) (*Bot, error) {
	const pollerTimeout = 10 * time.Second // default timeout for the long poller

	var bot = Bot{ // set default values
		log: slog.Default(),
	}

	for _, opt := range opts {
		opt(&bot)
	}

	client, err := tele.NewBot(tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: pollerTimeout},
		OnError: func(err error, c tele.Context) {
			bot.log.Error(
				"telegram client error",
				slog.String("error", err.Error()),
				slog.String("sender_name", c.Sender().FirstName),
				slog.String("sender_id", fmt.Sprintf("%d", c.Sender().ID)),
			)
		},
	})
	if err != nil {
		return nil, err
	}

	bot.client = client

	var lim = make(Limiter, bot.maxConcurrentDownloads)

	// register command and message handlers
	client.Handle("/start", bot.handleStartCommand())
	client.Handle("test", bot.handleTestCommand())

	var msgHandler = bot.handleMessages(ctx, lim)

	// handle multiple event types with the same message handler
	for _, event := range [...]string{tele.OnText, tele.OnForward, tele.OnReply} {
		client.Handle(event, msgHandler)
	}

	return &bot, nil
}

// Start begins polling updates from Telegram. Blocks until context is canceled.
func (b *Bot) Start(ctx context.Context) {
	var stopped = make(chan struct{})

	// stop bot when context is canceled
	go func() {
		defer close(stopped)

		<-ctx.Done()
		b.client.Stop()
	}()

	// blocking call that listens to updates
	b.client.Start()

	<-stopped
}

// handleStartCommand returns a handler for the "/start" command.
func (b *Bot) handleStartCommand() tele.HandlerFunc {
	return func(c tele.Context) (err error) {
		return b.reply(c.Message(), fmt.Sprintf(`Hello %s! I can help you download videos from hundreds of websites.

Please send or forward me a video URL, and I'll do my best to download it for you!`,
			c.Sender().FirstName,
		))
	}
}

// handleTestCommand returns a handler for a simple "test" command.
func (b *Bot) handleTestCommand() tele.HandlerFunc {
	return func(c tele.Context) (err error) {
		return b.reply(
			c.Message(),
			"Just send me a video URL or forward a message containing a link, "+
				"and I'll download it - that would be the perfect test!",
		)
	}
}

// handleMessages processes incoming user messages and attempts to download video content.
func (b *Bot) handleMessages(pCtx context.Context, lim Limiter) tele.HandlerFunc { //nolint:funlen
	const errWrongMessageReplyMd2 = "Please provide a valid video link\\." +
		"\n" +
		"\n" +
		"Examples:\n" +
		"\\- `https://www\\.youtube\\.com/watch?v=dQw4w9WgXcQ`\n" +
		"\\- `youtu\\.be/dQw4w9WgXcQ`\n" +
		"\n" +
		"You can also share a link to an Instagram reel, TikTok video, or any other video you'd like to download\\. " +
		"Hundreds of sites are supported, so feel free to give it a try\\!"

	return func(c tele.Context) error {
		ctx, cancel := context.WithCancel(pCtx)
		defer cancel()

		var (
			user, userMsg       = c.Sender(), c.Message()
			userUrl, userUrlErr = ExtractLink(c.Text())
		)

		// invalid link - inform user and react
		if userUrlErr != nil {
			_ = b.react(user, userMsg, emojiBadRequest)

			b.log.Info("received invalid link from user",
				slog.String("sender_name", user.FirstName),
				slog.Int64("sender_id", user.ID),
				slog.String("message_text", c.Text()),
			)

			return b.reply(userMsg, errWrongMessageReplyMd2, &tele.SendOptions{
				ParseMode:             tele.ModeMarkdownV2,
				DisableWebPagePreview: true,
				DisableNotification:   true,
			})
		}

		b.log.Info("received video download request",
			slog.String("sender_name", user.FirstName),
			slog.Int64("sender_id", user.ID),
			slog.String("video_url", userUrl.String()),
		)

		// limit concurrent downloads via semaphore
		if err := lim.Acquire(ctx); err != nil {
			return err
		}
		defer lim.Release()

		// clear any previous reactions once we're done
		defer func() { _ = b.clearReactions(user, userMsg) }()

		// indicate download in progress
		_ = b.react(user, userMsg, emojiDownloading)
		stopDownloadingAction := b.setChatAction(ctx, user, actDownloading)

		defer stopDownloadingAction()

		var ytDlpOpts []ytdlp.Option

		if b.cookiesFile != "" {
			ytDlpOpts = append(ytDlpOpts, ytdlp.WithCookiesFile(b.cookiesFile))
		}

		if b.jsRuntimes != "" {
			ytDlpOpts = append(ytDlpOpts, ytdlp.WithJSRuntimes(b.jsRuntimes))
		}

		// download the video
		dl, dlErr := ytdlp.Download(ctx, userUrl.String(), ytDlpOpts...)
		if dlErr != nil {
			b.log.Error("failed to download video",
				slog.String("error", dlErr.Error()),
				slog.String("sender_name", user.FirstName),
				slog.Int64("sender_id", user.ID),
				slog.String("video_url", userUrl.String()),
			)

			return b.reply(userMsg, "❌ Failed to download video")
		}

		stopDownloadingAction()

		// stat the file to get size info
		stat, statErr := os.Stat(dl.Filepath)
		if statErr != nil {
			b.log.Error("failed to stat downloaded video file",
				slog.String("error", statErr.Error()),
				slog.String("file_path", dl.Filepath),
				slog.String("sender_name", user.FirstName),
				slog.Int64("sender_id", user.ID),
				slog.String("video_url", userUrl.String()),
			)

			return b.reply(userMsg, "❌ Downloaded video file not available")
		}

		b.log.Debug("successfully downloaded video",
			slog.String("file_path", dl.Filepath),
			slog.String("sender_name", user.FirstName),
			slog.Int64("sender_id", user.ID),
			slog.String("video_url", userUrl.String()),
			slog.Int64("file_size", stat.Size()),
		)

		defer func() { _ = os.Remove(dl.Filepath) }() // clean up the downloaded file after sending

		// open the downloaded file
		fp, fpErr := os.Open(dl.Filepath)
		if fpErr != nil {
			return fpErr
		}

		defer func() { _ = fp.Close() }()

		// indicate upload in progress
		_ = b.react(user, userMsg, emojiUploading)
		stopUploadingAction := b.setChatAction(ctx, user, actUploading)

		defer stopUploadingAction()

		var fileSizeMb = float64(stat.Size()) / 1024 / 1024 // file size in MB

		// telegram upload limit is 50MB
		if fileSizeMb <= 50 { //nolint:mnd
			if err := b.replyWithVideo(userMsg, tele.Video{File: tele.FromReader(fp)}); err != nil {
				b.log.Error("failed to upload video to Telegram",
					slog.String("error", err.Error()),
					slog.Int64("file_size", stat.Size()),
					slog.String("sender_name", user.FirstName),
					slog.Int64("sender_id", user.ID),
					slog.String("video_url", userUrl.String()),
				)

				return b.reply(userMsg, fmt.Sprintf(
					"❌ Failed to send video (%.2f MB): %s",
					fileSizeMb,
					err.Error(),
				))
			}
		} else {
			// upload to file hosting if file is too large
			fileUrl, urlErr := filestorage.UploadToFileBin(ctx, fp, fmt.Sprintf("video%s", filepath.Ext(dl.Filepath)))
			if urlErr != nil {
				b.log.Error("failed to upload video file to file hosting",
					slog.String("error", urlErr.Error()),
					slog.Int64("file_size", stat.Size()),
					slog.String("sender_name", user.FirstName),
					slog.Int64("sender_id", user.ID),
					slog.String("video_url", userUrl.String()),
				)

				return b.reply(userMsg, "❌ Failed to upload video to file hosting")
			}

			return b.replyWithLink(
				userMsg,
				fmt.Sprintf(
					"[Your video](%s) is ready for download _\\(the link will expire in a couple of days\\)_:",
					userUrl.String(),
				),
				fmt.Sprintf("🚀 Download video (%.2f MB)", fileSizeMb),
				fileUrl,
				&tele.SendOptions{
					ParseMode:             tele.ModeMarkdownV2,
					DisableWebPagePreview: true,
				},
			)
		}

		stopUploadingAction()

		return nil
	}
}

// reply attempts to reply to a message; if the message is not found (e.g. deleted), sends a new message.
func (b *Bot) reply(to *tele.Message, msg string, opts ...any) (err error) {
	_, err = b.client.Reply(to, msg, opts...)
	if err != nil {
		_, err = b.client.Send(to.Sender, msg, opts...)
	}

	return
}

// replyWithVideo sends a video file either as a reply or a fresh message.
func (b *Bot) replyWithVideo(to *tele.Message, v tele.Video) (err error) {
	_, err = b.client.Reply(to, &v)
	if err != nil {
		_, err = b.client.Send(to.Sender, &v)
	}

	return
}

// replyWithLink sends a message with an inline download button.
func (b *Bot) replyWithLink(to *tele.Message, msgText, linkText, linkUrl string, opts ...any) (err error) {
	var markup = tele.ReplyMarkup{
		ResizeKeyboard: true,
		InlineKeyboard: [][]tele.InlineButton{
			{{
				Text: linkText,
				URL:  linkUrl,
			}},
		},
	}

	_, err = b.client.Reply(to, msgText, append(opts, &markup)...)
	if err != nil {
		_, err = b.client.Send(to.Sender, msgText, append(opts, &markup)...)
	}

	return
}

// react adds emoji reactions to a message.
func (b *Bot) react(to tele.Recipient, msg tele.Editable, emoji ...string) error {
	var reactions = make([]tele.Reaction, len(emoji))

	for i, e := range emoji {
		reactions[i] = tele.Reaction{
			Type:  tele.ReactionTypeEmoji,
			Emoji: e,
		}
	}

	return b.client.React(to, msg, tele.Reactions{Reactions: reactions})
}

// clearReactions removes all reactions from a message.
func (b *Bot) clearReactions(to tele.Recipient, msg tele.Editable) error {
	return b.client.React(to, msg, tele.Reactions{Reactions: []tele.Reaction{}})
}

// setChatAction periodically sends a chat action (e.g., typing). Returns a function to stop the action.
func (b *Bot) setChatAction(ctx context.Context, user *tele.User, action tele.ChatAction) (stop func()) {
	ctx, stop = context.WithCancel(ctx) // override the parent context to allow cancellation

	const interval = 4*time.Second + 500*time.Millisecond // 5 seconds is Telegram's recommended interval

	go func() {
		defer stop()

		if ctx.Err() != nil {
			return
		}

		if err := b.client.Notify(user, action); err != nil {
			return
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				if err := ctx.Err(); err != nil {
					return
				}

				if err := b.client.Notify(user, action); err != nil {
					return
				}
			}
		}
	}()

	return
}
