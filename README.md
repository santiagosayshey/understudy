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

## Overview

Understudy lets you replace Plex's actor portraits with pictures of your own by pretending to be Plex's metadata CDN. It is one small binary with three jobs:

- **proxy** answers as the CDN beside Plex, serving your portraits and forwarding the rest.
- **sync** asks Plex which URL it currently uses for each person you configured, since Plex names portraits by a hash that changes when its photo does, writes that down for the proxy, and clears Plex's photo cache when something changed.
- **edit** is a web page for choosing portraits: find the person, confirm it is the right one, crop, review, apply.

## Getting started

Plex fetches actor portraits itself, over HTTPS, from one hostname. Understudy answers at that hostname, so most of the setup is convincing Plex: a certificate it will trust, and a hosts entry that sends the hostname to the proxy. Nothing in Plex itself changes. With that in place you choose portraits in the editor, and sync tells the proxy which URLs to answer with them.

### Requirements

- Docker with Compose
- Plex in Docker, on the linuxserver image. Another image needs its own way of trusting a certificate.
- Your Plex [token](https://support.plex.tv/articles/204059436-finding-an-authentication-token-x-plex-token/)

### A folder for everything

Everything Understudy owns lives in one folder: the certificates, the configuration and portraits, and the state the proxy reads. Plex gets nothing from it but one startup script.

```bash
mkdir understudy && cd understudy
mkdir certs config portraits state
echo 'version: 1' > config/configuration.yml
echo 'PLEX_TOKEN=your-token' > .env
```

### Make a certificate Plex will trust

Plex checks the CDN's certificate, so the proxy needs one for the CDN's hostname that Plex accepts. No public authority will sign that, so you make your own: a private authority, and a certificate signed by it.

```bash
docker run --rm --user "$(id -u):$(id -g)" -v "$PWD/certs:/certs" ghcr.io/santiagosayshey/understudy:latest cert
```

The certificate stays with the proxy. The authority is for Plex, and `cert` writes it into a startup script under `certs/plex` for the next step. Keep `ca.key` with your other secrets: Plex will trust anything signed with it.

### Run the proxy

The proxy is what answers as the CDN. It serves your portraits and passes everything else through to the real one, so Plex sees no difference. Plex will find it by IP, so it gets a fixed address on a network of its own. Nothing else talks to it: no ports, no token.

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

### Point Plex at it

Plex needs two things: to resolve the CDN's hostname to the proxy, and to trust the authority that signed the proxy's certificate. The hosts entry does the first. The startup script that `cert` wrote does the second: the linuxserver image runs anything under `/custom-cont-init.d` on every start, so the authority is installed before Plex comes up, upgrades included.

```yaml
    extra_hosts:
      - "metadata-static.plex.tv:172.31.250.10"
    volumes:
      - /path/to/understudy/certs/plex:/custom-cont-init.d:ro
```

If Plex is on a Docker network rather than the host's, attach it to the `understudy` network as well.

### Choose portraits

The editor is where you pick who gets which picture. It talks to Plex to find people, so it needs the token, and it writes only two things: `configuration.yml` and the portraits folder.

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

Open http://localhost:8090. Search a name, open the person, drop in a photo, crop it, and apply from the review drawer.

### Sync

The proxy serves by URL, and only Plex knows which URL each person's portrait has right now. Sync asks Plex, writes the answer down for the proxy, and clears Plex's photo cache so the change shows. It runs once and exits.

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

Hard refresh your Plex client and the portraits are yours. Plex changes those URLs now and then, so run sync on a schedule too. It is what keeps a portrait attached when that happens:

```
0 * * * *  cd /path/to/understudy && docker compose run --rm sync
```

### All in one file

[contrib/compose.yml](contrib/compose.yml) is everything above in one file, Plex included.

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
