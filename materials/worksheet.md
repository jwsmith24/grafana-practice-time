# Troubleshooting worksheet

Today you're on call for the shop. Over one shift, six scenarios follow each other: a ticket from support, a page or two, and a review at the end. Each one builds on what you found in the one before, and each practises a different part of Grafana. They get harder as you go, but every step tells you where to click.

Work in pairs: one person drives Grafana, the other reads the steps, and you swap after each scenario. Use only Grafana (the **Lab Overview** dashboard and **Explore**) and the queries on the LogQL cheat sheet. Don't read the lab's source code.

You get one scenario at a time. Each answer is a password that unlocks the next scenario: solve scenario 1 below, then run

```sh
./unlock.sh 1 <answer>
```

from the lab's folder. A right answer prints scenario 2, and so on; the sixth prints a closing message. Case and extra spaces don't matter, and you don't need quotes. A wrong answer prints "Not quite": look again and retry. If you've been stuck on one scenario for more than 5 minutes, ask the facilitator.

Tips:

- "Query 1" means query 1 on the cheat sheet. In Explore, switch the query editor to **Code** and paste it, then replace each `<placeholder>`. The dropdowns only list a value once a line with it exists, so a value you need may not be there yet.
- Scenarios 3 to 5 each break something in the lab when you unlock them, and the right answer to scenario 5 puts the lab back to normal. A problem ends by itself after 3 minutes; if it ends before you've found the password, run the same `./unlock.sh` command again to restart it.

## 1. A ticket from support

09:05, the start of your shift. Sam from support forwards a ticket: a customer says checkout showed them an error earlier, and they want to know what happened to their order. Sam can trace any single order, but only if you can tell them which field in the logs holds a request's ID.

1. In Grafana, open **Explore** (the compass in the left menu) and check that the data source at the top is **Loki**.
2. Run query 1 with `service` = `orders` and `level` = `INFO`.
3. Click a line to expand it. Grafana lists every field the line carries. One of them holds an ID that is different for every request.

Grafana skill: Explore and log details.

The password is that field's name.
