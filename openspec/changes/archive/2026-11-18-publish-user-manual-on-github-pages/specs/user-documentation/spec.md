## ADDED Requirements

### Requirement: Concise getting-started README
The README SHALL be at most 150 lines and contain a short product description, core features, prerequisites, one verified quick-start path, the first browser URL and links to the manual, troubleshooting, release history and contribution information. Detailed configuration, API payloads and implementation walkthroughs SHALL live outside the README.

#### Scenario: First local run
- **WHEN** a new user with the documented Go toolchain and Kubernetes credentials follows the quick start from a clean checkout
- **THEN** the commands start AutoDocs with its static assets and identify the URL and expected first-use result
- **AND** the user does not need a sibling homelab checkout or the development test suite

### Requirement: Task-oriented user manual
The repository SHALL maintain its user manual as Markdown under `docs/`, with an index and navigable coverage of installation, configuration, dashboard use, endpoint publication, direct and Rancher cluster discovery, permissions, operations, troubleshooting and API reference. Instructions SHALL describe supported current behavior, prerequisites and expected outcomes; unreleased behavior SHALL be identified.

#### Scenario: Configure an additional cluster
- **WHEN** an operator follows the cluster-discovery guide
- **THEN** they can find the configuration example, required permissions, credential mount/rotation contract and verification or failure diagnostics without reading implementation proposals

#### Scenario: Use and diagnose inventory
- **WHEN** a user needs to filter/group inventory, arrange the Versions table or understand stale results
- **THEN** the manual explains those tasks and distinguishes browser preferences from shared refresh and persistence behavior

### Requirement: Separate implementation records from user guidance
Implementation rationale, source structure, algorithms, build/publishing internals and development validation history SHALL be recorded in relevant OpenSpec changes or their archive. Release history SHALL remain in CHANGELOG. The user manual SHALL retain technical details necessary to install, configure, operate or integrate with the application.

#### Scenario: Migrate existing README content
- **WHEN** the README is shortened
- **THEN** every original section has a recorded destination or explicit obsolete-content disposition
- **AND** useful user instructions remain accessible in the manual while implementation material is linked to its OpenSpec record

### Requirement: Usable published manual
The manual SHALL provide navigation and search, readable desktop/mobile pages, valid internal page/fragment/asset links under the GitHub Pages project path and usable source Markdown on GitHub. Published examples SHALL use generic sample data and placeholders instead of credentials or captured private inventory.

#### Scenario: Open a nested page directly
- **WHEN** a reader loads a nested manual URL under `/goK8sAutoDocs/`
- **THEN** the page, navigation, styles and search assets resolve correctly and links to other manual pages remain within the project site

#### Scenario: Read documentation in the repository
- **WHEN** a reader follows the README's source-manual link before Pages is published
- **THEN** `docs/index.md` and its relative Markdown links provide access to the manual
