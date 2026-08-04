# conformance-suite Specification

## Purpose
RankeDB ships a conformance suite that drives a series of commits and queries against
a running instance's API and asserts the results against reference values — those the
native engine produces. Run against any configuration, it validates that any
composition of adapters or execution engines reproduces the reference behaviour. This
capability is the starting point for the conformance group; engine- and
adapter-specific conformance are added alongside it.

## Requirements

### Requirement: The suite asserts a running instance against reference values
The system SHALL provide a conformance suite that runs a series of commits and
queries against a running instance's API and asserts the results against the
reference values the native engine produces.

#### Scenario: A configured instance matches the reference
- **WHEN** the suite runs its commits and queries against a running instance
- **THEN** every result matches the reference value for that operation

### Requirement: The suite is configuration-agnostic
The system SHALL allow the conformance suite to run against any server
configuration, validating that any composition of adapters or execution engines,
current or future, reproduces the reference results.

#### Scenario: A non-default composition passes the same suite
- **WHEN** the suite runs against an instance backed by a non-default adapter composition
- **THEN** it produces the same reference results as the native reference configuration

### Requirement: The ADT binary conformance suite grounds the hashes
The system SHALL inherit the foundation's binary conformance suite — example graphs
and operations with expected hashes — as the ground truth for claim ids and content
hashes, so that conformance to the ADT is decidable beneath the API-level suite.

#### Scenario: Reference hashes match the ADT vectors
- **WHEN** the suite computes claim ids and content hashes for the example graphs
- **THEN** they match the expected hashes shipped with the ADT conformance vectors
