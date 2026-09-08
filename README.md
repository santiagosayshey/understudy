<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/screenshots/header-dark.webp">
    <img src="docs/screenshots/header-light.webp" width="560" alt="understudy">
  </picture>
</p>

<p align="center"><b>Custom actor portraits for Plex.</b></p>

<p align="center">
  <a href="https://github.com/santiagosayshey/understudy/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/santiagosayshey/understudy/ci.yml?branch=develop&label=ci&logo=githubactions&logoColor=white" alt="ci"></a>
  <a href="https://github.com/santiagosayshey/understudy/releases"><img src="https://img.shields.io/github/v/release/santiagosayshey/understudy?label=release&logo=github&logoColor=white" alt="release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/santiagosayshey/understudy?logo=opensourceinitiative&logoColor=white" alt="license"></a>
</p>

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/screenshots/hero-dark.webp">
    <img src="docs/screenshots/hero-light.webp" width="800" alt="An actor's page in the editor">
  </picture>
</p>

## How it works

Understudy lets you replace Plex's actor portraits with pictures of your own by pretending to be Plex's metadata CDN. It is one small binary with three jobs:

- **proxy** answers as the CDN beside Plex, serving your portraits and forwarding the rest.
- **sync** asks Plex which URL it currently uses for each person you configured, since Plex names portraits by a hash that changes when its photo does, writes that down for the proxy, and clears Plex's photo cache when something changed.
- **edit** is a web page for choosing portraits: find the person, confirm it is the right one, crop, review, apply.

## Getting started

You need Plex in Docker, its [token](https://support.plex.tv/articles/204059436-finding-an-authentication-token-x-plex-token/), and Docker Compose. The Plex steps are for the linuxserver image; another image needs its own way of trusting a certificate.

### Make a folder

```bash
mkdir understudy && cd understudy
mkdir certs config portraits state plex-init
echo 'version: 1' > config/configuration.yml
echo 'PLEX_TOKEN=your-token' > .env
```

Everything below goes in here. The containers run as you so they can write to these folders: put your `id -u` and `id -g` in the `user:` lines.

### Make the certificates

```bash
docker run --rm --user "$(id -u):$(id -g)" -v "$PWD/certs:/certs" ghcr.io/santiagosayshey/understudy:latest cert
```

That writes a private certificate authority and a certificate for the CDN's hostname into `certs/`. Keep `ca.key` with your other secrets: Plex will trust anything signed with it.

### Run the proxy

`compose.yml`:

```yaml
services:
  proxy:
    image: ghcr.io/santiagosayshey/understudy:latest
    command: proxy
    user: "1000:1000"
    volumes:
      - ./certs:/certs:ro
      - ./portraits:/portraits:ro
      - ./state:/state:ro
    networks:
      understudy:
        ipv4_address: 172.31.250.10
    restart: unless-stopped

networks:
  understudy:
    ipam:
      config:
        - subnet: 172.31.250.0/24
```

```bash
docker compose up -d proxy
```

The proxy has a fixed address because Plex is pointed at it by IP. Nothing else talks to it, so it publishes no ports and needs no token.

### Point Plex at it

Two additions to the Plex container. One resolves the CDN's hostname to the proxy, the other lets it trust the proxy's certificate:

```yaml
    extra_hosts:
      - "metadata-static.plex.tv:172.31.250.10"
    volumes:
      - /path/to/understudy/certs/ca.crt:/understudy/ca.crt:ro
      - /path/to/understudy/plex-init:/custom-cont-init.d:ro
```

And the script that installs the authority, in `plex-init/10-understudy-ca.sh`, made executable:

```bash
#!/bin/bash
cp /understudy/ca.crt /usr/local/share/ca-certificates/understudy.crt
update-ca-certificates
```

The linuxserver image runs it on every start, so an image upgrade stays trusted. Recreate the Plex container. If Plex is on a Docker network rather than the host's, attach it to the `understudy` network as well.

### Choose portraits

Add the editor to `compose.yml`:

```yaml
  edit:
    image: ghcr.io/santiagosayshey/understudy:latest
    command: edit
    user: "1000:1000"
    environment:
      UNDERSTUDY_PLEX_URL: http://192.168.1.10:32400   # Plex, as a container sees it
      UNDERSTUDY_PLEX_TOKEN: ${PLEX_TOKEN}
    volumes:
      - ./config:/config
      - ./portraits:/portraits
      - ./state:/state:ro
    ports:
      - "127.0.0.1:8090:8090"
    restart: unless-stopped
```

```bash
docker compose up -d edit
```

Open http://localhost:8090. Search a name, open the person, drop in a photo, crop it, and apply from the review drawer. That writes `configuration.yml` and the portraits folder, nothing else.

### Sync

Add the one-shot job:

```yaml
  sync:
    image: ghcr.io/santiagosayshey/understudy:latest
    command: sync
    user: "1000:1000"
    profiles: [tools]
    environment:
      UNDERSTUDY_PLEX_URL: http://192.168.1.10:32400
      UNDERSTUDY_PLEX_TOKEN: ${PLEX_TOKEN}
      UNDERSTUDY_PLEX_CACHE: /plex/PhotoTranscoder
    volumes:
      - ./config:/config:ro
      - ./portraits:/portraits:ro
      - ./state:/state
      - "/path/to/plex/config/Library/Application Support/Plex Media Server/Cache/PhotoTranscoder:/plex/PhotoTranscoder"
```

```bash
docker compose run --rm sync
```

It asks Plex which URL each person's portrait has, writes that down for the proxy, and clears Plex's photo cache. Hard refresh your Plex client and the portraits are yours. Plex changes those URLs now and then, so run it on a schedule too:

```
0 * * * *  cd /path/to/understudy && docker compose run --rm sync
```

### All in one file

[contrib/compose.yml](contrib/compose.yml) is everything above in one file, Plex included, and [contrib/plex](contrib/plex) is the hook with a check that the certificate is there.

## Configuration

The editor writes this file. If you would rather write it yourself:

```yaml
version: 1
people:
  - name: Cailee Spaeny                # as Plex spells it
    image: cailee-spaeny.jpg           # in the portraits folder
  - name: Anthony Edwards
    tagKey: 5d776825880197001ec9003b   # only when two people share a name
    image: anthony-edwards.jpg
```

- Names match Plex's actor listing, ignoring case.
- `tagKey` is Plex's id for a person and tells apart two who share a name. The editor's Info dialog shows it.
- Images are JPEG or PNG. Square is what Plex's round avatars expect; anything else is centre-cropped.
- `understudy validate` checks the file against Plex without changing anything.

## Commands

| Command | What it does |
| --- | --- |
| `understudy proxy` | Stand in for the CDN. Long-lived, beside Plex. |
| `understudy edit` | Serve the editor. Long-lived, wherever the browser is. |
| `understudy sync` | Resolve the configuration, write the state, clear Plex's cache. One shot; run it after changes and on a schedule. |
| `understudy validate` | Check the configuration against Plex. One shot. |
| `understudy cert` | Write the certificate authority and the leaf certificate. Once. |

Every flag has an environment variable of the same name under `UNDERSTUDY_`; `understudy <command> -h` lists them. The editor and sync need `UNDERSTUDY_PLEX_URL` and `UNDERSTUDY_PLEX_TOKEN`. The proxy needs neither.

## Development

### Requirements

- Go 1.26
- Node 24 and pnpm 11
- Docker, only to build the image

### Develop

One command runs both halves with hot reload: the binary is rebuilt and restarted on any Go change, and Vite serves the frontend with its own reload, proxying `/api` to the binary. It works on a throwaway configuration under `dev/` and reads Plex through `UNDERSTUDY_PLEX_URL`.

```bash
make dev
```

`make check` runs everything CI runs. `make lint` runs only the project's own rules: `web/eslint/` forbids raw form elements outside the ui library and palette colours anywhere, and `internal/lint/` keeps package imports on the right side of the design's seams.

### Preview

Build the binary with the frontend embedded and serve it the way the image does.

```bash
make build && ./bin/understudy edit --plex-url http://your-plex:32400 --plex-token …
```

`docker build .` builds the image itself.

## Credits

The masks are Google's [Noto Color Emoji](https://github.com/googlefonts/noto-emoji), Apache 2.0. The typeface is Vercel's [Geist](https://vercel.com/font), SIL Open Font License. Icons are [Lucide](https://lucide.dev), ISC. Understudy itself is [MIT](LICENSE).
