package main

// The worksheet's scenarios. Scenario 1 is in materials/worksheet.md; each
// answer unlocks the scenario after it, and the last one unlocks the closing
// message. Each scenario practises one part of Grafana, and they get harder
// in order: Explore, a field filter, a dashboard panel, annotations, a data
// link and a metric query, the Table view. Only the hashes of the answers are
// kept (see answerHash).
var labWorksheet = worksheet{
	questions: []question{
		{
			title: "1. A customer calls support",
			hash:  "83225a4843d7c44ae7803ea4a14688a795ccbe9b5dfc43e6da696ea7bcb394ba",
		},
		{
			title: "2. Where did the order go?",
			body: `Support has the customer's request ID now, and asks: which services did
that order pass through?

In Explore, find a line whose endpoint is POST /orders, expand it, and copy
the ID from scenario 1. Run query 7 with that ID. It searches every service
at once, so you get two lines: one from orders, and one from the service
that orders called to finish the order.

Grafana skill: filtering logs on a field's value.

The password is the name of that other service.`,
			hash: "591926df98b0bd032a92ea9800c354fdb6a11d31ad3bcab7bdcecff3ff9969dc",
		},
		{
			title: "3. The shop feels sluggish",
			mode:  "B",
			body: `Customers say checkout takes forever, and some orders fail. Someone needs
to work out which service is holding everyone up.

Open the Lab Overview dashboard (Dashboards in the left menu). The p95
latency panel shows how long the slowest requests take, per service.

- Hover over the lines: the tooltip shows each service's value.
- Click a service's name in the legend under the panel to show only its
  line. Click it again to show every line.

Grafana skill: reading a dashboard panel, its tooltip and its legend.

The password is the name of the service whose line climbs highest.`,
			hash: "abe256fef7d5192604a987d77f6a189580b5953a7078a2309196b984ad957f11",
		},
		{
			title: "4. Did something just change?",
			mode:  "C",
			body: `Orders fail again, out of nowhere. When that happens, a good first
question is "did someone just ship a change?" Grafana can mark those
moments on a dashboard, as annotations.

On Lab Overview, check that the Deploys toggle at the top is on. A yellow
marker appears at the bottom of each panel just before the failures start.
Hover over it to see what it records.

Grafana skill: annotations.

The password is the version the marker shows.`,
			hash: "e666d985086a8a1f5e4e682d6d23e01ba64ef64406839b9ec5afa340344557b4",
		},
		{
			title: "5. One unlucky product",
			mode:  "A",
			body: `Orders fail again, but support says only some customers are affected.
Find out what the failing orders have in common.

1. On Lab Overview, click "orders POST /orders" in the legend of the
   Error rate panel, then click a point on its line. Choose "Logs for
   orders": Grafana opens Explore with that service's logs.
2. Change the query to query 1 with service = orders and level = ERROR.
   Expand an error line and find its item_id field.
3. Run query 9 with field = item_id, service = orders, level = ERROR and
   range = 5m. It counts the errors for each item. Switch the result to
   Table to read it.

Grafana skill: jumping from a dashboard to the logs with a data link, and
turning logs into numbers.

The password is the only item_id in that result.`,
			hash: "34f82c78c8367452a1fb03729101f297963b3a5830c2b626b41354b9fc65c2c0",
		},
		{
			title: "6. Stretch: the warning that cries wolf",
			body: `The lab is back to normal, yet the on-call engineer says one warning floods
the logs all day. Before anyone hides it, they want to know what it is.

In Explore, run query 10 with level = WARN and range = 1m, and switch the
result to Table. The top message shows up all the time, even when nothing
is broken. Query 6 hides it, so the warnings that matter stand out.

Grafana skill: counting log lines, and the Table view.

The password is the config key that message names.`,
			hash: "1b5281026d200d6f06bf15b0dd427c9c6c676a93c604968557bb233354e9ff6a",
		},
	},
	closing: `That's all six. Along the way you used Explore, log details, a dashboard
panel, annotations, a data link and the Table view.

Tell the facilitator you're done. While you wait, compare with another pair:
which part of Grafana helped you most?`,
}
