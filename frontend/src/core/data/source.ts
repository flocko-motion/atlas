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
import type { MockClaim } from '../mock/model.ts';
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

  constructor(private params: MockParams) {}

  describe(): string {
    return `generated · seed ${this.params.seed}, ~${this.params.claimsPerContribution} claims per contribution`;
  }

  /** A generator is always healthy; reporting so keeps the UI uniform. */
  async health(): Promise<ProbeResult> {
    return { state: 'ok', latencyMs: 0, detail: this.describe() };
  }

  async fetch(request: FetchRequest): Promise<ClaimPage> {
    // The limit is the query's; the shape is the connection's.
    const claims = Math.min(this.params.claims, request.limit);
    const archive = generate(claims, {
      seed: this.params.seed,
      claimsPerContribution: this.params.claimsPerContribution,
    });
    return {
      claims: archive.claims,
      contributions: archive.stats.contributions,
      elapsedMs: archive.stats.generateMs,
      origin: 'generator',
    };
  }
}

/** RestSource talks to a real instance. Health only, for now. */
export class RestSource implements DataSource {
  readonly kind = 'rest' as const;

  constructor(
    private connection: Connection,
    private secret: string,
  ) {}

  describe(): string {
    return `${this.connection.baseUrl} · ${this.connection.authKind}`;
  }

  health(): Promise<ProbeResult> {
    return probe(this.connection, this.secret);
  }

  /** headOf reads a branch head — the one moving target in an archive. */
  async headOf(branch: string): Promise<string> {
    const response = await fetch(
      endpoint(this.connection, `/branches/${encodeURIComponent(branch)}/head`),
      { headers: authHeaders(this.connection, this.secret) },
    );
    if (!response.ok) throw new Error(`HTTP ${response.status} reading ${branch} head`);
    return (await response.text()).trim();
  }

  async fetch(_request: FetchRequest): Promise<ClaimPage> {
    throw new NotWiredError('reading claims from a server');
  }
}

/** sourceFor builds the source a connection stands for. */
export function sourceFor(connection: Connection, secret: string): DataSource {
  return connection.kind === 'mock'
    ? new MockSource(connection.mock)
    : new RestSource(connection, secret);
}
