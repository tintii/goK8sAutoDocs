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

- [ ] 4.1 Commit and publish source; verify hosted checks, image source/platform and anonymous digest pull.
- [ ] 4.2 Update and push only the GitOps image pin and release documentation, retaining unrelated local changes and the published chart.
- [ ] 4.3 Verify the ready deployment, live HTTPS assets and real browser resize/sort/reload behavior; record the result and complete this checklist.
