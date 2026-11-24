# syntax=docker/dockerfile:1
#
# PhantomDeck web GUI as a self-contained container. Builds the PhantomDeck
# binary and the NmapResponder recommendation engine (pulled in as the named
# build context "nmapresponder" — see docker-compose.yml), then ships them on a
# slim Debian runtime. Projects/canvas persist on the /data volume.

# ---------- build ----------
FROM golang:1.27-bookworm AS build
ENV CGO_ENABLED=0 GOFLAGS=-trimpath

# NmapResponder first (stdlib only, no module download needed).
WORKDIR /src/nmapresponder
COPY --from=nmapresponder . .
RUN go build -o /out/nmapresponder .

# PhantomDeck — download modules before copying the source so rebuilds are fast.
WORKDIR /src/phantomdeck
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/phantomdeck .

# ---------- runtime ----------
FROM debian:bookworm-slim
RUN apt-get update \
 && apt-get install -y --no-install-recommends openvpn ca-certificates \
 && rm -rf /var/lib/apt/lists/*

COPY --from=build /out/phantomdeck   /usr/local/bin/phantomdeck
COPY --from=build /out/nmapresponder /usr/local/bin/nmapresponder

# Store engagement data on a mounted volume, and point the nmap importer at the
# bundled nmapresponder binary.
ENV PHANTOMDECK_DATA=/data \
    PHANTOMDECK_NMAPRESPONDER=/usr/local/bin/nmapresponder
VOLUME /data
EXPOSE 8777

ENTRYPOINT ["phantomdeck"]
CMD ["--web", "--web-addr", "0.0.0.0:8777"]
