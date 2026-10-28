## 1. Proposal

- [x] 1.1 Define and validate the proposal, design and executable acceptance scenarios.

## 2. Implementation

- [x] 2.1 Extend browser preferences with isolated, backward-compatible table settings.
- [x] 2.2 Implement content-aware widths, pointer/keyboard resizing, cancellation and auto-fit/reset.
- [x] 2.3 Implement two-key heading sorting with clear direction/priority feedback, stable ties and existing resource search semantics.
- [x] 2.4 Document the controls and preference behavior.

## 3. Verification

- [x] 3.1 Add browser coverage for sizing, pointer/keyboard controls, two-level sorting, missing/equal values, search and safe text.
- [x] 3.2 Cover persistence, existing preferences, invalid/unavailable storage, future schemas, cross-tab updates and mobile/light/dark layout.
- [x] 3.3 Pass OpenSpec validation, Go/race tests and the complete browser suite; inspect a representative table preview.

## 4. Publication and deployment

- [x] 4.1 Commit and publish source; verify hosted checks, image source/platform and anonymous digest pull.
- [x] 4.2 Update and push only the GitOps image pin and release documentation, retaining unrelated local changes and the published chart.
- [x] 4.3 Verify the ready deployment, live HTTPS assets and real browser resize/sort/reload behavior; record the result and complete this checklist.

## Verification record

Completed on 2026-10-28. Source `df4963214482744cc75c44907811c31c6dd4ccc2`
passed Go/race tests and all 59 browser checks locally and in the
[release workflow](https://github.com/tintii/goK8sAutoDocs/actions/runs/40638468969).
The native amd64 image passed the read-only/non-root smoke check; its revision,
platform and anonymous digest pull were verified:
`sha256:f660d9f2eeedbc089c371b9382c7e00f5a986d0ca650074526663d97e8c9630c`.

GitOps deployment commit `364ef22` retained published workload chart 0.4.0.
The deployment has one ready replica with no restarts. All three HTTPS pages
and five assets match source. The live API and table contain 43 workloads,
16 namespaces and 63 containers. A real browser verified content sizing,
Namespace then Resource sorting, pointer resizing of only one column,
width/sort persistence after reload, Fit columns, Clear sort and internal
mobile scrolling with intact kind/type labels. The existing HTTPRoute backend
defaulting remains the only AutoDocs OutOfSync resource. Unrelated local
chart changes were preserved and were not published.
