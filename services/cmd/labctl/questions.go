package main

// The worksheet's scenarios: the tickets and alerts of one on-call shift at
// the shop. Scenario 1 is in materials/worksheet.md; each answer unlocks the
// scenario after it, and the last one unlocks the closing message. Each
// scenario ends in one direct question about what's in Grafana, and gets a
// little harder; the steps to the answer are in its three hints, each more
// specific than the last. Query numbers in the hints are the ones on
// materials/logql-cheat-sheet.md. Only the hashes of the answers are kept
// (see answerHash).
var labWorksheet = worksheet{
	questions: []question{
		{
			title: "1. A ticket from support",
			hash:  "f970ad92bce294169a31b13b2ab451c6b83e7241ac4e04487054b73042a5769e",
			hints: [3]string{
				`Open Explore on the Loki data source, and set the time picker to Last
24 hours. Explore only searches the range you pick.`,
				`Run query 4 with service = orders, field = order_id and value =
ORD-000042.`,
				`Expand the line whose endpoint is POST /orders. Its status field is the
password.`,
			},
		},
		{
			title: "2. A follow-up on the same ticket",
			body: `Support replies: "Thanks. The customer also asks whether their item was
actually checked against stock." Placing an order takes more than one
service: while handling ORD-000042, orders called another one.

What HTTP status did that other service return?`,
			hash: "33acebbbdc9fdc7c0cdd6f9ffd30c15e2b3d0f51ae2459fc8d057894727c6382",
			hints: [3]string{
				`Every request carries a request_id, and each service it passes through
logs the same one.`,
				`Copy the request_id from ORD-000042's POST /orders line, then run query 7
with it. Keep the time picker on Last 24 hours.`,
				`You get one line from orders and one from the service it called. The
status field on that other line is the password.`,
			},
		},
		{
			title: "3. An alert fires",
			mode:  "B",
			body: `In the on-call channel:

  [FIRING] Checkout latency high: orders p95 above 2s

Support adds that customers are giving up partway through checkout, and some
orders fail.

Which service is slow?`,
			hash: "abe256fef7d5192604a987d77f6a189580b5953a7078a2309196b984ad957f11",
			hints: [3]string{
				`Start on the Lab Overview dashboard. Which panel moved?`,
				`On the p95 latency panel, hover over the lines to read each service's
value, and click a service in the legend to show only its line.`,
				`Orders gives up after 2 seconds, so its line stops near 2 s. The service
whose line climbs highest is the password.`,
			},
		},
		{
			title: "4. A new alert",
			mode:  "C",
			body: `The latency alert resolves. A few minutes later, a different one fires:

  [FIRING] Orders error rate high: 5xx above 5%

This time, latency looks normal everywhere.

What changed just before the errors started? The password is the new
version number.`,
			hash: "e666d985086a8a1f5e4e682d6d23e01ba64ef64406839b9ec5afa340344557b4",
			hints: [3]string{
				`When failures start suddenly, look for a change just before them.`,
				`Look for the deploy markers (vertical lines) on Lab Overview, or run
query 4 with service = orders, field = event and value = deploy.`,
				`Hover over the marker just before the errors start, or read the deploy
line. Its version is the password.`,
			},
		},
		{
			title: "5. The tickets keep coming",
			mode:  "A",
			body: `Support forwards three tickets in a row: "My order fails every time I
retry." Other customers' orders are going through fine.

Which SKU do the failing orders have in common?`,
			hash: "34f82c78c8367452a1fb03729101f297963b3a5830c2b626b41354b9fc65c2c0",
			hints: [3]string{
				`Failures that hit some orders and not others usually share a field
value.`,
				`Run query 9 on inventory with level = ERROR, and try counting by
different fields.`,
				`Count by item_id. One SKU accounts for every failure; it's the password.`,
			},
		},
		{
			title: "6. Stretch: the shift review",
			body: `At the review after the shift, someone asks why nobody noticed the early
warnings. One warning shows up in the logs every few seconds, all day,
whether or not anything is wrong.

Which config key does that warning complain about?`,
			hash: "1b5281026d200d6f06bf15b0dd427c9c6c676a93c604968557bb233354e9ff6a",
			hints: [3]string{
				`Find the warning that shows up most often.`,
				`Run query 10 with level = WARN and range = 1m, then switch the result to
Table.`,
				`The top message names a deprecated config key. That key is the
password.`,
			},
		},
	},
	closing: `That's all six.

Before the debrief, pick an app you work on and think about:

- If it failed the way scenario 3, 4 or 5 did, how would you find out, and
  how long would it take?
- Could you trace one request through it today? If not, what's missing from
  its logs?
- What's one change you could make this week: a request ID, JSON logs, a
  version logged at startup, or a dashboard panel?`,
}
