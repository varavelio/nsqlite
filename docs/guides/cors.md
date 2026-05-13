# CORS & Browser Access

NSQLite is built for server-to-server communication. The HTTP API expects other servers as clients, not browsers.

If you ever need to hit it from a browser (e.g. from a web-based tool) you'll run into CORS. That's normal, and there's a simple fix.

Only do this if you have a real need and a good reason. Ideally, NSQLite shouldn't be accessible from the outside at all, keep it behind a VPN or restricted to your internal network. If you're exposing it, take security seriously.

## Why a Reverse Proxy?

Browsers block cross-origin requests unless the server explicitly allows them via CORS headers. NSQLite doesn't ship with CORS support:

- **It's out of scope.** NSQLite's job is to give you access to your database. CORS is a concern for the layer that exposes services to the web, and specialized tools handle it better.
- **Server clients don't need it.** Backend services calling NSQLite don't enforce CORS, so shipping those headers would be dead weight.
- **You should opt in deliberately.** Browser access to your database shouldn't happen by accident.

The idiomatic way to handle this is a reverse proxy. It's the right layer for CORS, and since NSQLite is Docker-first, adding one takes two minutes.

Below is an example with Caddy because it's the simplest option with a simple config and automatic TLS (very important). You can achieve the same with Nginx or any other reverse proxy.

Create a `Caddyfile`:

```
nsqlite.example.com {
	@preflight method OPTIONS

	header {
		Access-Control-Allow-Origin "*"
		Access-Control-Allow-Methods "GET, POST, PUT, PATCH, DELETE, OPTIONS"
		Access-Control-Allow-Headers "Authorization, Content-Type, Accept, Origin, X-Requested-With"
		Access-Control-Allow-Credentials "true"
		Vary "Origin"
	}

	respond @preflight 204

	reverse_proxy nsqlite:9876
}
```

Tie it together in `docker-compose.yml`:

```yaml
services:
  nsqlite:
    image: varavel/nsqlite:latest
    environment:
      NSQLITE_AUTH_TOKEN: ${NSQLITE_AUTH_TOKEN}
    volumes:
      - nsqlite_data:/data
    networks:
      - internal
    restart: unless-stopped

  caddy:
    image: caddy:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - caddy_data:/data
    networks:
      - internal
    restart: unless-stopped

volumes:
  nsqlite_data:
  caddy_data:

networks:
  internal:
```

## Security Checklist

| Thing                | Why                                                                   |
| -------------------- | --------------------------------------------------------------------- |
| Pin your origin      | Replace `*` with your actual domain. Don't skip this.                 |
| Use auth tokens      | Set `NSQLITE_AUTH_TOKEN` (or `_RW` / `_RO`) on the NSQLite container. |
| Terminate TLS        | Let the reverse proxy handle HTTPS. Caddy does this out of the box.   |
| Keep NSQLite private | Only the proxy should be publicly reachable, not NSQLite itself.      |

That's it. A reverse proxy + CORS headers + auth tokens and you're good to go.
