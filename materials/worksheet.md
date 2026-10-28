# Troubleshooting worksheet

Today you're on call for the shop. Over one shift, six scenarios follow each other: support tickets, a couple of alerts, and a review at the end. Each one ends in one question you answer from Grafana, each practises a different part of Grafana, and they get harder as you go.

Work in pairs: one person drives Grafana, the other reads the scenario and the hints, and you swap after each scenario. Use only Grafana (the **Lab Overview** dashboard and **Explore**) and the queries on the LogQL cheat sheet. Don't read the lab's source code.

You get one scenario at a time. Each answer is a password that unlocks the next scenario: solve scenario 1 below, then run

```sh
./unlock.sh 1 <answer>
```

from the lab's folder. A right answer prints scenario 2, and so on; the sixth prints a closing message. Case and extra spaces don't matter, and you don't need quotes. A wrong answer prints "Not quite": look again and retry.

Stuck? `./hint.sh N` prints a hint for scenario N, and `./hint.sh N 2` and `./hint.sh N 3` print more specific ones. If you're still stuck 5 minutes after hint 3, ask the facilitator.

Tips:

- "Query 1" means query 1 on the cheat sheet. In Explore, switch the query editor to **Code** and paste it, then replace each `<placeholder>`. The dropdowns only list a value once a line with it exists, so a value you need may not be there yet.
- Scenarios 3 to 5 each break something in the lab when you unlock them, and the right answer to scenario 5 puts the lab back to normal. A problem ends by itself after 3 minutes; if it ends before you've found the password, run the same `./unlock.sh` command again to restart it.

## 1. A ticket from support

Support forwards a ticket: "A customer placed order ORD-000042 earlier today and never got a confirmation email. They want to know whether it went through before they try again."

What HTTP status did orders return when ORD-000042 was placed?
