/**
 * package: core / data
 * type:    interface
 * job:     the data-source port — where claims come from
 * limits:  contract only; backends are the mock generator and a REST connection (-> core/mock)
 *
 * A mock's "server details" are the generator's parameters, so configuring either is the same
 * act: one data path, not a real and a test one.
 */

import { EncodeQuery, newSeqReader } from '@flocko-motion/ranke';
import type { Query as RankeQuery } from '@flocko-motion/ranke';
import { contributionUnknown, drawn } from '../claims.ts';
import type { DrawnClaim } from '../claims.ts';
import { generate } from '../mock/generate.ts';
import type { MockArchive } from '../mock/model.ts';
import { ARCHIVE_SCOPE } from '../scope.ts';
import type { Scope } from '../scope.ts';
import { apiFor, probe } from '../connections.ts';
import type { Connection, MockParams, ProbeResult } from '../connections.ts';
import { Api } from './openapi.gen.ts';
import type { Error as ApiError, HttpResponse, Query as ApiQuery } from './openapi.gen.ts';

/** What a read returns: claims, and what the source can say about them. */
export interface ClaimPage {
  claims: DrawnClaim[];
  contributions: number;
  /** Time the source spent producing the page. */
  elapsedMs: number;
  /** How the page was obtained, for the log. */
  origin: string;
}

export interface FetchRequest {
  /** Cap on claims returned — `limit.results` in the query contract. */
  limit: number;
  /**
   * Scope to read — `select.branch`. A server read requires one, the contract making a
   * scope mandatory; a generator ignores it, having one archive.
   */
  scope?: Scope | null;
  /**
   * Called as claims arrive: how many are decoded so far, and how many bytes the reader has
   * taken. The count of claims yet to come would cost a second closure walk; the bytes are
   * the reader's own tally, so they cost nothing and move even between two whole records.
   */
  onProgress?: (read: number, bytesRead: number) => void;
}

export interface DataSource {
  readonly kind: 'mock' | 'rest';
  /** One line naming what this source is, shown in the UI. */
  describe(): string;
  health(): Promise<ProbeResult>;
  /** branches returns every scope with a head: each branch, and the archive. */
  branches(): Promise<Scope[]>;
  /**
   * scopeIds asks which claims a scope contains, as identities only (`output.detail: id`).
   * The engine answers it — a branch's membership is its closure — and the client intersects
   * the id set with its cache.
   */
  scopeIds(scope: Scope): Promise<string[]>;
  fetch(request: FetchRequest): Promise<ClaimPage>;
  /**
   * content reads one claim's bytes. Content is addressed by the claim that holds it, so the
   * scope the claim was read in is the scope its content is read in — and nothing about
   * whether the bytes sit inline or in a blob reaches the caller.
   */
  content(scope: Scope | null, id: string): Promise<Uint8Array>;
}

/** MockSource generates an archive locally. Its parameters are its server details. */
export class MockSource implements DataSource {
  readonly kind = 'mock' as const;

  private params: MockParams;

  constructor(params: MockParams) {
    this.params = params;
  }

  describe(): string {
    return (
      `generated · seed ${this.params.seed}, up to ${this.params.claims.toLocaleString('en-US')} ` +
      `claims, ~${this.params.claimsPerContribution} per contribution`
    );
  }

  /** A generator is always healthy; reporting so keeps the UI uniform. */
  async health(): Promise<ProbeResult> {
    return { state: 'ok', latencyMs: 0, detail: this.describe() };
  }

  /**
   * branches reads the branch table the generator recorded, describing the whole archive
   * rather than the slice a capped read returns — so a listed branch may be unloaded, as
   * against a real instance.
   */
  async branches(): Promise<Scope[]> {
    const archive = this.wholeArchive();
    return [
      { name: ARCHIVE_SCOPE, head: archive.head },
      ...Object.entries(archive.branches).map(([name, head]) => ({ name, head })),
    ];
  }

  /**
   * scopeIds reads the branch each claim was stamped with. An unstamped claim is shared —
   * contributors, the branch table — so it belongs to every scope.
   */
  async scopeIds(scope: Scope): Promise<string[]> {
    return scopedClaims(this.wholeArchive(), scope.name).map((c) => c.claim.id);
  }

  /**
   * fetch answers the scoped read, as the server does: the scope generates the result set and
   * the limit caps it. Capping the archive instead would return a prefix, which disagrees
   * with what `scopeIds` says the scope contains — the two have to describe one archive.
   */
  async fetch(request: FetchRequest): Promise<ClaimPage> {
    const archive = this.wholeArchive();
    const inScope = scopedClaims(archive, request.scope?.name);
    const claims = inScope.slice(0, request.limit);
    return {
      claims,
      contributions: archive.stats.contributions,
      elapsedMs: archive.stats.generateMs,
      origin: request.scope ? `generator · ${request.scope.name}` : 'generator',
    };
  }

  /**
   * content has none to give: the generator produces sizes and encodings but never bytes,
   * since what it exists to exercise is topology and paint. A property of a generated
   * archive, not a capability still to be wired — a connection answers this.
   */
  async content(): Promise<Uint8Array> {
    throw new Error(
      'a generated archive holds no content bytes, only their sizes — read content from a ' +
        'connection to a real instance instead.',
    );
  }

  /** wholeArchive is this source's full archive, whatever a read of it is capped at. */
  private wholeArchive(): MockArchive {
    return memoised(this.params, this.params.claims);
  }
}

/**
 * scopedClaims is the mock's one answer to "what is in this scope", so a read and an id
 * listing cannot disagree. An unstamped claim is shared and belongs to every scope; a name
 * the branch table does not hold is no scope, and contains nothing.
 */
function scopedClaims(archive: MockArchive, scope?: string): DrawnClaim[] {
  if (!scope || scope === ARCHIVE_SCOPE) return archive.claims;
  if (!(scope in archive.branches)) return [];
  return archive.claims.filter((c) => c.branch === scope || c.branch === '');
}

/**
 * memoised reuses the last archive when the parameters match. The generator is deterministic,
 * so this changes no answer — it stops a branch listing regenerating 300k claims.
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

/**
 * RestSource talks to a real instance through the client generated from `openapi.yaml`, so
 * every route, path and response type it uses is the contract's rather than this file's.
 */
export class RestSource implements DataSource {
  readonly kind = 'rest' as const;

  private connection: Connection;
  private secret: string;
  private api: Api<unknown>;

  constructor(connection: Connection, secret: string) {
    this.connection = connection;
    this.secret = secret;
    this.api = apiFor(connection, secret);
  }

  describe(): string {
    return `${this.connection.baseUrl} · ${this.connection.authKind}`;
  }

  health(): Promise<ProbeResult> {
    return probe(this.connection, this.secret);
  }

  /**
   * branches reads the branch listing, plus the archive report for the branch-table head —
   * the only route reporting it, and so the only way the archive becomes a nameable scope.
   * If that read fails the archive is dropped rather than guessed.
   */
  async branches(): Promise<Scope[]> {
    const listed = await this.answer(() => this.api.branches.listBranches(), 'listing branches');
    const branches = (listed.data.branches ?? [])
      .filter((b) => Boolean(b.name && b.head))
      .map(({ name, head }) => ({ name, head }));

    try {
      const archive = await this.answer(
        () => this.api.archive.getArchiveInfo(),
        'reading the archive',
      );
      if (archive.data.head) return [{ name: ARCHIVE_SCOPE, head: archive.data.head }, ...branches];
    } catch {
      // Not grantable to this subject, or an older instance: the branches still stand.
    }
    return branches;
  }

  /**
   * scopeIds runs the scoped query: `select.branch` names the scope, `output.detail: id` asks
   * for identities. The answer is a sequence of bare ids — see idsFromSequence for why that
   * one read still frames its own records.
   */
  async scopeIds(scope: Scope): Promise<string[]> {
    const response = await this.query(
      {
        select: { branch: scope.name, head: scope.head },
        output: { detail: 'id', encoding: 'json' },
      },
      `reading the ids in ${scope.name}`,
    );
    return idsFromSequence(await response.text());
  }

  /**
   * fetch reads claims through the query route: `detail: claims` carries each with its edges,
   * `limit.results` caps it, and `form: original` is the id-defining form — what arrives is
   * what the id was computed over.
   */
  async fetch(request: FetchRequest): Promise<ClaimPage> {
    if (!request.scope) {
      throw new Error(
        'a server read needs a scope: pick a branch in the header first, since the query ' +
          'contract makes select.branch mandatory',
      );
    }
    const t0 = performance.now();
    const response = await this.query(
      {
        select: { branch: request.scope.name, head: request.scope.head },
        output: { detail: 'claims', form: 'original', encoding: 'json' },
        limit: { results: request.limit },
      },
      `reading ${request.scope.name}`,
    );
    // Read as it arrives rather than in one piece: 24,000 claims take long enough that a
    // reader deserves to watch the count climb, and the library's reader is a push parser,
    // so the decode is incremental anyway.
    const claims = await claimsFromBody(response, request.onProgress);
    return {
      claims,
      // A server read carries no contribution index; see contributionUnknown in core/claims.
      contributions: contributionUnknown,
      elapsedMs: performance.now() - t0,
      origin: `${request.scope.name} · query`,
    };
  }

  /**
   * content reads the claim's bytes from the route that serves them for its scope: this picks
   * which scope's route answers, and the client builds the path.
   */
  async content(scope: Scope | null, id: string): Promise<Uint8Array> {
    const branch = scope === null || scope.name === ARCHIVE_SCOPE ? null : scope.name;
    const response = await this.answer(
      () =>
        branch === null
          ? this.api.archive.getArchiveClaimContent(id)
          : this.api.branches.getBranchClaimContent(segment(branch), id),
      `reading the content of ${id.slice(0, 12)}…`,
    );
    return new Uint8Array(await response.arrayBuffer());
  }

  /** headOf reads a branch head — the one moving target in an archive. */
  async headOf(branch: string): Promise<string> {
    const reported = await this.answer(
      () => this.api.branches.getBranchHead(segment(branch)),
      `reading ${branch} head`,
    );
    // The route answers `{"head": "<id>"}`, so the id has to be read out of it — reading the
    // body as text returned the envelope.
    if (typeof reported.data.head !== 'string') {
      throw new Error(`no head in the answer for ${branch}`);
    }
    return reported.data.head;
  }

  /**
   * query posts a RankeQL query with the response format unset, which the route otherwise
   * declares as `json`: a result set is a JSON *sequence*, so parsing it as one document
   * would both fail and defeat the streaming. Unset, the client answers with the response
   * itself, whose body the readers below take apart record by record as it arrives.
   *
   * What goes on the wire is `query_codec`'s: `EncodeQuery` holds the query to the rules the
   * schema cannot state and drops what says nothing. Its text is handed back as an object
   * because the client formats its own bodies — a string body would be JSON-encoded twice.
   */
  private query(query: RankeQuery, what: string): Promise<Response> {
    const canonical = JSON.parse(EncodeQuery(query)) as ApiQuery;
    return this.answer(() => this.api.query.query(canonical, { format: undefined }), what);
  }

  /**
   * answer runs one generated call, naming the read in whatever goes wrong. The client throws
   * the response rather than returning it, so a failure is only a status until it is told what
   * was being read.
   */
  private async answer<T>(
    call: () => Promise<HttpResponse<T, ApiError>>,
    what: string,
  ): Promise<HttpResponse<T, ApiError>> {
    try {
      return await call();
    } catch (thrown) {
      throw await failedRead(thrown, what);
    }
  }
}

/**
 * segment escapes a value the client interpolates into a route path. A claim id is base32 by
 * the contract's pattern and needs none; a branch name is an unconstrained string, so a `/`
 * or a space in one would otherwise change which route is called.
 */
function segment(value: string): string {
  return encodeURIComponent(value);
}

/**
 * failedRead turns what a generated call threw into the error a reader sees. A route that asked
 * for a parsed body carries the contract's `Error` in `error`; one left unparsed still has its
 * body to read; and a request that never reached the server throws an `Error` whose message is
 * all there is to report.
 */
async function failedRead(thrown: unknown, what: string): Promise<Error> {
  if (thrown instanceof Error) return new Error(`${what} failed: ${thrown.message}`);
  if (typeof thrown !== 'object' || thrown === null) {
    return new Error(`${what} failed: ${String(thrown)}`);
  }
  const answer = thrown as HttpResponse<unknown, ApiError>;
  if (typeof answer.status !== 'number') return new Error(`${what} failed: ${String(thrown)}`);
  const detail = statedError(answer.error) || (await failureBody(answer));
  return new Error(`HTTP ${answer.status} ${what}${detail ? `: ${detail}` : ''}`);
}

/**
 * statedError reads the message out of the contract's error body: parsed, where the route asked
 * the client to parse one, or still text, where it did not.
 */
function statedError(body: unknown): string {
  if (typeof body === 'string') {
    try {
      return statedError(JSON.parse(body));
    } catch {
      return ''; // not the contract's shape, so the body itself is the best there is
    }
  }
  if (!body || typeof body !== 'object') return '';
  const { code, error } = body as Partial<ApiError>;
  return typeof error === 'string' ? error : typeof code === 'string' ? code : '';
}

/**
 * failureBody reads a failure body the client left unparsed, capped — it lands in a log line,
 * not a document.
 */
async function failureBody(response: Response): Promise<string> {
  let text = '';
  try {
    text = (await response.text()).trim();
  } catch {
    // A body already read, or none at all: the status still says what happened.
    return '';
  }
  return statedError(text) || text.slice(0, 400);
}

/**
 * idsFromSequence reads the ids out of an `output.detail: id` body, whose records are bare
 * JSON strings rather than claims.
 *
 * STOPGAP — `@flocko-motion/ranke` owns sequence framing, and its reader decodes claim
 * records only: an id record carries no `type` and no `created_at`, so `newSeqReader` refuses
 * it. Until the library reads an identity sequence, this repeats RFC 7464's framing, which is
 * the duplication the rest of this file exists to remove. Delete it the moment that lands.
 */
export function idsFromSequence(body: string): string[] {
  const ids: string[] = [];
  for (const record of body.split(RS)) {
    const text = record.trim();
    if (text === '') continue;
    let parsed: unknown;
    try {
      parsed = JSON.parse(text);
    } catch {
      continue; // a truncated record, or the execution report a query may append
    }
    if (typeof parsed === 'string') ids.push(parsed);
  }
  return ids;
}

/** The record separator RFC 7464 opens every record of a JSON sequence with. */
const RS = '\u001e';

/** encoder turns a body already in hand into the bytes the library's reader takes. */
const encoder = new TextEncoder();

/**
 * claimsFromBody decodes a result sequence with the library's reader, reporting as it goes:
 * how many claims are decoded, and how many bytes the reader has taken. A record split across
 * two chunks is the reader's business, not this loop's.
 */
export async function claimsFromBody(
  response: Response,
  onProgress?: (read: number, bytesRead: number) => void,
): Promise<DrawnClaim[]> {
  const reader = newSeqReader('json');
  const claims: DrawnClaim[] = [];
  const body = response.body;

  if (!body) {
    // No stream to pull: the body is already in hand, which is what a stubbed response gives.
    for (const claim of reader.push(encoder.encode(await response.text()))) {
      claims.push(drawn(claim));
    }
    for (const claim of reader.end()) claims.push(drawn(claim));
    onProgress?.(claims.length, reader.bytesRead);
    return claims;
  }

  const chunks = body.getReader();
  try {
    for (;;) {
      const { done, value } = await chunks.read();
      if (done) break;
      for (const claim of reader.push(value)) claims.push(drawn(claim));
      // Reported per chunk rather than per claim: the point is a moving number, not every one.
      onProgress?.(claims.length, reader.bytesRead);
    }
    for (const claim of reader.end()) claims.push(drawn(claim));
  } finally {
    chunks.releaseLock();
  }
  onProgress?.(claims.length, reader.bytesRead);
  return claims;
}

/** sourceFor builds the source a connection stands for. */
export function sourceFor(connection: Connection, secret: string): DataSource {
  return connection.kind === 'mock'
    ? new MockSource(connection.mock)
    : new RestSource(connection, secret);
}
