-- Access store schema. Identity is schemaf's; we key on the opaque subject.

-- Subjects we hold a fact about (currently: disabled). Lazy — a subject with no
-- row is simply not disabled. token_epoch is reserved for future token-cohort
-- revocation; default 0 means "no revocation boundary".
CREATE TABLE ranke_subjects (
    id          TEXT PRIMARY KEY,
    disabled    BOOLEAN NOT NULL DEFAULT FALSE,
    token_epoch INTEGER NOT NULL DEFAULT 0
);

-- Grants: (subject, scope, role). scope_ra = '' means the scope is the tenant
-- itself; otherwise it is the named archive within scope_tenant. Grants are
-- additive (MySQL-style — a collection): role is part of the primary key, so a
-- subject may hold several roles on one scope (e.g. write + operator).
CREATE TABLE ranke_grants (
    subject      TEXT NOT NULL,
    scope_tenant TEXT NOT NULL,
    scope_ra     TEXT NOT NULL DEFAULT '',
    role         TEXT NOT NULL,
    PRIMARY KEY (subject, scope_tenant, scope_ra, role)
);

CREATE INDEX ranke_grants_subject_idx ON ranke_grants (subject);
