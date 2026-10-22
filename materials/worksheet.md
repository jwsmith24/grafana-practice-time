# Troubleshooting worksheet

Work in pairs: one person drives Grafana, the other keeps notes, and you swap after each question. Use only Grafana (the **Lab Overview** dashboard and **Explore**) and the queries on the LogQL cheat sheet. Don't read the lab's source code.

You get one question at a time. Each answer is a password that unlocks the next question: answer question 1 below, then run

```sh
./unlock.sh 1 <answer>
```

from the lab's folder. A right answer prints question 2, and so on; the sixth prints a closing message. Case and extra spaces don't matter, and you don't need quotes. A wrong answer prints "Not quite": look again and retry. If you've been stuck on one question for more than 5 minutes, ask the facilitator.

Each mode ends by itself after 3 minutes, and its logs stay in Loki. If you need it running again, start it again. Before you start a mode, check that `./incident.sh status` prints `none`.

For questions 3 to 5, write down:

- **When:** the time it started (and ended, if it has).
- **Where:** the service whose logs show the problem first.
- **First error:** the first error or warning line, copied as it appears.
- **One request:** a `request_id` that shows the problem, and every line it produced.
- **Root cause:** what is wrong, in one sentence, and the query that proves it.

## 1. Warm-up: what is in a log line?

Open **Explore** and show the logs of the `orders` service. Which labels does every stream carry? Expand one line: which fields does it have, and which of them would you use to find a single request?

The password is the name of that field.
