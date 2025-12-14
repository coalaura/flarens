# flarens

Small Go tool that keeps a single Cloudflare DNS A record updated with your current public IP (simple dynamic DNS).

## How it works

1. Reads configuration from `config.yml`
2. Fetches your current public IP from `https://ip.shrt.day`
3. Looks up the configured DNS record in Cloudflare
4. Creates or updates the A record if needed
5. Re-checks every minute and updates on IP change

## Config

Create `config.yml` in the working directory:

```yaml
key: "CLOUDFLARE_API_TOKEN"
zone: "YOUR_ZONE_ID"
record: "home.example.com"
```

The API token must have DNS read/write access for the given zone.

## Run

```bash
go build -o flarens
./flarens
```

Leave it running (e.g. as a systemd service) to keep the record in sync.
