# IstkharMusic

Premium Telegram music bot (Go + TDLib + ntgcalls). Updates: https://t.me/betabot_hub · Support: https://t.me/betabot_support

## Features
Premium small-caps UI with banner, random coloured player buttons (progress bar, pause/skip/end, add to playlist, autoplay toggle),
autoplay, queue, playlists, VC logger (`/vclogger on`), link remover (`/linkremove on`), bio remover (`/bioremove on`),
`/approve` exemptions, `/bdata` daily growth report, `/vc` status, daily 7 AM IST auto promo + `/selfpromo run`,
play logger to `LOGGER_ID`, blacklist, welcome panel when added to a group.

## Deploy

[![Deploy to Heroku](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy?template=https://github.com/TEAM-ISTKHAR/IstkharMusic)
[![Deploy on Railway](https://railway.com/button.svg)](https://railway.app/new/template?template=https://github.com/TEAM-ISTKHAR/IstkharMusic)

1. Fill `.env` (API_ID, API_HASH, TOKEN, STRING1, MONGO_URI, OWNER_ID, LOGGER_ID). YouTube API is already set.
2. **VPS:** `bash deploy/vps.sh`  (or `docker compose up -d --build`)
3. **Heroku:** use the Deploy button / `heroku stack:set container` + `git push heroku`; set `HEROKU_APP_NAME`.
4. **Railway:** New Project → Deploy from repo (uses `railway.json`/Dockerfile); add the same variables.

The banner is served by the bot itself at `/banner.jpg` (needs `BASE_URL` on VPS; auto on Heroku/Railway).
Or set `START_IMG`, `PLAY_IMG`, etc. to any public image URL.

## License
Maintained by [TEAM-ISTKHAR](https://github.com/TEAM-ISTKHAR). GPL-3.0; original copyright notices are kept as the license requires (see LICENSE).
