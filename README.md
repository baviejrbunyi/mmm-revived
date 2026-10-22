# MMM Revived

Private server for Marvel Mystic Mayhem.

## You need

- [Go 1.27.2 or newer](https://go.dev/dl/)
- [just](https://github.com/casey/just#installation)
- [The Marvel MM CN game client](https://archive.org/details/marvel-mystic-mayhem-v-3-2262-full-build)

## Host a game

1. (Optional DLC) Download `MMM-2.1.20.zip` from https://archive.org/details/mmm-2.1.20
2. Put the zip in the `content` folder of this project (do not unzip it).
3. Close the game, then run:

   ```
   just configure "<game folder>"
   just run
   ```

4. Open the game and accept the download and restart prompts.

To let friends join, set `host` in `build/config.toml` to an address they can reach (a Tailscale IP works well), run `just configure "<game folder>"` again, then `just run`. Without a VPN, open TCP ports 14100 and 18080.

## Join a friend

You do not need the content zip.

1. Clone the Repo
2. run (in the project directory):

   ```
   just join "<game folder>" http://<host address>:18080
   ```

2. Open the game and accept the download and restart prompts.

## Config

Edit `build/config.toml`, then run `just configure "<game folder>"` and `just run` again.

| Option | What it does |
| --- | --- |
| `host` | Your IP that you or your friends connect to |
| `port` | Game port |
| `httpPort` | Download port (DLC) |
| `fresh_start` | `true` = start from zero // `false` = have everything |
| `start` | `"items"` = all items, `"unlocked"` = all items + story done |

## What's added (Custom Content)

- **English translation** of the CN client: text, login screen and UI.
- **Friend duels:** challenge a friend from the friend list with Own Heroes, Random 5 or Current Draft, then pick how abilities are played:
  - **Auto Abilities:** heroes play themselves.
  - **Manual Abilities:** a custom PvP mode where you play every card and ultimate yourself, synced live between both players. No timer and no speed-up; the duel ends when one side's heroes are all dead.
- **Live friend status:** online and offline updates show up instantly.
- **Latest CN content (2.1.20):** all heroes, events and missions.
- **Local login:** username and password, no NetEase account needed.

## Issues

If you run into a problem, feel free to open an issue and I will get to it in my free time.
