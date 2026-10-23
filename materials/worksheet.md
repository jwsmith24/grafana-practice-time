# Troubleshooting worksheet

Work in pairs: one person drives Grafana, the other reads the steps, and you swap after each question. Use only Grafana (the **Lab Overview** dashboard and **Explore**) and the queries on the LogQL cheat sheet. Don't read the lab's source code.

You get one question at a time. Each answer is a password that unlocks the next question: answer question 1 below, then run

```sh
./unlock.sh 1 <answer>
```

from the lab's folder. A right answer prints question 2, and so on; the sixth prints a closing message. Case and extra spaces don't matter, and you don't need quotes. A wrong answer prints "Not quite": look again and retry. If you've been stuck on one question for more than 5 minutes, ask the facilitator.

Tips:

- "Query 1" means query 1 on the cheat sheet. In Explore, switch the query editor to **Code** and paste it, then replace each `<placeholder>`. The dropdowns only list a value once a line with it exists, so a value you need may not be there yet.
- Questions 3 to 5 each start a mode in the lab when you unlock them, and the right answer to question 5 puts the lab back to normal. A mode ends by itself after 3 minutes; if it ends before you've found the password, run the same `./unlock.sh` command again to restart it.

## 1. Warm-up: find a request's ID

In Explore, run query 1 with `service` = `orders` and `level` = `INFO`. Click a line to expand it. One of its fields holds an ID that is different for every request.

The password is that field's name.
