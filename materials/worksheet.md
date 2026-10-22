# Troubleshooting worksheet

Work in pairs: one person drives Grafana, the other keeps notes, and you swap after each question. Use only Grafana (the **Lab Overview** dashboard and **Explore**) and the queries on the LogQL cheat sheet. Don't read the lab's source code.

Each mode ends by itself after 3 minutes, and its logs stay in Loki. If you need it running again, start it again. Before you start a mode, check that `./incident.sh status` prints `none`.

For questions 3 to 5, write down:

- **When:** the time it started (and ended, if it has).
- **Where:** the service whose logs show the problem first.
- **First error:** the first error or warning line, copied as it appears.
- **One request:** a `request_id` that shows the problem, and every line it produced.
- **Root cause:** what is wrong, in one sentence, and the query that proves it.

## 1. Warm-up: what is in a log line?

Open **Explore** and show the logs of the `orders` service. Which labels does every stream carry? Expand one line: which fields does it have, and which of them would you use to find a single request?

## 2. Warm-up: follow one request

Copy the `request_id` of one `POST /orders` line from `orders`. Find every line in the lab with that `request_id`. Which services handled the request, in what order, and how long did each one take?

## 3. Mode A

Run `./incident.sh A`. Orders start failing. Find when, where, the first error, one request, and the root cause.

## 4. Mode B

Run `./incident.sh B`. Orders start failing again. Find when, where, the first error, one request, and the root cause.

## 5. Mode C

Run `./incident.sh C`. Orders start failing again. Find when, where, the first error, one request, and the root cause.

## 6. Stretch: the noisy warning

One WARN line shows up all the time, whether or not anything is broken. Which line is it, which service writes it, and how many times a minute? Write a query that shows every WARN line in the lab except that one.
