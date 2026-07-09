/*
Package solution provides constructs to for solution-engineering of backend
applications that comply with this module.

# Stakeholders

# From Design to Deployment

There are several points in time wherein code is written and runs:

 1. Developers WRITE application code and publish it as catalogues.

 2. Solution engineers DESIGN solutions, composed of application elements in
    published catalogues.

 3. Platform engineers BUILD a deployment system matching the company's needs.

 4. Solution engineers pass design artefacts to the DEPLOYMENT system.

 5. The deployment system ENACTS the desired state of the deployed system.

The main innovation of this module is the separation of concerns between
developers, solution engineers, and platform.

# Example Systems

While the deployment needs of companies vary greatly, we can think of a few
archetype systems. Though each company may customize the fine details of each,
the overall lifecycle and underlying constructs remain the same.

  - Apps running on Kubernetes
  - Linux services, managed by systemd.
  - Playground host serving the entire solution in a single process.
  - End-to-end testing harness.

Even just these four cover a huge range of deployment needs. A single company
may use all four. Nonetheless, the applications powering the software solution
effectively behave the same. This indifference to deployment systems means that
software developers can and should write the same code regardless.

Consider a software company with a managed Kubernetes powering the cloud
offering, yet for on-premises customers the company hosts on a Linux machine
with systemd. Internally, that company proves solutions on developer machines
using an end-to-end testing harness in a single process, while that same
multi-hosting ability is used to enable demonstration by the sales team on their
laptops.

## Analysis of Deployments

As seen by the vast differences in the four archetypes of deployment
environments, it is not trivial to find effective boundaries between the phases
(i.e. app coding, solution definition, deployment). This sections tries to draw
some meaningful boundaries.

We've established that applications shall (for the most part) remain agnostic to
the deployment environments in which they may/will operate - we say that they
have "deployment indifference". Technically, we have designed how to share
information between the deployment environment and apps' code such that it may
specialize for environments that are known at coding time. With this "backdoor"
in mind, we can safely assume total indifference.

The solution definition carries the desire/intent of technical people (solution
engineers) to deploy software in a way that serves a specific purpose. When the
same software is composed differently, it behaves differently, thus serving
different purposes. The exact same solution definitions may be deployable in
multiple environments (e.g. Kubernetes, OpenShift, etc.), but there's a
possibility we cannot yet rule out, that certain knowledge of the deployment
environment is required while defining a solution. For example, knowledge of how
secrets are managed and passed to running processes. However, the goal of this
project is to minimize the amount of deployment knowledge required to define a
solution. For example, how a process eventually runs (e.g. k8s pod with a
side-car, Docker container, systemd service, manual invocation), and how that
Unix process knows to run the appropriate application, is within the bounds of
the deployment environment, and irrelevant when designing the solution.

To keep all this invariance, a deployment environment must resolve all the
knowledge required to run the application that together satisfy the solution.

For example, in containerized environments, which containers run which
applications; do these containers require specific config to trigger the correct
application; and how are the applications configured.

With all the wrappers, ultimately, applications must run in a Unix process. But
the harness may span more than just the part from Go's func main to calling the
app's entrypoint function. We call the in-process part of the harness "host".

So, what we require now is a list of data the deployment environment resolves
and provides to the host.

### Case #1: providing secrets, with NATS client as a use-case

To securely manage NATS authentication secrets in production Go applications, we
avoid hardcoding credentials and instead decouple them from your codebase using
the client's flexible configuration surface:

  - Static tokens, usernames, and passwords should be injected via environment
    variables or fetched at runtime from dedicated secret managers like HashiCorp
    Vault or AWS Secrets Manager.

  - When leveraging NATS’ advanced decentralized security (JWT and NKeys), we use
    the UserCredentials helper to read wrapped .creds files directly from secure,
    ephemeral filesystem volumes ensuring that private seed keys never leave the
    client container (e.g. Kubernetes Secrets or Nomad variables).

  - For organizations prioritizing transport-layer identity, mutual TLS (mTLS) can
    externalize secret management entirely to automated PKI infrastructure by
    binding short-lived client certificates and private keys directly into the
    tls.Config surface.

So, let's consider a specific application. Its code must have explicitly defined
which authentication mechainsms it supports, and how it gets the appropriate
secret values. For our case, we support both a TOKEN from env-var and
decentralized auth from a file at one of the hard-coded paths.

We rather inputs of applications don't go unnoticed, so the application library
supports flags as the means to define an app's configuration surface explicitly.
We assume this specific app has a token flag, though paths are hard-coded. (TODO: consider app that reads env without flag, like AWS SDK, but still needs to be explicitly surfaced for documentation somehow).

Then, the company's Kubernetes deployment environment must collaborate with that specific application. It must also bridge some knowledge gaps, like where are secrets stored, and which specific secret is relevant here.
*/
package solution
