# Grafana Troubleshooting Lab

A small shop runs on your laptop: an `orders` service, an `inventory` service, and a load generator that places orders all day. Grafana, Loki, Prometheus and Alloy watch it. During the session the worksheet switches the lab into broken modes, and you use Grafana to find out what went wrong.

## Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/), running
- About 2 GB of free memory for the lab, on top of your browser and video call
- 3 GB of free disk space
- Git

## Setup

```sh
git clone https://github.com/jwsmith24/grafana-practice-time.git
cd grafana-practice-time
cp .env.example .env
docker compose up -d
./check.sh
```

The first `docker compose up -d` downloads the images and builds the three lab services, which takes a few minutes. After that, the lab starts in seconds and needs no network.

When `./check.sh` prints `All checks passed`, open http://localhost:3300. No login is needed. The home page is **Lab Overview**, and its request-rate panel should show traffic for `orders` and `inventory`.

## Check your lab

```sh
./check.sh
```

`check.sh` checks that Docker and all seven containers are running, that Grafana, Loki and Prometheus answer, that logs are arriving from every service, and that Prometheus can scrape the services. It prints one `PASS` or `FAIL` line per check, and a fix under each `FAIL`. It exits 0 only when every check passes. Run it again after any fix.

## Worksheet

The session's worksheet is `materials/worksheet.md`, with the LogQL cheat sheet `materials/logql-cheat-sheet.md` next to it. The worksheet is six short scenarios that follow one on-call shift, each practising a different part of Grafana, from Explore to annotations and data links. It starts with scenario 1. Each answer is a password that unlocks the next scenario:

```sh
./unlock.sh 1 <answer>   # a right answer prints scenario 2, and so on
```

Unlocking scenarios 3 to 5 starts the lab's modes for you, so you don't need `./incident.sh` during the worksheet.

Stuck on a scenario for more than 5 minutes? Ask the facilitator.

## Reset

To start over with an empty lab (no logs, no metrics, no running mode):

```sh
docker compose down -v
docker compose up -d
./check.sh
```

## Troubleshooting

Each `FAIL` line from `./check.sh` comes with one of these messages.

**Docker isn't running.**
Start Docker Desktop and wait until it says it is running, then run `./check.sh` again.

**Not running: <services>.**
Some containers stopped or never started. Run `docker compose up -d`. If one keeps stopping, run `docker compose logs <service>` and send the output to the facilitator.

**Another program is using port <port>.**
Something else on your Mac already uses Grafana's port (3300). Pick a free port, set it in `.env` (for example `GRAFANA_PORT=3301`), run `docker compose up -d`, and open http://localhost:3301 instead.

**Grafana isn't answering at <url>.**
Run `docker compose up -d grafana`, wait 30 seconds, and run `./check.sh` again. If it still fails, check `docker compose logs grafana`.

**Loki isn't ready.**
Run `docker compose up -d loki` and wait 30 seconds. Loki takes a little longer to start than the rest.

**Prometheus isn't ready.**
Run `docker compose up -d prometheus`, then check `docker compose logs prometheus`.

**There are no lines from <service> in the last minute.**
Logs from that service aren't reaching Loki. Run `docker compose up -d <service> alloy`, wait a minute, and run `./check.sh` again.

**Prometheus can't scrape <service>.**
Run `docker compose up -d <service>`, then check `docker compose logs <service>`.

Other problems:

- **Low memory.** If containers keep restarting, or Docker Desktop reports low memory, open Docker Desktop's **Settings → Resources** and give it at least 4 GB of memory. Then close other heavy apps and run `docker compose up -d`.
- **Image pulls fail** (`pull access denied`, `toomanyrequests`, or timeouts). Wait a few minutes and run `docker compose up -d` again. If it keeps failing, ask the facilitator for a registry, set it in `.env` as `IMAGE_REGISTRY=<registry>`, and run `docker compose pull` and then `docker compose up -d --build`. Leave the image tags in `.env` as they are.
