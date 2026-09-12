---
status: accepted
since: 2026-09-12
derived-from:
  - M-02-concept-of-operations
---

# Evolving an Operational Solution

This counterfactual story follows Stripe's engineers and operators evolving a payment-processing solution that one merchant already depends on.
They must meet new needs and recover from a failed upgrade while keeping the service dependable and its intended configuration understandable.

The solution runs on Linux VMs under Stripe's control.
Stripe's [published account of its build infrastructure][Stripe-build-infrastructure] describes Linux virtual machines, not this payment deployment.
The merchant-specific solution, systemd supervision, toolkit adoption, responsibilities, capabilities and events are invented.
The lifecycle meanings follow [M-02-concept-of-operations].

## Growing beyond the classification pilot

The merchant already relies on the solution to process payments from its storefronts.
Stripe operates the applications and the VMs, handles releases and responds to incidents.
A Stripe customer-delivery lead remains responsible for the merchant relationship while different engineering and operating teams change the solution.
The merchant's staff use the payment results in their daily work; they do not administer the machines.
The merchant is the business beneficiary: its needs justify the solution and guide its changes.
The payment workers serve no other customer.

The merchant's support staff increasingly need to distinguish new purchases from renewals when investigating payment enquiries.
An experimental classification daemon covers only a limited set of payments, leaving them without classifications for the rest.
The merchant's service owner asks for that useful behaviour to become a supported part of the service across its payment workload.
The main payment application already offers the required classification capability, although it is not enabled in this solution.
Meeting the request therefore calls for a change to the merchant's solution, not development of a new application feature.

Stripe adopted the toolkit before this request; its established release, monitoring and on-call practices remain in place.
The toolkit added a handoff that carries the solution's durable identity and intended design from solution engineering into operation.
Each intended edition is a solution revision, which the solution engineer and operators can compare with the applications and versions running on the VMs.
The current revision includes the experimental daemon and two payment workers.
Classification remains disabled in the main application.
The revision history explains why the pilot is still present.

## Making the supported capability available

The solution engineer first works with the merchant's service owner to confirm that the supported classifications provide what the merchant's support staff need for their enquiries.
The engineer then prepares another solution revision with classification enabled in the main payment application.
The experimental daemon remains in the intended composition for a comparison period, allowing the merchant to check the supported output before giving up the pilot.

After Stripe approves the revision through its normal change process, the solution engineer hands it to platform engineers through the established solution-artefact path.
They check the VMs and backing services, deploy the revision through Stripe's local procedures and accept its operational readiness.

Following deployment, the merchant's support staff use the revised solution in their ordinary work.
They can distinguish the payment categories they need without relying on the pilot's limited coverage.
The merchant's service owner accepts the capability on that basis.
Stripe remains responsible for deployment and operational readiness.

Software engineers remain responsible for the reusable application, but this change needs neither a patch nor their participation in the deployment.

## Keeping payment responses timely

With classification now enabled across the merchant's workload, each payment requires additional processing.
During the next busy periods, monitoring shows the queue growing and the merchant's customers waiting longer for payment results.
Operators need to restore timely service without withdrawing the capability the merchant has just accepted.

In response, platform engineers confirm that the VMs have enough capacity and the application can run four workers.
Because worker count belongs to the solution's intended footprint, the solution engineer records the increase from two to four in another revision.

After Stripe approves the four-worker revision, operators deploy it.
The merchant does not approve this internal process count because the agreed behaviour is unchanged.
When all four workers are running, operators observe the queue receding and response times returning to the usual range.
The merchant continues receiving the supported classifications.
This becomes a known working revision in the same solution's history.

## A later upgrade interrupts payments

After the four-worker revision has remained stable, Stripe schedules a software upgrade through its normal release process.
The solution engineer then prepares a revision that selects the newer payment application.
It keeps classification enabled in the main application and retains four workers and the experimental daemon.
Once Stripe approves the upgrade, operators begin introducing the revision into the merchant's solution.

Under this workload, the upgraded workers repeatedly fail and restart.
Monitoring shows both the restart pattern and a rise in failed payment attempts, while the merchant sees customers having to try again.
The capacity that served the previous version well cannot compensate for repeated process failures.

This story assumes that a payment attempt interrupted by a restart fails visibly and can be retried.
A transaction reported as accepted does not silently disappear.
These conditions belong to the payment service, not the toolkit, and the story leaves the transaction and retry mechanisms open.
Even so, repeated failures disrupt purchases and increase the merchant's support work.

While software engineers investigate the defective version, Stripe's incident commander orders a rollback to the preceding working revision identified in the solution history.
That revision selects the earlier application and keeps classification enabled.
It also calls for four workers.

## Returning to a working revision

The rollback restores that whole revision under the solution's existing identity and leaves the failed upgrade in its history.
Operators restore the earlier application while keeping classification enabled and returning all four workers.
They do not have to reconstruct the preceding feature and capacity changes around an older binary.

For this upgrade, the earlier application remains compatible with the solution's data and backing services.
Returning to that revision neither undoes processed payments nor requires a separate data-restoration operation.

Following the incident commander's rollback decision, operators deliberately hold failing workers stopped while returning the earlier version.
The intended revision now calls for the earlier application, but the running workers reach it gradually:

| Recovery point | Intended solution | Observed operation |
| --- | --- | --- |
| Upgrade misbehaves | Upgraded version; classification enabled in the main application; four workers | Upgraded workers repeatedly fail and restart. |
| Rollback is under way | Earlier working revision; classification remains enabled; four workers | Earlier-version workers start while others remain stopped or await replacement. |
| Recovery is accepted | The same restored revision | Four workers run the earlier version; payment behaviour is back within the usual operating range. |

Stripe's monitoring shows the application version each worker is running.
The toolkit helps operators relate those observations to the intended revision and solution history.
Stripe's operating record explains deliberate differences during rollback, such as a failed upgraded worker held stopped while its replacement starts.
Stripe's operators decide whether a difference is expected or requires action, and remain responsible for any correction.
The toolkit does not make that judgment for them.

Platform engineers accept recovery after checking the running versions, process count and service behaviour.
The merchant's service owner confirms that payment use has returned to normal and the supported classifications remain available.
After recovery, software engineers continue investigating the defective upgrade.

## Ending the experiment without ending the solution

The incident review also revisits why the experimental classification daemon is still present.
It did not cause the failed upgrade, but operators had to assess its health and contribution while restoring service.
It still consumes capacity and generates alerts that operators must assess.
The comparison period has shown that the supported classification meets the merchant's need, and the merchant's service owner confirms that staff no longer depend on the experimental output.

After that confirmation, the solution engineer prepares a revision that omits the daemon.
It keeps classification enabled in the working application and retains four workers.
Once Stripe approves the removal, operators apply the revision, remove the daemon, verify its absence and accept the remaining service as operational.
The supported classifications continue to reach the merchant without the separate experiment.

## Boundaries that remain

Revision identification, ordering, branching, storage and exchange remain open.
So do SEF partitioning, solution-artefact layout and the representation of feature settings and process counts.
The story does not prescribe toolkit commands, interfaces, systemd unit generation, monitoring protocols or automatic reconciliation.
Those are later design choices; the operators here need only to know the intended solution and what is actually running.

## The solution continues

Stripe's customer-delivery lead reviews the resulting service with the merchant's service owner.
The merchant's support staff retain the broader classifications, payment response times are back within their usual range, and removing the daemon has left no service gap.
On that evidence, the lead accepts the customer follow-up as complete.

Stripe now sustains the same logical solution with one fewer daemon, while its history still explains the experiment, expansion and recovery.
Removing the experiment simplifies its operational life rather than ending it.

[M-02-concept-of-operations]: M-02-concept-of-operations.md
[Stripe-build-infrastructure]: https://stripe.dev/blog/fast-secure-builds-choose-two
