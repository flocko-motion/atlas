/**
 * package: core / data
 * type:    interface
 * job:     the data-source port — where claims come from
 * limits:  contract only; backends are the mock generator and a REST connection (-> core/mock)
 *
 * A mock's "server details" are the generator's parameters, so configuring either is the
 * same act: one data path, not a real and a test one. `RestSource.fetch` is not wired to
 * the query contract yet; `health` works.
 */

import { generate } from '../mock/generate.ts';
import type { ClaimId, MockArchive, MockClaim } from '../mock/model.ts';
import { ARCHIVE_SCOPE } from '../scope.ts';
import type { Scope } from '../scope.ts';
import { authHeaders, endpoint, probe } from '../connections.ts';
import type { Connection, MockParams, ProbeResult } from '../connections.ts';

/** What a read returns: claims, and what the source can say about them. */
export interface ClaimPage {
  claims: MockClaim[];
  contributions: number;
  /** Time the source spent producing the page. */
  elapsedMs: number;
  /** How the page was obtained, for the log. */
  origin: string;
}

export interface FetchRequest {
  /** Cap on claims returned — `limit.results` in the query contract. */
  limit: number;
}

export interface DataSource {
  readonly kind: 'mock' | 'rest';
  /** One line naming what this source is, shown in the UI. */
  describe(): string;
  health(): Promise<ProbeResult>;
  /**
   * branches returns every scope this source can be browsed by — each named branch, and
   * the archive when its head is known. A scope with no head is not returned: it could
   * not be a closure test.
   */
  branches(): Promise<Scope[]>;
  /**
   * scopeIds asks which claims a scope contains, as identities only —
   * `output.detail: id` in the query contract.
   *
   * The engine answers this, never the client: membership in a branch is that branch's
   * closure, which is the library's traversal. What comes back is an id set to intersect
   * with the cache, so switching scopes costs a query and a lookup rather than a re-read.
   */
  scopeIds(scope: Scope): Promise<ClaimId[]>;
  fetch(request: FetchRequest): Promise<ClaimPage>;
}

/** NotWiredError marks a capability the explorer has not bound to the contract yet. */
export class NotWiredError extends Error {
  constructor(what: string) {
    super(
      `${what} is not wired yet: the REST query contract has merged, but the explorer ` +
        'imports no generated client from it so far. Use a mock source for now.',
    );
    this.name = 'NotWiredError';
  }
}

/** MockSource generates an archive locally. Its parameters are its server details. */
export class MockSource implements DataSource {
  readonly kind = 'mock' as const;

  private params: MockParams;

  constructor(params: MockParams) {
    this.params = params;
  }

  describe(): string {
    return `generated · seed ${this.params.seed}, ~${this.params.claimsPerContribution} claims per contribution`;
  }

  /** A generator is always healthy; reporting so keeps the UI uniform. */
  async health(): Promise<ProbeResult> {
    return { state: 'ok', latencyMs: 0, detail: this.describe() };
  }

  /**
   * branches reads the branch table the generator recorded, plus the archive head that
   * indexes it. The whole archive is described, not the slice a limited read returns — a
   * branch this source lists may therefore be absent from a truncated load, which is the
   * same "listed but not here" state a real instance produces.
   */
  async branches(): Promise<Scope[]> {
    const archive = this.wholeArchive();
    return [
      { name: ARCHIVE_SCOPE, head: archive.head },
      ...Object.entries(archive.branches).map(([name, head]) => ({ name, head })),
    ];
  }

  /**
   * scopeIds reads the branch each claim was stamped with as it was generated. A claim
   * with no branch is shared — the contributors and the branch table — and belongs to
   * every scope, as it does in an archive where every claim references a contributor.
   */
  async scopeIds(scope: Scope): Promise<ClaimId[]> {
    const archive = this.wholeArchive();
    if (scope.name === ARCHIVE_SCOPE) return archive.claims.map((c) => c.id);
    return archive.claims
      .filter((c) => c.branch === scope.name || c.branch === '')
      .map((c) => c.id);
  }

  async fetch(request: FetchRequest): Promise<ClaimPage> {
    // The limit is the query's; the shape is the connection's.
    const archive = memoised(this.params, Math.min(this.params.claims, request.limit));
    return {
      claims: archive.claims,
      contributions: archive.stats.contributions,
      elapsedMs: archive.stats.generateMs,
      origin: 'generator',
    };
  }

  /** wholeArchive is this source's full archive, whatever a read of it is capped at. */
  private wholeArchive(): MockArchive {
    return memoised(this.params, this.params.claims);
  }
}

/**
 * memoised generates an archive, reusing the last one when the parameters match. The
 * generator is deterministic, so this changes no answer — it stops listing the branches
 * of a 300k-claim archive from regenerating it.
 */
let lastKey = '';
let lastArchive: MockArchive | null = null;

function memoised(params: MockParams, claims: number): MockArchive {
  const key = `${params.seed}/${params.claimsPerContribution}/${claims}`;
  if (key !== lastKey || !lastArchive) {
    lastArchive = generate(claims, {
      seed: params.seed,
      claimsPerContribution: params.claimsPerContribution,
    });
    lastKey = key;
  }
  return lastArchive;
}

/** RestSource talks to a real instance. Health only, for now. */
export class RestSource implements DataSource {
  readonly kind = 'rest' as const;

  private connection: Connection;
  private secret: string;

  constructor(connection: Connection, secret: string) {
    this.connection = connection;
    this.secret = secret;
  }

  describe(): string {
    return `${this.connection.baseUrl} · ${this.connection.authKind}`;
  }

  health(): Promise<ProbeResult> {
    return probe(this.connection, this.secret);
  }

  /**
   * branches reads `GET /branches`, the route that exists so a client need not be told a
   * branch name out of band, and `GET /archive/info` for the branch-table head — the only
   * route that reports it, and so the only way the archive becomes a nameable scope.
   *
   * The archive is dropped rather than guessed if that read fails: a scope with no head
   * cannot be a scope, and the branches are still worth returning.
   */
  async branches(): Promise<Scope[]> {
    const listed = await this.getJSON<{ branches?: { name?: string; head?: string }[] }>(
      '/branches',
      'listing branches',
    );
    const branches = (listed.branches ?? [])
      .filter((b): b is { name: string; head: string } => Boolean(b.name && b.head))
      .map(({ name, head }) => ({ name, head }));

    try {
      const archive = await this.getJSON<{ head?: string }>('/archive/info', 'reading the archive');
      if (archive.head) return [{ name: ARCHIVE_SCOPE, head: archive.head }, ...branches];
    } catch {
      // Not grantable to this subject, or an older instance: the branches still stand.
    }
    return branches;
  }

  /** getJSON reads one JSON route, naming what was being read when it fails. */
  private async getJSON<T>(path: string, what: string): Promise<T> {
    const response = await fetch(endpoint(this.connection, path), {
      headers: authHeaders(this.connection, this.secret),
    });
    if (!response.ok) {
      throw new Error(`HTTP ${response.status} ${what}`);
    }
    return (await response.json()) as T;
  }

  /**
   * scopeIds runs the scoped query the server answers: `select.branch` names the scope,
   * `output.detail: id` asks for identities only. The engine walks the closure; this parses
   * the ids it streamed back.
   *
   * The response is a JSON sequence (RFC 7464), one record per line with an optional
   * leading record separator, so it is read line by line rather than as one document.
   */
  async scopeIds(scope: Scope): Promise<ClaimId[]> {
    const response = await fetch(endpoint(this.connection, '/query'), {
      method: 'POST',
      headers: { ...authHeaders(this.connection, this.secret), 'Content-Type': 'application/json' },
      body: JSON.stringify({
        select: { branch: scope.name, head: scope.head },
        output: { detail: 'id', encoding: 'json' },
      }),
    });
    if (!response.ok) {
      throw new Error(`HTTP ${response.status} reading the ids in ${scope.name}`);
    }
    return idsFromSequence(await response.text());
  }

  /** headOf reads a branch head — the one moving target in an archive. */
  async headOf(branch: string): Promise<string> {
    const response = await fetch(
      endpoint(this.connection, `/branches/${encodeURIComponent(branch)}/head`),
      { headers: authHeaders(this.connection, this.secret) },
    );
    if (!response.ok) throw new Error(`HTTP ${response.status} reading ${branch} head`);
    // The route answers `{"head": "<id>"}`, so the id has to be read out of it — reading the
    // body as text returned the envelope.
    const body = (await response.json()) as { head?: string };
    if (typeof body.head !== 'string') throw new Error(`no head in the answer for ${branch}`);
    return body.head;
  }

  async fetch(_request: FetchRequest): Promise<ClaimPage> {
    throw new NotWiredError('reading claims from a server');
  }
}

/**
 * idsFromSequence reads the ids out of a JSON-sequence body. Each record is a claim
 * identity — a bare string, or an object carrying one — and the record separator RFC 7464
 * prefixes each with is stripped before parsing.
 */
export function idsFromSequence(body: string): ClaimId[] {
  const ids: ClaimId[] = [];
  for (const line of body.split('\n')) {
    const text = line.replace(/^\u001e/, '').trim();
    if (text === '') continue;
    let record: unknown;
    try {
      record = JSON.parse(text);
    } catch {
      continue; // a partial line, or the execution report the query may append
    }
    if (typeof record === 'string') {
      ids.push(record);
    } else if (record && typeof record === 'object') {
      const id = (record as { id?: unknown }).id;
      if (typeof id === 'string') ids.push(id);
    }
  }
  return ids;
}

/** sourceFor builds the source a connection stands for. */
export function sourceFor(connection: Connection, secret: string): DataSource {
  return connection.kind === 'mock'
    ? new MockSource(connection.mock)
    : new RestSource(connection, secret);
}
