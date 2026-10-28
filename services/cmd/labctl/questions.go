package main

// The worksheet's scenarios: one on-call shift at the shop, told in six
// parts. Scenario 1 is in materials/worksheet.md; each answer unlocks the
// scenario after it, and the last one unlocks the closing message. Each part
// leans on what the one before it found, practises one part of Grafana, and
// is a little harder: Explore and the time picker, a field filter, a panel's
// tooltip and legend, annotations, a data link and a metric query, the Table
// view. Only the hashes of the answers are kept (see answerHash).
var labWorksheet = worksheet{
	questions: []question{
		{
			title: "1. A ticket from support",
			hash:  "f970ad92bce294169a31b13b2ab451c6b83e7241ac4e04487054b73042a5769e",
		},
		{
			title: "2. Behind the scenes of one order",
			body: `The order went through, so Kyle tells the customer not to pay again.
Kyle has one more question: placing an order takes more than one service, so
which other service did the shop call to finish ORD-000042? Knowing the
route a healthy order takes tells you what normal looks like, before
anything breaks.

Every request gets a request_id, and each service it passes through logs
the same one. In Explore, expand ORD-000042's POST /orders line again and
copy its request_id. Then, with the time picker still on Last 24 hours,
run query 7 with that ID. It searches every service at once, so you get
two lines: one from orders, and one from the service orders called.

Grafana skill: following one request across services with a field filter.

The password is the name of that other service.`,
			hash: "591926df98b0bd032a92ea9800c354fdb6a11d31ad3bcab7bdcecff3ff9969dc",
		},
		{
			title: "3. An alert fires: checkout is crawling",
			mode:  "B",
			body: `10:40. An alert lands in the on-call channel: "checkout is slow". Kyle
adds that customers are giving up halfway through, and some orders fail. Orders gets the blame,
but you now know orders waits on another service for every order. Is
orders itself slow, or is it waiting?

Open the Lab Overview dashboard (Dashboards in the left menu). The p95
latency panel shows how long the slowest requests take, per service.

- Hover over the lines: the tooltip shows each service's value.
- Click a service's name in the legend under the panel to show only its
  line. Click it again to show every line.

Grafana skill: reading a dashboard panel, its tooltip and its legend.

The password is the name of the service whose line climbs highest: the
team you call in next.`,
			hash: "abe256fef7d5192604a987d77f6a189580b5953a7078a2309196b984ad957f11",
		},
		{
			title: "4. It's broken again, and nobody touched inventory",
			mode:  "C",
			body: `11:15. The inventory team fixed their side, and the p95 panel looks calm.
Ten minutes later, orders start failing again. The inventory team swears
they changed nothing this time. Your teammate asks: "did anyone ship
something just now?"

Grafana can mark moments like a release on a dashboard, as annotations.
On Lab Overview, check that the Deploys toggle at the top is on. A yellow
marker appears at the bottom of each panel just before the failures
start. Hover over it to see what it records.

Grafana skill: annotations.

The password is the version the marker shows: the release you ask the
orders team to roll back.`,
			hash: "e666d985086a8a1f5e4e682d6d23e01ba64ef64406839b9ec5afa340344557b4",
		},
		{
			title: "5. The tickets keep coming",
			mode:  "A",
			body: `11:50. The rollback went out: a new marker shows the old version back in
place. But Kyle's queue is filling up again, this time with customers
whose orders fail every time they try. Most orders go through fine. What do the
failing ones have in common?

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

The password is the only item_id in that result: the product Kyle can warn
customers about while it's fixed.`,
			hash: "34f82c78c8367452a1fb03729101f297963b3a5830c2b626b41354b9fc65c2c0",
		},
		{
			title: "6. Stretch: the review after the shift",
			body: `The shop is healthy again. In the review after your shift, the team asks
why nobody spotted the trouble in the warnings earlier. You suspect one
warning floods the logs all day, so the ones that matter get lost.

In Explore, run query 10 with level = WARN and range = 1m, and switch the
result to Table. The top message shows up all the time, even when nothing
is broken. Query 6 hides it, so the warnings that matter stand out.

Grafana skill: counting log lines, and the Table view.

The password is the config key that message names: the one the team
fixes first.`,
			hash: "1b5281026d200d6f06bf15b0dd427c9c6c676a93c604968557bb233354e9ff6a",
		},
	},
	closing: `That's all six, and the end of your shift. You answered Kyle's ticket,
traced an order through the shop, found who was holding up checkout,
caught a bad release, found the product behind the failing orders and cut
the noise for next time, using Explore, log details, a dashboard panel,
annotations, a data link and the Table view.

Tell the facilitator you're done. While you wait, write the three-line
summary you'd send Kyle, and compare it with another pair's.`,
}
