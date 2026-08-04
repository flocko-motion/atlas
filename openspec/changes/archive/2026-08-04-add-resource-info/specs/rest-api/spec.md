## ADDED Requirements

### Requirement: A branch and the archive each report what is known about them

The REST contract SHALL provide `GET /branches/{branch}/info` and `GET /archive/info`,
reporting what is known about that resource beyond the head id its `/head` form answers:
for a branch its name, head, the head claim's height and when it last moved; for the
archive the branch-table head, its height, when it last moved, and how many branches the
table holds.

`GET /archive/info` SHALL be the route by which a client obtains the **branch-table head**.
Without it the `$archive` scope — reserved by the access model, accepted by RankeQL, and
read within by `/archive/claims/{id}` — is a scope no client can name.

Each SHALL be gated by the scope it reads: **R** on the branch, and **R** on `$archive`.
Both SHALL be cacheable with revalidation, every field moving when the resource does.

A claim count SHALL NOT be among the fields. Counting the claims in a scope is a walk of
its closure, which is a query (`POST /query`), and a resource field that costs a closure
walk would hide that.

#### Scenario: A branch reports more than its head
- **WHEN** a client issues `GET /branches/{branch}/info`
- **THEN** it receives the branch's name and head with the head claim's height and the time it last moved

#### Scenario: The archive head is obtainable
- **WHEN** a client issues `GET /archive/info`
- **THEN** it receives the branch-table head, which it can then name as the `$archive` scope in a query and read by id in that scope

#### Scenario: The narrow head form is unchanged
- **WHEN** a client issues `GET /branches/{branch}/head`
- **THEN** it receives the head id alone, as before, `/info` being a second read of the same resource rather than a replacement

#### Scenario: Each info read needs its scope's right
- **WHEN** an account holding no `R $archive` grant issues `GET /archive/info`
- **THEN** the request is denied by the access decision
