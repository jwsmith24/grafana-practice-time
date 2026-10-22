package main

// The worksheet's questions. Question 1 is in materials/worksheet.md; each
// answer unlocks the question after it, and the last one unlocks the closing
// message. Only the hashes of the answers are kept (see answerHash).
var labWorksheet = worksheet{
	questions: []question{
		{
			title: "1. Warm-up: what is in a log line?",
			hash:  "83225a4843d7c44ae7803ea4a14688a795ccbe9b5dfc43e6da696ea7bcb394ba",
		},
		{
			title: "2. Warm-up: follow one request",
			body: `Copy the request_id of one POST /orders line from orders. Find every line
in the lab with that request_id. Which services handled the request, in what
order, and how long did each one take?

The password is the name of the service the request reached second.`,
			hash: "591926df98b0bd032a92ea9800c354fdb6a11d31ad3bcab7bdcecff3ff9969dc",
		},
		{
			title: "3. Mode A",
			body: `Run ./incident.sh A. Orders start failing. Find when, where, the first
error, one request, and the root cause.

The password is the one value the failing requests share that healthy ones
don't.`,
			hash: "c4657c3c1e79cf6430217c38118f496adb1e8a367b64708a624fb52a83df1684",
		},
		{
			title: "4. Mode B",
			body: `Run ./incident.sh B. Orders start failing again. Find when, where, the
first error, one request, and the root cause.

The password is the name of the service responsible for the failures.`,
			hash: "f7ec7e6cca5c4a3e752500096e2c25d60bd10750ec7d55ed7d251f714eb3aca1",
		},
		{
			title: "5. Mode C",
			body: `Run ./incident.sh C. Orders start failing again. Find when, where, the
first error, one request, and the root cause.

The password is the version that was running when the failures began.`,
			hash: "1fc0eb8985a0492239a3768ff6becc48a4eee15f7dc25661f8a5637ac80070aa",
		},
		{
			title: "6. Stretch: the noisy warning",
			body: `One WARN line shows up all the time, whether or not anything is broken.
Which line is it, which service writes it, and how many times a minute? Write
a query that shows every WARN line in the lab except that one.

The password is the config key that line names.`,
			hash: "1b5281026d200d6f06bf15b0dd427c9c6c676a93c604968557bb233354e9ff6a",
		},
	},
	closing: `That's all six. Tell the facilitator you're done. While you wait, compare
your notes for questions 3 to 5 with another pair: did you find the same
first error and the same proof?`,
}
