---
status: draft
since: 2026-11-03
derived-from:
  - M-02-concept-of-operations
---

# ConOps Story: Testing a Partial Recovery

Suppose Git pushes are failing and Actions workflows are stuck in the queue.
Pushes start working again while workflows are still stuck.
Anyone checking the [GitHub Status page from M-08][M-08 § Setting] before rerunning a workflow still needs to see the Actions warning.

## Setting

Every minute, a Go program called the reporter asks each service over HTTP how it is doing.
It sends the answers to the status backend, which supplies the list of services and the overall status that the dashboard reads through the gateway.
The backend is supposed to remember the last status it received for each service.
If the reporter says only that Git Operations has recovered, the Actions warning should stay on the page until the reporter says Actions has recovered too.

## Watching the warnings clear

Before releasing changes to the status page, two engineers check what a visitor would see during that incident.
They use a private copy of the page.
One engineer controls the replies the reporter gets by editing two files served over HTTP, one for Git Operations and one for Actions.
The other watches the page.

```text
Time ────────────► Both in trouble ───► Git recovers ──────► Actions recovers
Git Operations     Degraded             Operational          Operational
Actions            Degraded             Degraded             Operational
Warnings on page   Git and Actions      Actions only         None
Overall status     Disruption           Disruption           All clear
```

The first engineer sets both files to “degraded” and waits for the second engineer to see both warnings.
Then they change the Git file to “operational” and leave the Actions file alone.
The second engineer refreshes until the page lists Git Operations as operational and checks that *on that same refresh* it still warns about Actions.
Only then does the first engineer change the Actions file so they can check the final all-clear.

## Turning the page check into a Go test

Maya turns the two engineers' check into a Go test called `TestPartialRecovery`.
She uses the toolkit to start the real reporter, status backend and gateway together in the test process.
A small HTTP server supplies the two replies that the engineers used to set by hand.
Maya sets the reporter to check several times a second during the test.
The test can keep Actions degraded for as long as it takes to see Git Operations recover.

The test reads through the gateway as a signed-in client, just as the dashboard would.
It waits for both warnings before changing the Git reply.
As soon as a response lists Git Operations as operational, the test checks that Actions is still listed as degraded and the page still reports a disruption.
Only after that check passes does the test let Actions recover.
It then checks that both services are operational and the page gives an all-clear.
If the gateway keeps returning the old status, the test fails when its deadline expires.
After each run it stops the three programs, waits for them to finish and closes the HTTP server.
The test passes with the programs the team runs today and goes into CI for future changes.

## How a smaller report hides the Actions warning

The reporter sends the whole list every minute, even when nothing has changed.
The team asks a coding agent to reduce that repeated traffic while preserving the status information shown to visitors.
The agent changes the reporter's code so it sends each service's status once and sends it again only when that status changes.
The reporter's own tests pass.
When only Git Operations recovers, the new reporter tells the backend that Git Operations is operational and says nothing about Actions.

The system test fails at the middle step.
The gateway lists Git Operations as operational, leaves Actions off the list and gives an all-clear.
The agent runs `go test -run TestPartialRecovery` in its checkout and gets the same wrong answer.
On the next run it also reads the status backend directly at the failing step.
Actions is already missing there.

The backend code throws away its list whenever a report arrives and builds a new list from that report alone.
This worked as long as the reporter always named every service.
Now the list has just one entry, Git Operations, so every service on the list looks healthy.
The backend has lost the Actions warning without ever receiving word that workflows recovered.

The agent changes the backend to keep each service's last known status until a report names that service again.
Maya runs the recovery test against the corrected backend twice, once with the old reporter and once with the new one.
Both times the Actions warning stays up until workflows recover.
She can review the failing and passing responses alongside the code, without having to match a staging deployment to a build in the team's GitOps repository.

Those runs give Maya grounds to release the corrected backend first.
Once the team confirms it is serving the live page, they can deploy the reporter that sends shorter reports.
The engineers responsible for the release still check what the live gateway serves, as described in [M-01 § System boundary and operating model].

[M-08 § Setting]: M-08-debugging-the-status-gateway.md#setting
[M-01 § System boundary and operating model]: M-01-business-mission-analysis.md#system-boundary-and-operating-model
