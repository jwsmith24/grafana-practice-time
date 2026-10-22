# LogQL cheat sheet

Paste a query into **Explore** (Loki). Replace each `<placeholder>`. The labels are `service` (`orders`, `inventory`, `loadgen`), `level` (`INFO`, `WARN`, `ERROR`) and `env` (`lab`). Every other field lives in the JSON line: use `| json` to reach it. A `<range>` is a window such as `5m`.

**1. All lines from one service at one level (label selector)**
```logql
{service="<service>", level="<level>"}
```
**2. Lines that contain one text and not another (line filters)**
```logql
{service="<service>"} |= "<text>" != "<other_text>"
```
**3. Lines that match a pattern, in any service (regex, `(?i)` ignores case)**
```logql
{env="lab"} |~ "(?i)<pattern>"
```
**4. Lines where a JSON field has a value (`| json` + field filter)**
```logql
{service="<service>"} | json | <field>="<value>"
```
**5. Requests that took longer than some milliseconds (numeric field filter)**
```logql
{service="<service>"} | json | duration_ms > <ms>
```
**6. WARN lines without the noisy one (hide the red herring)**
```logql
{level="WARN"} != "deprecated"
```
**7. One request across every service (follow a `request_id`)**
```logql
{env="lab"} | json | request_id="<request_id>"
```
**8. Lines per second at one level, by service (`rate()`)**
```logql
sum by (service) (rate({level="<level>"}[1m]))
```
**9. How often each value of a field appears (`count_over_time()` + `sum by`)**
```logql
sum by (<field>) (count_over_time({service="<service>", level="<level>"} | json [<range>]))
```
**10. The five most frequent messages at one level**
```logql
topk(5, sum by (msg) (count_over_time({level="<level>"} | json [<range>])))
```

Tips: set the time picker to start just before things went wrong. Click a field in a log row to see its values, or to filter by it. Metric queries (8–10) show as a graph; switch to **Table** to read the numbers.
