# djin web UI

Vite + React source for the djin UI. Built with `npm run build` into
`../internal/webui/dist/`, where the Go binary embeds it (see
`internal/webui/embed.go`).

Until the real UI lands (a later MVP task), `internal/webui/dist/index.html`
holds a committed placeholder page so the binary builds and serves standalone.
The `djin ui` command already implements the hey app contract v0 in full
(`--port 0 --json` handshake, `/healthz`, `POST /hey/shutdown`, origin guard).
