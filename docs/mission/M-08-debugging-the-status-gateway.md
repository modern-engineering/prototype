---
status: draft
since: 2026-11-03
derived-from:
  - M-02-concept-of-operations
---

# ConOps Story: Debugging the Status Gateway

This story imagines the team behind [GitHub Status] asking Maya to lift the page's refresh limit for signed-in visitors.
She wants to try the change in her browser before anyone deploys it.
She can do that with two programs on her laptop, as described in [M-02 § Engineering GitHub Status].

## Setting

When someone opens the dashboard, a Go HTTP server sends their browser a static site embedded in the server executable.
To display current information, the browser requests a status report from a second Go program, the gateway.
The gateway checks the visitor's quota before it asks the status backend for that report.

## Why does a signed-in visitor still hit the limit?

Maya uses the toolkit to start the dashboard server and gateway together in a short Go program, using the same packages the team deploys.
She runs that program under Delve, the Go debugger.
Her browser loads the dashboard from her laptop and sends its requests to her local gateway.
The gateway accepts her staging test account and reads current reports directly from the staging status backend.

Maya sets the local guest quota to two requests per address per hour.
The gateway keeps its count in memory.
In a private browser window, her first two guest requests return the current report.
The third gets HTTP 429, “Too Many Requests.”
She leaves that window open, signs in through her regular window and tries again from the same address.
She still gets a 429.

The local gateway log says the request used the guest quota.
Maya sets a breakpoint where the gateway checks the quota and refreshes the signed-in window.
She sees the gateway reject her request before it has even checked who she is.
She changes the code to check credentials first and count a request only if it comes from a guest.

Maya stops the local program and both servers release their ports.
She starts it again with her edit, on the same ports.
The gateway's guest count is back at zero and her regular browser window is still signed in.
In the guest window she gets two reports and then a 429 again.
This time she sees the signed-in request skip the quota check and get the report from staging.
One more guest refresh still gets a 429.

## What the local run saved

Restarting the gateway gave Maya a fresh quota at the same address as often as she needed it.
She could repeat the failure immediately without waiting an hour or asking anyone to clear a counter on a shared gateway.
She could also stop at a breakpoint for as long as she needed while staging kept serving other users.

To try each edit on staging, the team would have had to build the gateway in CI, select that build in a separate GitOps repository and wait for deployment.
On the laptop, Maya could see which code had rejected her request and try the correction on her next run.
A coding agent with the same staging access can use that Go program to start from zero, reproduce a 429, edit the gateway and check both browser cases.
If the first edit fails, the agent can keep investigating without asking someone which build or settings are running in staging.
The team can compare the time to a working fix, the help the agent needed and the cost of maintaining the local program, as described in [M-01 § Mission and opportunity].

[GitHub Status]: https://www.githubstatus.com/
[M-02 § Engineering GitHub Status]: M-02-concept-of-operations.md#engineering-github-status
[M-01 § Mission and opportunity]: M-01-business-mission-analysis.md#mission-and-opportunity
