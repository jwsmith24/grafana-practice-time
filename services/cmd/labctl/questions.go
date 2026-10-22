package main

// The worksheet's questions. Question 1 is in materials/worksheet.md; each
// answer unlocks the question after it, and the last one unlocks the closing
// message. Only the hashes of the answers are kept (see answerHash).
var labWorksheet = worksheet{
	questions: []question{
		{
			title: "1. Warm-up: find a request's ID",
			hash:  "83225a4843d7c44ae7803ea4a14688a795ccbe9b5dfc43e6da696ea7bcb394ba",
		},
		{
			title: "2. Warm-up: follow one request",
			body: `Copy the ID from question 1 out of a line whose endpoint is POST /orders.
Run query 7 with that ID. You get two lines: one from orders, and one from
the service that orders called.

The password is the name of that other service.`,
			hash: "591926df98b0bd032a92ea9800c354fdb6a11d31ad3bcab7bdcecff3ff9969dc",
		},
		{
			title: "3. Mode A",
			mode:  "A",
			body: `Orders start failing, and the Error rate panel goes up for orders.

1. Run query 1 with service = orders and level = ERROR. Expand an error line
   and find its item_id field.
2. Run query 9 with field = item_id, service = orders, level = ERROR and
   range = 5m. It counts the errors for each item.

The password is the only item_id in that result.`,
			hash: "c4657c3c1e79cf6430217c38118f496adb1e8a367b64708a624fb52a83df1684",
		},
		{
			title: "4. Mode B",
			mode:  "B",
			body: `Orders start failing again. Look at the p95 latency panel: which service's
line climbs highest?

The password is the name of that service. To see it in the logs, run query 5
with that service and ms = 2000.`,
			hash: "f7ec7e6cca5c4a3e752500096e2c25d60bd10750ec7d55ed7d251f714eb3aca1",
		},
		{
			title: "5. Mode C",
			mode:  "C",
			body: `Orders start failing again. A yellow marker appears on the dashboard's
panels just before the failures start (the Deploys toggle at the top shows
or hides these markers). Hover over that marker.

The password is the version it shows.`,
			hash: "1fc0eb8985a0492239a3768ff6becc48a4eee15f7dc25661f8a5637ac80070aa",
		},
		{
			title: "6. Stretch: the noisy warning",
			body: `Run query 10 with level = WARN and range = 1m, and switch the result to
Table. The top message shows up all the time, even when nothing is broken.
Query 6 hides it.

The password is the config key that message names.`,
			hash: "1b5281026d200d6f06bf15b0dd427c9c6c676a93c604968557bb233354e9ff6a",
		},
	},
	closing: `That's all six. Tell the facilitator you're done. While you wait, compare
with another pair: which cheat-sheet query helped you most?`,
}
