# mikrom-cli

Command-line interface for the [Mikrom](https://github.com/spluca/mikrom) API.

## Installation

```bash
git clone https://github.com/spluca/mikrom-cli.git
cd mikrom-cli
go build -o mikrom .
```

## Usage

### Authentication

```bash
mikrom auth register --email user@example.com --password secret --name "John Doe"
mikrom auth login --email user@example.com --password secret
mikrom auth profile
mikrom auth logout
```

### Virtual Machines

```bash
mikrom vm list
mikrom vm get <vm-id>
mikrom vm create --name my-vm --vcpus 2 --memory 1024
mikrom vm start <vm-id>
mikrom vm stop <vm-id>
mikrom vm restart <vm-id>
mikrom vm delete <vm-id>
```

### IP Pools

```bash
mikrom ippool list
mikrom ippool get <pool-id>
mikrom ippool create --name my-pool --cidr 10.100.0.0/24 --gateway 10.100.0.1 --start-ip 10.100.0.10 --end-ip 10.100.0.254
mikrom ippool stats <pool-id>
mikrom ippool delete <pool-id>
```

## Configuration

After login, credentials are saved to `~/.mikrom/config.json`. The API URL defaults to `http://localhost:8080` and can be overridden with the `--api-url` flag.

```bash
mikrom --api-url https://api.example.com vm list
```
