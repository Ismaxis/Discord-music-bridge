# Discord music bridge

A personal Go service that accepts a YouTube URL in one Telegram chat, downloads
its audio locally, and streams it into one configured Discord voice channel.
Only one request plays at a time; additional requests are declined rather than
queued.

Use this only for audio you are authorized to download and play. Follow the
applicable YouTube, Telegram, and Discord terms and local laws.

## Requirements

- Go 1.22 or newer
- DAVE-capable DiscordGo fork (pinned in `go.mod`)
- `yt-dlp` available on `PATH`
- `ffmpeg` with `libopus` support available on `PATH`
- A Discord bot invited to your server with permission to connect and speak in
  the target voice channel
- A Telegram bot and the numeric chat ID that is allowed to control it

No privileged Discord intent is needed; the bot uses guild and voice-state
intents. The Discord library handles DAVE voice encryption and the app waits
for encryption setup to finish before accepting playback. Configure the
Discord bot with the `bot` OAuth scope and `Connect`/`Speak` permissions for
the voice channel.

To play an audio attachment, voice message, or audio document, send it in the
authorized Telegram chat as a reply to a message from the bot (for example, its
`/help` instructions). Documents are accepted when their MIME type is `audio/*`
or their filename has a recognized audio extension. Telegram's hosted Bot API
currently limits bot downloads to 20 MiB per file; larger uploads are rejected
with an explanatory message. Audio uploads are kept in a temporary directory
only for the duration of playback and are removed afterward.

## Configure and run

Create a `.env` file in the project directory with these values:

```dotenv
DISCORD_TOKEN=your-discord-bot-token
DISCORD_GUILD_ID=your-discord-server-id
DISCORD_VOICE_CHANNEL_ID=your-voice-channel-id
TELEGRAM_BOT_TOKEN=your-telegram-bot-token
TELEGRAM_CHAT_ID=your-numeric-telegram-chat-id

# Optional: choose one cookie source for videos requiring sign-in.
# YTDLP_COOKIES_FROM_BROWSER=chrome
# YTDLP_COOKIES_FILE=C:\secure\youtube-cookies.txt

# Optional: explicitly select a supported yt-dlp JavaScript runtime.
# YTDLP_JS_RUNTIME=deno
```

Keep `.env` private; do not commit or share it. Run the service with `go run .`
from the project directory.

To use executable paths that are not on `PATH`, set `YTDLP_PATH` and/or
`FFMPEG_PATH`. The service joins the configured voice channel at startup. Send
`/start` or `/help` to the authorized Telegram chat for instructions, then send
a YouTube URL, `/play <URL>`, or an audio file. `Ctrl+C` stops the service.
Send `/stop` in the authorized chat to cancel the current download or stop
playback.

For age-restricted videos, yt-dlp needs cookies from an account that is
authorized to watch them. Set either `YTDLP_COOKIES_FROM_BROWSER` to a browser
yt-dlp supports (for example, `chrome`), or `YTDLP_COOKIES_FILE` to a
Netscape-format cookies file exported for yt-dlp. Do not set both. Browser
cookies are read as the same Windows user running the app. Treat exported
cookies like a password: keep the file outside the project, never share or
commit it, and revoke the session if it is exposed. Cookie support cannot
bypass age verification or access restrictions the signed-in account does not
meet.

The JavaScript-runtime warning is separate from the age restriction. Install a
supported runtime such as Deno and make sure it is on `PATH`; if yt-dlp does
not detect it, set `YTDLP_JS_RUNTIME=deno`. Keep yt-dlp and its EJS support
components up to date.

The bot does not automatically discover the Telegram chat ID. Obtain it from a
trusted Telegram bot/API workflow and set `TELEGRAM_CHAT_ID`; requests from
other chats are ignored.
