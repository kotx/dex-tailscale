# dex-tailscale

Simple [Dex](https://dexidp.io) setup to POC using [Cloudflare Access](https://www.cloudflare.com/zero-trust/products/access/) with [Tailscale](https://tailscale.com) as an authentication source.

## Setup

Fill in the values in `dex.env` and `tailscale.env` using `dex.env.example` and `tailscale.env.example` respectively.

The default `dex.yaml` configuration includes Cloudflare Access as a client, but you can update this if you want :)

Then just run `docker compose up` and you're done!
