# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build ./...          # compile
go run main.go          # run without building
go build -o mikrom .    # build binary
go test ./...           # run all tests
go mod tidy             # sync dependencies
```

## Architecture

The CLI is a **Cobra-based Go CLI** that consumes the Mikrom REST API (`../mikrom`).

```
main.go                   # entry point → cmd.Execute()
cmd/
  root.go                 # root command, persistent --api-url / --token flags, initConfig()
  auth.go                 # auth {login, register, profile, logout}
  vm.go                   # vm {list, get, create, deploy, delete, start, stop, restart}
  ippool.go               # ippool {list, get, create, delete, stats}
internal/
  api/client.go           # HTTP client wrapping all API endpoints
  config/config.go        # ~/.mikrom/config.json (api_url + token)
```

**Auth flow**: `auth login` calls `POST /api/v1/auth/login`, saves the JWT token to `~/.mikrom/config.json`. All protected commands call `requireAuth()` which exits early if no token is present.

**API URL**: defaults to `http://localhost:8080`, overridable via `--api-url` flag or the saved config.

## Mikrom API (`../mikrom`)

- REST API on port 8080, JWT-authenticated
- Resources: Users, VMs (Firecracker microVMs), IP Pools
- VM operations (create, start, stop, restart, delete) are async — they queue tasks via Redis/asynq and return immediately
- VM states: `pending`, `provisioning`, `building`, `starting`, `running`, `stopping`, `stopped`, `restarting`, `error`, `deleting`
