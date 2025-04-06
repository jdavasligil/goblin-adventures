# Goblin Adventures
A bot that runs the game 'Goblin Adventures', a text based coop RPG on Twitch.


## Environment Variables
Create a file named `.env` and supply the environment variables:
```
CLIENT_ID=<Copy ID Here>
CLIENT_SECRET=<Copy Secret Here>
BOT_USER_ID=<Copy Bot ID Here>
BROADCASTER_ID=<Copy Broadcaster ID here>
```

## Build
Generate TLS certs by running `./scripts/make_ssl_keys.sh` in the terminal from
the project root directory.

Run the bot with `go run .`
