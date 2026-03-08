# Srikar Book Deals (SBD)

## What this repo is
SBD is a fully responsive website for selling Srikar’s used and gently-used books online.

- The application will be built and maintained entirely using **Codex**.
- Spec-driven development: Nicole helps produce business-level specs; Codex implements **one feature at a time**.

## Canonical MVP spec (single source of truth)
To prevent drift, the MVP requirements live in exactly one place:

- `spec/MVP-SPEC.md`

If something changes, update **only** `spec/MVP-SPEC.md`.

## Codex operating instructions
- `CODEX-INSTRUCTIONS.md` — always-on Codex task discipline for this repo
- `MVP-RULES-REFERENCE.md` — optional reference; do not paste wholesale into every Codex task

## Backlog
- `BACKLOG.md` — deferred product items / future features (single source of truth for backlog)

## Project root
This project lives at `~/.openclaw/projects/sbd/`.

## Run with Docker

Start app + Postgres:

```bash
docker compose up -d --build
```

The compose stack now runs a one-shot `migrate` service first; `app` starts only after migrations complete successfully.

If port `8080` is already in use locally:

```bash
APP_PORT=8081 docker compose up -d --build
```
