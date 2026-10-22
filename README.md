# Grafana Troubleshooting Lab

A small shop runs on your laptop: an `orders` service, an `inventory` service, and a load generator that places orders all day. Grafana, Loki, Prometheus and Alloy watch it. During the session you switch the lab into a broken mode and use Grafana to find out what went wrong.

## Prerequisites

- A Mac with Apple Silicon or Intel
- [Docker Desktop](https://www.docker.com/products/docker-desktop/), running
- About 2 GB of free memory for the lab, on top of your browser and video call
- 3 GB of free disk space
- Git

## When to set up

- **Finish setup at least one day before the session.** The first start downloads about 1 GB of images, which is slow on conference Wi-Fi.
- If `./check.sh` fails and the hints don't fix it, **message the facilitator** with its output.
- You'll work in pairs, and a pair needs only one working laptop, but everyone sets up, so each pair has a spare.

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

## Incidents

The facilitator will tell you which mode to start. Each mode ends by itself after 3 minutes.

```sh
./incident.sh A        # start mode A (or B, or C)
./incident.sh status   # show which mode is running and how long is left
./incident.sh clear    # end it now
```

Your job is to work out what is wrong from Grafana alone, without reading the lab's source code.

## Worksheet

The session's worksheet is `materials/worksheet.md`, with the LogQL cheat sheet `materials/logql-cheat-sheet.md` next to it. The worksheet starts with question 1. Each answer is a password that unlocks the next question:

```sh
./unlock.sh 1 <answer>   # a right answer prints question 2, and so on
```

Stuck on a question for more than 5 minutes? Ask the facilitator.

## Reset

To start over with an empty lab (no logs, no metrics, no running mode):

```sh
docker compose down -v
docker compose up -d
./check.sh
```

## Registry fallback

If your network blocks Docker Hub, the facilitator will give you another registry. Set it in `.env`:

```sh
IMAGE_REGISTRY=<the registry the facilitator gives you>
```

Then pull and start again:

```sh
docker compose pull
docker compose up -d --build
./check.sh
```

Change `IMAGE_REGISTRY` only if the facilitator tells you to. Leave the image tags in `.env` as they are.

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
- **Image pulls fail** (`pull access denied`, `toomanyrequests`, or timeouts). Wait a few minutes and run `docker compose up -d` again. If it keeps failing, ask the facilitator for a registry and follow [Registry fallback](#registry-fallback).
