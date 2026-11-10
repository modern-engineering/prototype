---
status: draft
since: 2026-11-03
derived-from:
  - M-02-concept-of-operations
---

# ConOps Story: Probing Git Operations

A developer pushes a fix and Git says the push succeeded.
A colleague fetches the same branch and gets the commit from before the fix.
When they report the problem, the operator on duty checks whether it is happening on the team's own branch too.

## Setting

The [GitHub Status page in this story][M-08 § Setting] shows [the last reported condition of each service][M-09 § Setting].
To help decide whether to warn other developers about delayed fetches, the operators run a Go program called a prober.
It pushes and fetches on its own private repository even between incidents.
The operators deploy and supervise it alongside their other services.
They use the toolkit to start it with a Git checkout, access to the public Git endpoint and read-only access to the repository service.
Only the prober writes to its branch.

## Following one push

The prober's branch is at commit A.
The prober makes commit B, pushes it to the same Git endpoint developers use and records when the push succeeds.
It asks the repository service directly which commit is at the head of its branch.
The answer is B.
It then fetches the branch from the public endpoint into an empty repository.
The fetch returns A.

The operators expect a new commit to be visible from the prober's location within a minute.
After the minute is up, the prober repeats both reads.
The repository service still says B and a fresh public fetch still returns A.
The prober reports the delay, leaves the branch at B and keeps checking for that commit.

The operator tells the Git service engineer which repository and branch to look at, which two commits the prober saw and when the push succeeded.
The engineer uses them to trace how Git served those fetches.
So far the operator has seen the delay on one public route and wants to know how widely it affects developers before changing the status page.

Eventually a fresh fetch returns B.
The prober reports the time of that first matching fetch and moves on to the next push.
The operator asks the colleague who saw the old commit to fetch their own branch again.
This time the colleague gets the fix and confirms it with the operator, who closes their report.

## Running the prober in development

The Git engineer wants to know whether the code that served those fetches can return an old commit when it has just one copy of the repository to read.
A coding agent uses the toolkit to start that version and the same Go prober together in one local process.
The agent gives the prober its own local repository and credentials and points both its Git requests and its private reads at the local service.
For several pushes in a row, a fresh fetch gets the new commit within the minute.
The agent stops both programs and waits for them to finish before releasing their working directories.

The engineer can repeat those pushes and reads locally as the fetch code changes.
So far only the fetches through the production route have shown the delay.
The deployed prober keeps checking that route.

[M-08 § Setting]: M-08-debugging-the-status-gateway.md#setting
[M-09 § Setting]: M-09-testing-a-partial-recovery.md#setting
