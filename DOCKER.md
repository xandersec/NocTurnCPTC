# Running PhantomDeck in Docker (tailnet-hosted)

Run the web GUI as an always-on container on a Tailscale-connected box and reach
it from Kali (or anything on your tailnet) at `http://<host-tailnet-ip>:8777`.
Keeps it off Kali's resources and off the public internet.

> **VPN note:** the *Connect VPN* button tunnels the **container**, not Kali.
> Upload/build/map/themes all work remotely — but connect your exam/lab VPN on
> Kali itself. Treat this container as your command-builder + attack-map brain.

## One-time setup

Both `PhantomDeck/` and `NmapResponder/` must sit side by side (the build pulls
NmapResponder in as a named build context). Then:

```bash
cd PhantomDeck
cp .env.example .env
# set PD_BIND to THIS host's tailnet IP:
echo "PD_BIND=$(tailscale ip -4 | head -1)" > .env

docker compose up -d --build
```

Open `http://<PD_BIND>:8777` from any tailnet device.

## What persists

- Projects + the generated Obsidian `.canvas` live in the `phantomdeck-data`
  volume, so they survive restarts and rebuilds.
- To get at the `.canvas` files on the host:
  `docker compose cp phantomdeck:/data ./pd-data` (or inspect the named volume).

## Omarchy themes

The theme picker reads the Docker host's Omarchy files. If this host runs
Omarchy, uncomment the two `omarchy` mounts in `docker-compose.yml`. On a
headless server the theme list just stays empty (everything else works).

## Common commands

```bash
docker compose logs -f          # watch it
docker compose restart          # restart
docker compose down             # stop (keeps the data volume)
docker compose up -d --build    # rebuild after code changes
```

## Exposure

`PD_BIND` publishes the port on a single host IP. Point it at your tailnet IP so
only your tailnet can reach it. Avoid `0.0.0.0` unless you firewall the port —
the GUI is unauthenticated and accepts `.ovpn` uploads.
