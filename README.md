# UniFeed Telegram Bot
Bot permits you to gather your channel subscriptions into one feed. This bot simply forwards you posts from your subscribed channels

## How it works

### Subscription system
Telegram bots cannot subsribe themselves to public channels (telegram supergroups) unless their admin won't invite it. Hence, this bot works in pair with some dummy user which subscribes to channels at bot's request. Dummy user, once every new post appears in one of subscibed channels, forwards it to bot which then tranfers it to all the users which are subscribed to this channel.

>Yes, you understood it right: you will need a dummy telegram account as well as dummy tdlib app in order to this bot to work.

### Prerequisites
1. Create a dummy telegram user (you can still use your personal account but all subscribed channels will appear in your own feed)

2. Create a telegram app [here](https://my.telegram.org/apps). Get your app ID and HASH and put them in the appropriate environmental variables (see below)

3. Create a bot via [BotFather](https://t.me/botfather) and put it's token into appropriate environmental variable (see below)

4. Get to know chat_id of the user as seen by the bot and vice versa. Fill in the corresponding environmental variables

5. Set up a MySQL (or MariaDB) server with the schema described in `unifeed-db.sql` file. This simple table is needed to store one-to-many subscriptions. Compose its DSN string as in the example below (you will probably only need to change IP address and credentials).

## Quick Start
The easiest way to quickly get started is with Docker:

First, you will need to add mandatory environmental variables:
```bash
export UNIFEED_TELEGRAM_BOT_API_TOKEN="<tg bot api token>"
export UNIFEED_TELEGRAM_API_ID="<tdlib app id>"
export UNIFEED_TELEGRAM_API_HASH="<tdlib app hash>"
export UNIFEED_USER_TO_BOT_CHAT_ID="<chat_id of the bot as seen by user>"
export UNIFEED_BOT_TO_USER_CHAT_ID="chat_id of the user as seen by bot"
export UNIFEED_SQL_DSN="<user>:<password>@tcp(<IP-address>:<port>)/<schema name>" # e.g. "unifeed-user:unifeed-password@tcp(127.0.0.1:3306)/unifeed-db"
```

```bash
docker run -it -d --name unifeed-bot -v ~/.tdlib:/go/src/github.com/arkhipovkm/unifeed-go/.tdlib -e UNIFEED_TELEGRAM_BOT_API_TOKEN="$UNIFEED_TELEGRAM_BOT_API_TOKEN" -e UNIFEED_TELEGRAM_API_ID="$UNIFEED_TELEGRAM_API_ID" -e UNIFEED_TELEGRAM_API_HASH="$UNIFEED_TELEGRAM_API_HASH" -e UNIFEED_USER_TO_BOT_CHAT_ID="$UNIFEED_USER_TO_BOT_CHAT_ID" -e UNIFEED_BOT_TO_USER_CHAT_ID="$UNIFEED_BOT_TO_USER_CHAT_ID" -e UNIFEED_SQL_DSN="$UNIFEED_SQL_DSN" arkhipovkm/unifeed-bot:latest
```

On the first launch there is no ./tdlib database yet so you'll need to login into Telegram through your dummy app. That is why the above `docker run` command is with the -it (interactive tty) flags. Attach to container to enter Phone number of the dummy account and the verification code:

```bash
docker attach unifeed-bot
```

Nothing will show up, just type in your dummy account's phone number and hit enter. The app will ask for verification code. Type it in and hit Enter.

Once you're logged in, DON'T PRESS `Ctrl+C` or `exit`. This will exit the container. Press `Ctrl+P` then `Ctrl+Q` to **detach** from container and not **kill** it.

Congratulations! You're all set up!

This manipulation is only needed on the first setup.

There is a known [issue](https://github.com/tdlib/td/issues/791) in tdlib which results in error when the dummy user tries to forward posts to the bot. This is because bot's chatID is still not in the tdlib database. To resolve the bug, send any message to your bot using a dummy account.
