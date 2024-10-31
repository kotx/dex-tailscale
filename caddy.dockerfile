FROM caddy:builder AS builder

RUN xcaddy build v2.8.4 \
    --with github.com/tailscale/caddy-tailscale

FROM caddy:2.8.4

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
