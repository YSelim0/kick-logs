# VPS Memory Stability Runbook

This runbook covers running Kick Logs on a small single-node VPS (target: 4 GB RAM).

## Background

ClickHouse otherwise assumes it owns the whole machine (default
`max_server_memory_usage` ~90% of host RAM) and, combined with no container limits, lets one
service's memory growth take down the entire host: RAM fills, the host swaps, every process
(api, web, listener, even SSH) becomes unresponsive, and only a full reboot recovers it. The
fixes below bound each process so a memory spike is contained instead of fatal.

## Memory budget (4 GB host)

Applied in `compose.yaml` and `clickhouse/config.d/memory.xml`:

| Service      | `mem_limit` | Notes                                                          |
| ------------ | ----------- | -------------------------------------------------------------- |
| `clickhouse` | 1536m       | `max_server_memory_usage` ~1.2 GiB (below the container limit) |
| `web`        | 768m        | Next.js production build (`next start`)                        |
| `listener`   | 512m        | `GOMEMLIMIT=410MiB`                                            |
| `api`        | 384m        | `GOMEMLIMIT=307MiB`                                            |

Total ~3.2 GB, leaving headroom for the OS. Every long-running service has
`restart: unless-stopped`, so a service that exceeds its limit is OOM-killed and restarted by
Docker instead of taking down the host.

## Swap safety net (P2 backstop)

Memory limits are the fix; swap is a backstop. A small swap file turns a sudden spike into a
brief slowdown instead of an OOM lockup that needs a reboot. Add 2–4 GB of swap on the VPS host
(not in a container):

```bash
# Create a 2 GB swap file (use 4G for more headroom)
sudo fallocate -l 2G /swapfile        # or: sudo dd if=/dev/zero of=/swapfile bs=1M count=2048
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile

# Persist across reboots
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab

# Optional: reduce swap aggressiveness (prefer RAM, swap only under pressure)
sudo sysctl vm.swappiness=10
echo 'vm.swappiness=10' | sudo tee /etc/sysctl.d/99-swappiness.conf
```

Verify: `swapon --show` and `free -m`.

## Verify after deploy

```bash
docker compose up --build -d
docker stats --no-stream                      # per-service RAM/CPU vs mem_limit
free -m                                        # host memory + swap usage
```

In ClickHouse, watch memory and part/merge pressure over time:

```sql
SELECT value FROM system.metrics WHERE metric = 'MemoryTracking';
SELECT count() FROM system.parts WHERE active;
SELECT count() FROM system.merges;
```

Expectation after the fixes: each service's RES stays under its `mem_limit`; hitting a limit
restarts a single container instead of locking up the host. If a service keeps getting
OOM-killed, raise its `mem_limit` and lower another's so the total still fits with OS headroom.
