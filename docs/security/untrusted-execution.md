# Containing Untrusted Assessment Execution

## Why this document exists

Professor grades applicant submissions by running them. `pnpm install` executes
the submission's `package.json` lifecycle scripts, and `vite build` executes its
`vite.config.ts`. Both are **arbitrary code execution by an untrusted party, by
design** — there is no way to "test" a web app without running its code.

The security question is therefore not *whether* attacker code runs (it does)
but *what that code can reach*. This document records the threat model, the
hardening implemented in the single-service design, an **honest platform
limitation** that single-service hardening cannot overcome on Cloud Run, and the
runner/reporter split kept as future work.

## Threat model

- **Attacker-controlled:** the assessment repo (`RepoURL`) and everything its
  code does once executing inside the container.
- **Trusted:** the test repo, the Cloud Tasks caller, the Professor binary.
- **Assets to protect:**
  1. `GITHUB_PAT` (clones repos) — theft enables access to whatever the PAT can
     reach.
  2. Firestore grading data — tampering enables grade forgery and cross-applicant
     data access.
  3. Integrity of the trusted test repo — tampering enables grade forgery.

## Chosen direction: single-service process restriction

We keep Professor as one Cloud Run service and harden the process boundary
around untrusted code, rather than splitting into two services. The runner/
reporter split (below) is recorded as future work.

### Implemented in this change set

- **Non-root runtime user** (`Dockerfile`): the container runs as uid 10001, not
  root. Limits the blast radius of a container escape and stops untrusted code
  overwriting the `professor` binary or other container files. Playwright
  browsers live in a shared, world-readable `/ms-playwright`
  (`PLAYWRIGHT_BROWSERS_PATH`) so the non-root user can still run them.
- **Subprocess env allowlist** (`util/safeenv.go`, pre-existing): `GITHUB_PAT`
  and any `*TOKEN*/*SECRET*/*KEY*/*AUTH*/*PASSWORD*` var is stripped from the
  environment handed to `pnpm`/`tsc`/`vite`/`playwright`. Untrusted code cannot
  read the PAT from its own environment.
- **Execution timeouts** (`builder/builder.go`, `playwright/testrunner.go`):
  install (5m), build (5m), extract-tests (2m), and the pre-existing test run
  (10m) are killed on deadline. Bounds hang/CPU-based cost DoS.
- **Filesystem separation** (`routes/grade/job.go`): the assessment and test
  trees are cloned into **separate temp roots**, not siblings under one parent,
  removing the naive `../tests` traversal. The assessment is exposed to the tests
  only as static files over HTTP, never imported into the test process.
- **Reduced Node-side execution surface** (`builder/builder.go`):
  - `pnpm install` runs with `--ignore-scripts`, blocking the assessment's root
    lifecycle scripts (`preinstall`/`install`/`postinstall`/`prepare`) — the
    dominant Node-side RCE vector.
  - Before building, the trusted test repo's `build_config/` directory is
    overlaid onto the assessment, so the build runs against grader-controlled
    config (`vite.config`, `tsconfig`, `postcss`/`tailwind` config, …) instead of
    the submission's own, which would otherwise execute as Node code during the
    build. Symlinks in the overlay are skipped so it can't redirect writes.

  See "Why Node-side execution is the real risk" below.

### Why Node-side execution is the real risk (and browser-side isn't)

Attacker code runs in two places: **Node** (during `pnpm install` and
`vite build`) and the **browser** (the built app, loaded by Playwright). Only the
Node side is a practical credential-theft vector:

- The **browser** app can *attempt* to `fetch` the metadata endpoint, but the
  metadata server requires the `Metadata-Flavor: Google` request header (a custom
  header, which triggers a CORS preflight the metadata server rejects) and returns
  no CORS headers, so browser JS cannot read a token response. The headless
  browser also has no access to the container filesystem or the PAT. So
  browser-side attacker code is effectively contained by the browser sandbox.
- The **Node** side (install/build) has full container network + filesystem
  access — this is where F1/F2 live.

Therefore shrinking Node-side execution directly shrinks F1/F2. Note that
"overwrite `vite.config`" *alone* is insufficient: `pnpm install` root scripts
run first (dominant RCE), and other Node-executed configs (`postcss.config`,
`tailwind.config`, `babel.config`) also run at build. Both are addressed:
`--ignore-scripts` closes the install-script vector, and the `build_config/`
overlay replaces *all* the submission's build-config files with the grader's, so
no attacker-authored config executes.

**Residual (known):** the build still invokes the build binaries and plugins from
the submission's own `node_modules` (`node_modules/.bin/vite`, `.../tsc`). With
`--ignore-scripts` these can't run install hooks, but a submission could still
alias e.g. `"vite"` to a malicious package so the invoked binary is attacker
code. Fully closing this requires running the **trusted test repo's** toolchain
(`vite`/`tsc` and the plugins its config imports) against the assessment source,
rather than the submission's binaries. Recommended next step if the residual is
unacceptable. Regardless, surface reduction is a hardening layer, not a proof of
zero execution for arbitrary submissions, so the containment controls above
remain necessary.

### The metadata limitation (important, and honest)

The highest-severity path (F1) is: untrusted code performs an HTTP GET to
`http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token`
and mints the service account's OAuth token, which grants Firestore write and
Secret Manager read (the PAT).

**This path cannot be fully closed inside a single Cloud Run container:**

- The metadata endpoint (`169.254.169.254`) is **link-local**, served by the
  environment. It is *not* routed through the VPC, so VPC firewall / egress rules
  and Cloud NAT **cannot block it**.
- Blocking it in-container would require an `iptables` rule (e.g. an owner-match
  on the untrusted uid), which needs `CAP_NET_ADMIN`. **Cloud Run containers are
  unprivileged and cannot be granted added Linux capabilities**, so this is not
  available.
- A nested sandbox with its own network namespace (nsjail/bubblewrap) needs
  either root/`CAP_SYS_ADMIN` or unprivileged user namespaces; availability on
  the Cloud Run gen2 execution environment is not guaranteed and must be
  validated before being relied upon.

Consequently, in the single-service model the token remains *reachable* by
untrusted code. The mitigations we *can* apply reduce what a stolen token is
worth, but do not eliminate the theft:

1. **Least-privilege the service account.** Grant `professor-service` only the
   exact roles it needs, on the narrowest resources. Note Firestore IAM is not
   collection-scoped — to bound cross-applicant blast radius, host grading data
   in a **dedicated Firestore database/project** whose SA cannot touch other
   applicant systems.
2. **Least-privilege the PAT** — the single most effective, cheapest control.
   Use a **fine-grained, read-only** PAT scoped to *only* the specific test/
   template repositories it must clone, with a short expiry and rotation. A
   stolen fine-grained read-only PAT is far less damaging than a stolen classic
   `repo`-scoped one.
3. **Minimize Secret Manager exposure.** `--set-secrets` requires the runtime SA
   to hold `secretmanager.secretAccessor`, so the metadata token can re-read the
   PAT. Accept this only in combination with (2); a minimal PAT makes re-reading
   it low-value.
4. **Restrict non-metadata egress** (F9). Route egress through a VPC connector
   with `--vpc-egress=all-traffic` and a firewall/Cloud NAT allowlist
   (`github.com`, `registry.npmjs.org`) to blunt exfiltration/C2 — with the
   explicit caveat that this does **not** affect the link-local metadata
   endpoint.

**Bottom line for F1 under single-service:** hardening substantially lowers the
*value* of what untrusted code can steal, but on Cloud Run it cannot make the SA
token *unreachable*. Fully eliminating F1 requires running untrusted code under
an identity with no useful privileges — the split below.

### F2 residual under single-service

Separate temp roots defeat the naive `../tests` write, but both trees are owned
by the same (non-root) uid on a shared filesystem, so untrusted code that
enumerates `$TMPDIR` can still locate the test tree, and a process backgrounded
during install can act after install returns. Closing this within one service
requires running the untrusted steps as a **distinct uid** and giving the test
tree `0700` ownership under a different uid — which in turn requires the
Professor process to start as root and drop privileges per step (in tension with
the uniform non-root `USER`), or Linux mount/user namespaces. Track as a
follow-up if single-service is retained.

## Future work: runner/reporter split (the complete fix)

The robust design makes the stolen metadata token **worthless** rather than
merely harder to reach:

- **Untrusted runner** — a separate Cloud Run service (or job) whose service
  account holds **no IAM roles** and no PAT. It clones the assessment, runs
  install/build/serve/test, and emits raw NDJSON results. If its metadata token
  is stolen, it grants nothing; if its filesystem is tampered with, only its own
  throwaway workspace is affected.
- **Privileged reporter/orchestrator** — never executes untrusted code. Holds
  the PAT and Firestore write access, validates the runner's results, and writes
  grades.

This removes co-residency of untrusted code with credentials entirely and is the
recommended target if the single-service residual risks above are unacceptable.

## Verification checklist

- Non-root: `docker run --rm <image> id -u` → `10001`; a grade job still
  completes end-to-end (browsers found via `/ms-playwright`).
- Timeouts: `go test ./builder/` covers the deadline-kill path.
- Fail-closed size gate: `go test ./git/ -run TestGetRepoSizeKB`.
- FS separation: `professor grade` locally; confirm assessment and test trees
  are under distinct temp roots.
- SA/PAT least-privilege and egress allowlist: verified in GCP IAM / networking
  config, not in this repo.
