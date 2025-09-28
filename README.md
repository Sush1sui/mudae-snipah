# Sniper Bot

Lightweight Discord bot that monitors Mudae messages and notifies configured users when specified characters appear.

## Features

- Detects Mudae character embeds in a configured channel.
- Looks up character metadata (rank, kakera) from a local JSON file.
- Sends a direct message with a jump link and character metadata.
- Notifies members in a configured role after a configurable delay.

## Requirements

- Go 1.18+ (or compatible)
- A Discord bot token with the following intents:
  - Guilds
  - Guild Members
  - Guild Presences (if needed by other parts of the bot)
  - Direct Messages (bot must be able to create DM channels)
- The bot added to the target guild with appropriate permissions.

## Configuration

Environment variables used by the bot:

- DISCORD_TOKEN — Discord bot token.
- MUDAE_CHANNEL_ID — Channel ID where Mudae messages appear.
- SNIPER_ROLE_ID — Role ID whose members will be notified after delay.
- SECRET — Number of seconds to wait before notifying the role (used as a delay).

Characters metadata file:

- Path: `internal/common/characters.json`
- Expected format: JSON array of objects. Each object may include:
  - `rank` (number)
  - `name` (string)
  - `kakera` (number)
    Example entry:

```json
{
  "rank": 1,
  "name": "zero two",
  "kakera": 1065
}
```

## Build & Run

1. Set required environment variables (Windows PowerShell example):
   $env:DISCORD_TOKEN="your_token"
   $env:MUDAE_CHANNEL_ID="channel_id"
   $env:SNIPER_ROLE_ID="role_id"
   $env:SECRET="5"

2. Build:
   go build ./...

3. Run:
   ./sniper_bot.exe

## Troubleshooting

- Check bot permissions and intents in the Discord Developer Portal.
- Ensure `characters.json` is valid JSON and placed at `internal/common/characters.json`.
- Review bot logs printed to stdout for errors when loading configuration or sending messages.

## Contributing

- Open an issue for bugs or feature requests.
- PRs should include tests where appropriate and be formatted with `gofmt`.

## License

MIT
