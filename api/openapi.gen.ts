/* eslint-disable */
/* tslint:disable */
// @ts-nocheck
/*
 * ---------------------------------------------------------------
 * ## THIS FILE WAS GENERATED VIA SWAGGER-TYPESCRIPT-API        ##
 * ##                                                           ##
 * ## AUTHOR: acacode                                           ##
 * ## SOURCE: https://github.com/acacode/swagger-typescript-api ##
 * ---------------------------------------------------------------
 */

/** A tree-structured read — generate, filter, shape, bound. */
export interface Query {
  /**
   * A generator: a starting point and a traversal. `branch` roots at a branch's
   * current head and confines the query to it; `claim` optionally roots at a
   * claim id within the branch; `claim` **without** `branch` is a privileged
   * Universe read. Without `path`, the generator follows every edge outward to
   * the full closure.
   */
  select: Select;
  /**
   * A boolean tree of comparisons. `and`/`or`/`not` combine subtrees (`or` also
   * unions result sets); otherwise the object is a field→comparison map, each
   * field tested by one comparator.
   */
  where?: Where;
  /** Shapes each result and fixes the wire serialization. */
  output?: Output;
  /**
   * A named sort. Without it, results order by `(created_at, id)`; claims lacking
   * the named field sort last.
   */
  order?: Order;
  /** Bounds the read. */
  limit?: Limit;
  /**
   * Tunes and inspects how the query runs (paper 02 §Filtered Reads). Queries are
   * declarative; the planner lowers each to the most capable engine the storage
   * stack offers, so these controls affect execution and diagnostics only, never
   * the result set.
   */
  execution?: Execution;
}

/**
 * A generator: a starting point and a traversal. `branch` roots at a branch's
 * current head and confines the query to it; `claim` optionally roots at a
 * claim id within the branch; `claim` **without** `branch` is a privileged
 * Universe read. Without `path`, the generator follows every edge outward to
 * the full closure.
 */
export interface Select {
  /** Branch name to root at (its current head). */
  branch?: string;
  /** Claim id to root at; without `branch`, a privileged Universe read. */
  claim?: string;
  /** Traversal steps, applied in order. */
  path?: PathStep[];
}

/** One traversal step following typed edges to a bounded depth. */
export interface PathStep {
  /** Edge types to follow; a leading `-` on an entry excludes that type. */
  edges: string[];
  /**
   * provenance (outgoing, default), uses (incoming), connections (either).
   * @default "provenance"
   */
  dir?: "provenance" | "uses" | "connections";
  /**
   * Maximum hops for this step.
   * @min 1
   */
  depth?: number;
  /** Endpoint node types the step may land on; a leading `-` excludes a type. */
  nodes?: string[];
}

/**
 * A boolean tree of comparisons. `and`/`or`/`not` combine subtrees (`or` also
 * unions result sets); otherwise the object is a field→comparison map, each
 * field tested by one comparator.
 */
export type Where =
  | {
      and: Where[];
    }
  | {
      or: Where[];
    }
  | {
      /**
       * A boolean tree of comparisons. `and`/`or`/`not` combine subtrees (`or` also
       * unions result sets); otherwise the object is a field→comparison map, each
       * field tested by one comparator.
       */
      not: Where;
    }
  | Record<string, Comparison>;

/**
 * A test on one field. Exactly one operator is expected. `in` takes a set;
 * `glob` a shell-style wildcard; the rest a scalar.
 */
export interface Comparison {
  eq?: any;
  ne?: any;
  lt?: any;
  le?: any;
  gt?: any;
  ge?: any;
  in?: any[];
  glob?: string;
}

/** Shapes each result and fixes the wire serialization. */
export interface Output {
  /**
   * id (just the id), claim (the reached claim), path (the whole route).
   * @default "claim"
   */
  detail?: "id" | "claim" | "path";
  /**
   * Cap on inlined content bytes per claim. `false` (default) carries no
   * content; a byte size (e.g. `4096` or `"4kb"`) inlines up to that much.
   * @default false
   */
  content?: boolean | number | string;
  /**
   * How to handle content past the `content` cap: cutoff (truncate),
   * omit (drop the content), reference (return a hash stub in its place).
   */
  overflow?: "cutoff" | "omit" | "reference";
  /**
   * Wire serialization. json-seq (RFC 7464) is a convenience projection
   * (content base64-encoded, NOT id-verifiable); cbor-seq (RFC 8742) is the
   * canonical, verifiable form.
   * @default "json-seq"
   */
  encoding?: "json-seq" | "cbor-seq";
}

/**
 * Tunes and inspects how the query runs (paper 02 §Filtered Reads). Queries are
 * declarative; the planner lowers each to the most capable engine the storage
 * stack offers, so these controls affect execution and diagnostics only, never
 * the result set.
 */
export interface Execution {
  /**
   * Pin execution to a named storage layer — any layer the stack composes —
   * instead of letting the planner choose which layer answers. The engine then
   * follows from that layer's capability (its native walk, or a Cypher/GQL
   * engine if it has one).
   */
  layer?: string;
  /**
   * When true, the query returns detail on how it ran: the sequence's **final
   * item** is a QueryReport (identifiable by its shape, always last).
   * @default false
   */
  report?: boolean;
}

/**
 * A named sort. Without it, results order by `(created_at, id)`; claims lacking
 * the named field sort last.
 */
export interface Order {
  field: string;
  /** @default "asc" */
  dir?: "asc" | "desc";
}

/** Bounds the read. */
export interface Limit {
  /**
   * Maximum number of claims returned.
   * @min 1
   */
  results?: number;
  /** Wall-clock budget (duration, e.g. `"5s"`); the query is cancelled when exceeded. */
  time?: string;
}

/**
 * Diagnostic report emitted as the **final item** of a `POST /query` sequence
 * when `execution.report` is true. Identifiable by its shape — result items never
 * carry these fields — and always last.
 */
export interface QueryReport {
  /** The execution engine the query lowered to (e.g. cypher, native-walk). */
  engine?: string;
  /** The storage layer the query executed against. */
  layer?: string;
  /** The lowered query as executed (e.g. the Cypher text). */
  lowered?: string;
  /** Wall-clock execution time in milliseconds. */
  elapsedMs?: number;
  /** Number of result items emitted before this report. */
  results?: number;
  /** True if `limit.results` or `limit.time` cut the read short. */
  truncated?: boolean;
}

/** The outcome of a contribution — the new branch-table head and the appended claim ids. */
export interface ContributionResult {
  /** The new branch-table head id after the merge. */
  head: string;
  /** Content-addressed ids of the contributed claims, in order. */
  ids: string[];
}

export interface BranchHead {
  /** The content-addressed head claim id. */
  head: string;
}

export interface Health {
  /** @example "ok" */
  status: string;
  /**
   * The contributor identity this stack signs merges with.
   * @example "did:key:z6Mk..."
   */
  signer?: string;
}

export interface StorageLayer {
  name: string;
  /** Adapter type (e.g. memory, filesystem, s3, postgres, neo4j). */
  type: string;
}

export interface StorageLayerList {
  /** Read-through tiers, top (cache) to bottom (authoritative). */
  layers: StorageLayer[];
}

/**
 * Parameters for a verification run — the same shape whether declared in the
 * stack config (scheduled) or posted ad-hoc. Depths are those of
 * core-verification.
 */
export interface VerificationConfig {
  /** Root of the closure to verify — a branch name or a claim id. */
  closure: string;
  /**
   * Storage layer (by name) to read directly. Omit to read the composed
   * read-through view — which may mask the loss of an object on a deeper
   * layer, so name a layer to verify it without blind spots.
   */
  layer?: string;
  /**
   * completeness (a `has` sweep), record-correctness (recanonicalise and
   * recheck the id-chain and signatures), or full-content (also re-hash blobs).
   */
  depth?: "completeness" | "record-correctness" | "full-content";
  /**
   * For full-content, the max content size in bytes to re-read and re-hash
   * per claim; larger content is skipped this run. Omit to verify all content.
   */
  contentThreshold?: number;
}

/** A point-in-time record of a verification run; embeds the config that produced it. */
export interface VerificationReport {
  id: string;
  /**
   * Parameters for a verification run — the same shape whether declared in the
   * stack config (scheduled) or posted ad-hoc. Depths are those of
   * core-verification.
   */
  config: VerificationConfig;
  /**
   * The head id the run verified — the closure it pinned at start. When
   * `config.closure` names a branch, this is the branch's head at start time;
   * the report stays fixed to it even as the branch head moves on.
   */
  head: string;
  /**
   * running (in progress), complete (finished on its own), stopped (cancelled
   * by an operator via the cancel action — partial findings kept), or error
   * (the run itself failed).
   */
  status: "running" | "complete" | "stopped" | "error";
  /** @format date-time */
  startedAt: string;
  /** @format date-time */
  completedAt?: string;
  /** Claims checked so far; advances while `status` is `running`. */
  claimsChecked?: number;
  /** Content bytes re-read so far; advances while `status` is `running`. */
  bytesRead?: number;
  /** True when no failures were found in this run. */
  ok: boolean;
  /**
   * The failures found, accumulating while `status` is `running`. Empty (with
   * `ok: true`) for an intact closure; a healthy archive verifies with none.
   */
  failures?: VerificationFailure[];
}

export interface VerificationReportList {
  reports: VerificationReport[];
}

export interface VerificationFailure {
  /** The claim or object id that failed. */
  id: string;
  /**
   * corrupt-bytes — stored bytes don't match their hash (storage rot/loss;
   * self-heals via read-through if a deeper layer is intact).
   * invalid-content — the claim itself doesn't validate (e.g. bad signature);
   * unrepairable.
   */
  mode: "corrupt-bytes" | "invalid-content";
  /** The layer where the failure was observed. */
  layer: string;
  detail?: string;
}

export interface Error {
  /** Human-readable message. Carries no subject id, even on 403. */
  error: string;
}

export type QueryParamsType = Record<string | number, any>;
export type ResponseFormat = keyof Omit<Body, "body" | "bodyUsed">;

export interface FullRequestParams extends Omit<RequestInit, "body"> {
  /** set parameter to `true` for call `securityWorker` for this request */
  secure?: boolean;
  /** request path */
  path: string;
  /** content type of request body */
  type?: ContentType;
  /** query params */
  query?: QueryParamsType;
  /** format of response (i.e. response.json() -> format: "json") */
  format?: ResponseFormat;
  /** request body */
  body?: unknown;
  /** base url */
  baseUrl?: string;
  /** request cancellation token */
  cancelToken?: CancelToken;
}

export type RequestParams = Omit<
  FullRequestParams,
  "body" | "method" | "query" | "path"
>;

export interface ApiConfig<SecurityDataType = unknown> {
  baseUrl?: string;
  baseApiParams?: Omit<RequestParams, "baseUrl" | "cancelToken" | "signal">;
  securityWorker?: (
    securityData: SecurityDataType | null,
  ) => Promise<RequestParams | void> | RequestParams | void;
  customFetch?: typeof fetch;
}

export interface HttpResponse<D extends unknown, E extends unknown = unknown>
  extends Response {
  data: D;
  error: E;
}

type CancelToken = Symbol | string | number;

export enum ContentType {
  Json = "application/json",
  JsonApi = "application/vnd.api+json",
  FormData = "multipart/form-data",
  UrlEncoded = "application/x-www-form-urlencoded",
  Text = "text/plain",
}

export class HttpClient<SecurityDataType = unknown> {
  public baseUrl: string = "/";
  private securityData: SecurityDataType | null = null;
  private securityWorker?: ApiConfig<SecurityDataType>["securityWorker"];
  private abortControllers = new Map<CancelToken, AbortController>();
  private customFetch = (...fetchParams: Parameters<typeof fetch>) =>
    fetch(...fetchParams);

  private baseApiParams: RequestParams = {
    credentials: "same-origin",
    headers: {},
    redirect: "follow",
    referrerPolicy: "no-referrer",
  };

  constructor(apiConfig: ApiConfig<SecurityDataType> = {}) {
    Object.assign(this, apiConfig);
  }

  public setSecurityData = (data: SecurityDataType | null) => {
    this.securityData = data;
  };

  protected encodeQueryParam(key: string, value: any) {
    const encodedKey = encodeURIComponent(key);
    return `${encodedKey}=${encodeURIComponent(typeof value === "number" ? value : `${value}`)}`;
  }

  protected addQueryParam(query: QueryParamsType, key: string) {
    return this.encodeQueryParam(key, query[key]);
  }

  protected addArrayQueryParam(query: QueryParamsType, key: string) {
    const value = query[key];
    return value.map((v: any) => this.encodeQueryParam(key, v)).join("&");
  }

  protected toQueryString(rawQuery?: QueryParamsType): string {
    const query = rawQuery || {};
    const keys = Object.keys(query).filter(
      (key) => "undefined" !== typeof query[key],
    );
    return keys
      .map((key) =>
        Array.isArray(query[key])
          ? this.addArrayQueryParam(query, key)
          : this.addQueryParam(query, key),
      )
      .join("&");
  }

  protected addQueryParams(rawQuery?: QueryParamsType): string {
    const queryString = this.toQueryString(rawQuery);
    return queryString ? `?${queryString}` : "";
  }

  private contentFormatters: Record<ContentType, (input: any) => any> = {
    [ContentType.Json]: (input: any) =>
      input !== null && (typeof input === "object" || typeof input === "string")
        ? JSON.stringify(input)
        : input,
    [ContentType.JsonApi]: (input: any) =>
      input !== null && (typeof input === "object" || typeof input === "string")
        ? JSON.stringify(input)
        : input,
    [ContentType.Text]: (input: any) =>
      input !== null && typeof input !== "string"
        ? JSON.stringify(input)
        : input,
    [ContentType.FormData]: (input: any) => {
      if (input instanceof FormData) {
        return input;
      }

      return Object.keys(input || {}).reduce((formData, key) => {
        const property = input[key];
        formData.append(
          key,
          property instanceof Blob
            ? property
            : typeof property === "object" && property !== null
              ? JSON.stringify(property)
              : `${property}`,
        );
        return formData;
      }, new FormData());
    },
    [ContentType.UrlEncoded]: (input: any) => this.toQueryString(input),
  };

  protected mergeRequestParams(
    params1: RequestParams,
    params2?: RequestParams,
  ): RequestParams {
    return {
      ...this.baseApiParams,
      ...params1,
      ...(params2 || {}),
      headers: {
        ...(this.baseApiParams.headers || {}),
        ...(params1.headers || {}),
        ...((params2 && params2.headers) || {}),
      },
    };
  }

  protected createAbortSignal = (
    cancelToken: CancelToken,
  ): AbortSignal | undefined => {
    if (this.abortControllers.has(cancelToken)) {
      const abortController = this.abortControllers.get(cancelToken);
      if (abortController) {
        return abortController.signal;
      }
      return void 0;
    }

    const abortController = new AbortController();
    this.abortControllers.set(cancelToken, abortController);
    return abortController.signal;
  };

  public abortRequest = (cancelToken: CancelToken) => {
    const abortController = this.abortControllers.get(cancelToken);

    if (abortController) {
      abortController.abort();
      this.abortControllers.delete(cancelToken);
    }
  };

  public request = async <T = any, E = any>({
    body,
    secure,
    path,
    type,
    query,
    format,
    baseUrl,
    cancelToken,
    ...params
  }: FullRequestParams): Promise<HttpResponse<T, E>> => {
    const secureParams =
      ((typeof secure === "boolean" ? secure : this.baseApiParams.secure) &&
        this.securityWorker &&
        (await this.securityWorker(this.securityData))) ||
      {};
    const requestParams = this.mergeRequestParams(params, secureParams);
    const queryString = query && this.toQueryString(query);
    const payloadFormatter = this.contentFormatters[type || ContentType.Json];
    const responseFormat = format || requestParams.format;

    return this.customFetch(
      `${baseUrl || this.baseUrl || ""}${path}${queryString ? `?${queryString}` : ""}`,
      {
        ...requestParams,
        headers: {
          ...(requestParams.headers || {}),
          ...(type && type !== ContentType.FormData
            ? { "Content-Type": type }
            : {}),
        },
        signal:
          (cancelToken
            ? this.createAbortSignal(cancelToken)
            : requestParams.signal) || null,
        body:
          typeof body === "undefined" || body === null
            ? null
            : payloadFormatter(body),
      },
    ).then(async (response) => {
      const r = response as HttpResponse<T, E>;
      r.data = null as unknown as T;
      r.error = null as unknown as E;

      const responseToParse = responseFormat ? response.clone() : response;
      const data = !responseFormat
        ? r
        : await responseToParse[responseFormat]()
            .then((data) => {
              if (r.ok) {
                r.data = data;
              } else {
                r.error = data;
              }
              return r;
            })
            .catch((e) => {
              r.error = e;
              return r;
            });

      if (cancelToken) {
        this.abortControllers.delete(cancelToken);
      }

      if (!response.ok) throw data;
      return data;
    });
  };
}

/**
 * @title ranke-db API
 * @version 0.2.0
 * @baseUrl /
 *
 * REST/HTTP binding for a single **ranke-db** stack: read the verifiable graph
 * with a declarative query, and contribute signed claims to it.
 *
 * A server hosts exactly **one stack**, launched from a config file — there is
 * no tenant or archive routing in the paths.
 *
 * ## Reads are queries
 *
 * The full read surface is `POST /query`: a tree-structured query (`select`
 * generators, `where` filters, `output` shaping, `limit` bounds) as a JSON
 * object. Cypher/GQL is **never** a client route — it is an internal execution
 * engine the planner lowers a query to. A **cacheable GET subset** covers the by-id
 * reads without a query body: `GET /{branch}/head`, `GET /{branch}/claim/{id}`,
 * `GET /{branch}/content/{hash}`, and the privileged `GET /$universe/claim/{id}` /
 * `GET /$universe/content/{hash}`. Content also rides **inline** in query results
 * via `output.content`; the content route fetches a single blob by hash — including
 * the blob an `output.overflow: reference` stub names.
 *
 * ## Encodings and verifiability
 *
 * A query returns a **streaming sequence** of results, one item each, in the
 * serialization named by `output.encoding`:
 *
 *   - `json-seq` (RFC 7464, `application/json-seq`) — the default; JSON records,
 *     content base64-encoded when inlined. A **convenience projection**: easy to
 *     read and debug, but **not** independently verifiable.
 *   - `cbor-seq` (RFC 8742, `application/cbor-seq`) — the **canonical** form: each
 *     item is the claim's signed CBOR, re-hashable and signature-checkable against
 *     its id. Use this when verifiability matters.
 *
 * Because a claim's id signs its canonical CBOR, only `cbor-seq` round-trips a
 * claim verifiably; the `json-seq` encoding is a rendering for convenience.
 *
 * A by-id claim GET returns the claim as its signed CBOR (`application/cbor`); a
 * content GET streams the blob as raw bytes (`application/octet-stream`). In query
 * results content is instead carried inline (base64 in the JSON encodings, raw CBOR
 * byte strings in `cbor-seq`), capped by `output.content`.
 *
 * ## Credentials and authorization
 *
 * The contract **binds no per-operation rights** and defines **no** read/write/admin
 * ladder — authorization is entirely core-access (system accounts with CRUDA rights
 * over branch globs), decided on the subject the credential resolves to. Every route
 * accepts the same set of credentials; the shape of a route never varies by right.
 *
 * What the contract *does* enumerate is how a credential is **presented**, because
 * the endpoint routes deterministically on the presented scheme to the auth adapter
 * of that type — no parsing a token to guess its kind:
 *
 *   - `Authorization: Bearer <jwt>` → the **JWT** adapter
 *   - `X-API-Key: <key>` → the **API-key** adapter
 *   - `Authorization: Macaroon <token>` → the **macaroon** adapter
 *   - **no** credential header → the **noauth** adapter
 *
 * Which of these adapters is actually mounted is the stack's config; a presentation
 * with no configured adapter is `401`. (Only if two mechanisms were forced onto one
 * scheme would the endpoint fall back to trying each — the schemes above are
 * distinct, so routing stays deterministic.) A `403` body never discloses the
 * subject id.
 *
 * ## Closure and the 404 rule
 *
 * Every branch read is bounded by that branch's closure. A claim or content that
 * exists in the Universe but lies **outside** the named branch's closure returns
 * `404` — indistinguishable from one that does not exist. Head-id reads that
 * bypass the branch table are privileged and reached only through the reserved
 * `$universe` name (`$` is illegal in ordinary branch names).
 */
export class Api<
  SecurityDataType extends unknown,
> extends HttpClient<SecurityDataType> {
  health = {
    /**
     * @description Reports that the stack is up and the signing/contributor identity it attests merges under. Whether this requires a privileged subject is a core-access decision, not part of this contract.
     *
     * @tags system
     * @name Health
     * @summary Liveness and signer identity
     * @request GET:/health
     * @secure
     */
    health: (params: RequestParams = {}) =>
      this.request<Health, any>({
        path: `/health`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),
  };
  query = {
    /**
     * @description Runs a `select`/`where`/`output`/`limit` query tree and streams the result set, one item per result, in the serialization chosen by `output.encoding` (default `json-seq`). Results are ordered by `(created_at, id)` unless a named `order` overrides it; to page, carry the last result's order key into the next request's `where`. When `execution.report` is `true`, the **final item** in the sequence is a `QueryReport` (see the schema) — identifiable by its shape and always last — carrying the execution engine/layer, the lowered query, timing, and whether a limit truncated the read. `execution.layer` pins which storage layer runs it. The response media type mirrors `output.encoding`: `application/json-seq` or `application/cbor-seq`. A branch named in `select` that does not exist is `404`; claims outside the branch closure simply do not appear. A branch-less `select.claim` is a privileged Universe read.
     *
     * @tags read
     * @name Query
     * @summary Read the graph with a declarative query
     * @request POST:/query
     * @secure
     */
    query: (data: Query, params: RequestParams = {}) =>
      this.request<File, Error>({
        path: `/query`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),
  };
  contribute = {
    /**
     * @description Contributes one or more signed claims to a branch as a single **atomic** merge — all are absorbed under one new branch-table head, or none are. The request body is the claims as **signed CBOR** in ranke's bundle format (a subgraph; claims reference each other and their closure by content-addressed id). Content-addressed and therefore **idempotent**: re-contributing yields the same ids with no duplicates. The target branch is **required** and given as the `branch` query parameter — a contribution is always a set of claims merged onto one named branch, and core checks the **C** right on it before the sequencer merges. It is not in the body: the body carries only the claims being added, and the sequencer itself creates the `contribution/branches` branch-table claim that records them (paper 02 §Sequencer), so the target cannot be derived from the submitted bundle. On success the new branch-table head id and the contributed claim ids are returned.
     *
     * @tags write
     * @name Contribute
     * @summary Contribute signed claims (atomic)
     * @request POST:/contribute
     * @secure
     */
    contribute: (
      query: {
        /** The target branch the claims are merged onto (the `C` right applies here). */
        branch: string;
      },
      data: File,
      params: RequestParams = {},
    ) =>
      this.request<ContributionResult, Error>({
        path: `/contribute`,
        method: "POST",
        query: query,
        body: data,
        secure: true,
        format: "json",
        ...params,
      }),
  };
  branch = {
    /**
     * @description Returns the branch's current head id — a moving target (it advances on every contribution). Cacheable **with revalidation** (weak `ETag`, `Cache-Control: no-cache`): a conditional request is cheap when the head has not moved. To inspect the head claim itself, fetch it via `GET /{branch}/claim/{id}`.
     *
     * @tags read
     * @name GetBranchHead
     * @summary Current head id of a branch
     * @request GET:/{branch}/head
     * @secure
     */
    getBranchHead: (branch: string, params: RequestParams = {}) =>
      this.request<BranchHead, Error>({
        path: `/${branch}/head`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Returns claim `{id}` as its signed CBOR bytes, **only if it lies in branch `{name}`'s closure**. A claim that is superseded, contradicted, or otherwise outside the closure returns `404`, indistinguishable from one that does not exist. Immutably **cacheable** by id (strong `ETag`, `Cache-Control: public, immutable`): the id content-addresses the bytes, so they never change.
     *
     * @tags read
     * @name GetBranchClaim
     * @summary Fetch a claim within a branch's closure
     * @request GET:/{branch}/claim/{id}
     * @secure
     */
    getBranchClaim: (branch: string, id: string, params: RequestParams = {}) =>
      this.request<File, Error>({
        path: `/${branch}/claim/${id}`,
        method: "GET",
        secure: true,
        ...params,
      }),

    /**
     * @description Streams the content blob addressed by `{hash}` — **only if a claim in branch `{name}`'s closure references it** (same closure guarantee as claims; out-of-closure or unknown → `404`). The bytes verify against `{hash}` and size as they stream. Immutably **cacheable** by hash. This is how a client pulls a specific blob: the "source content" step of a read, and the way to retrieve a blob an `output.overflow: reference` stub named rather than inlined.
     *
     * @tags read
     * @name GetBranchContent
     * @summary Fetch content bytes within a branch's closure
     * @request GET:/{branch}/content/{hash}
     * @secure
     */
    getBranchContent: (
      branch: string,
      hash: string,
      params: RequestParams = {},
    ) =>
      this.request<Blob, Error>({
        path: `/${branch}/content/${hash}`,
        method: "GET",
        secure: true,
        ...params,
      }),
  };
  universe = {
    /**
     * @description Returns claim `{id}` as its signed CBOR bytes directly from the Universe, bypassing any branch table. This is a **privileged** head-id read, conferred only through the reserved `$universe` name (see core-access). Immutably cacheable by id.
     *
     * @tags read
     * @name GetUniverseClaim
     * @summary Fetch a claim by id from the Universe (privileged)
     * @request GET:/$universe/claim/{id}
     * @secure
     */
    getUniverseClaim: (id: string, params: RequestParams = {}) =>
      this.request<File, Error>({
        path: `/$universe/claim/${id}`,
        method: "GET",
        secure: true,
        ...params,
      }),

    /**
     * @description Streams the content blob addressed by `{hash}` directly from the Universe, bypassing any branch table — a **privileged** read conferred only through `$universe`. The bytes verify against `{hash}` and size as they stream. Immutably cacheable by hash.
     *
     * @tags read
     * @name GetUniverseContent
     * @summary Fetch content bytes by hash from the Universe (privileged)
     * @request GET:/$universe/content/{hash}
     * @secure
     */
    getUniverseContent: (hash: string, params: RequestParams = {}) =>
      this.request<Blob, Error>({
        path: `/$universe/content/${hash}`,
        method: "GET",
        secure: true,
        ...params,
      }),
  };
  system = {
    /**
     * @description Lists the stack's storage layers (read-through tiers) by **name and type only** — never connection details or secrets. Naming layers is what lets a verification run target one directly (a read-through view can mask the loss of an object on a deeper layer).
     *
     * @tags system
     * @name ListStorageLayers
     * @summary List storage layers
     * @request GET:/system/layers
     * @secure
     */
    listStorageLayers: (params: RequestParams = {}) =>
      this.request<StorageLayerList, Error>({
        path: `/system/layers`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Verification runs across the whole stack, newest first. Each report is a **point-in-time record**: a layer repaired externally shows clean in a later run, so reports accumulate rather than overwrite — until explicitly removed with `DELETE /system/verification/{reportId}`. The API lives under `/system/` (rather than the paper's root `/verification…`) for two reasons: a root-level `/verification/{id}` would collide with `/{branch}/head` under net/http (both match `/verification/head`, neither more specific), and a run is a stack-wide operational resource — it roots at a closure named in the request body, not in the path.
     *
     * @tags system
     * @name ListVerifications
     * @summary List verification runs
     * @request GET:/system/verification
     * @secure
     */
    listVerifications: (params: RequestParams = {}) =>
      this.request<VerificationReportList, Error>({
        path: `/system/verification`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Starts a run from a `VerificationConfig`: walk the closure rooted at `closure` (a branch name — resolved to its current head and **pinned** for the life of the run — or a head id directly), reading the named `layer` directly, re-checking each claim to the configured depth. It returns *findings*, not contents, so it never leaks across closures. A run may take a **very long time** (hours to days over a large closure at full-content depth), so it is always **asynchronous**: the call returns `202` immediately with the running report and a `Location` header pointing at the report resource. Poll it with `GET /system/verification/{reportId}`, pacing by the `Retry-After` hint; stop it with `DELETE`. Starting the same run twice yields two independent point-in-time reports (runs are not deduplicated). Verification is resource-heavy, so the stack caps the number of runs that may execute **concurrently** (configured; default 1). When that cap is already reached the call returns `429` — the server never stops a run to make room. To proceed, either wait (per `Retry-After`) or `GET /system/verification` to see what is running and free a slot deliberately: `cancel` a run to stop it while keeping its report, or `DELETE` it to remove it entirely.
     *
     * @tags system
     * @name StartVerification
     * @summary Start a verification run
     * @request POST:/system/verification
     * @secure
     */
    startVerification: (data: VerificationConfig, params: RequestParams = {}) =>
      this.request<VerificationReport, Error>({
        path: `/system/verification`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description The report for a run. Poll until `status` leaves `running`; while it is still running the response carries a `Retry-After` hint and the progress counters (`claimsChecked`, `bytesRead`) advance between polls.
     *
     * @tags system
     * @name GetVerification
     * @summary Show a verification run
     * @request GET:/system/verification/{reportId}
     * @secure
     */
    getVerification: (reportId: string, params: RequestParams = {}) =>
      this.request<VerificationReport, Error>({
        path: `/system/verification/${reportId}`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Really deletes the run and its report. If the run is still `running` it is stopped first, then the record is removed — so this both aborts a run and cleans up finished history in one step. Unlike the cancel action, nothing survives: a subsequent `GET` is `404`. Either verb frees a concurrency slot; cancel keeps the report, delete removes it.
     *
     * @tags system
     * @name DeleteVerification
     * @summary Delete a verification run
     * @request DELETE:/system/verification/{reportId}
     * @secure
     */
    deleteVerification: (reportId: string, params: RequestParams = {}) =>
      this.request<void, Error>({
        path: `/system/verification/${reportId}`,
        method: "DELETE",
        secure: true,
        ...params,
      }),

    /**
     * @description Stops a `running` run but **keeps** its report: the record stays in history with `status` `stopped` and whatever partial findings it had gathered, and the concurrency slot is freed. Use this to abort a run you want to keep a record of; use `DELETE` to remove it entirely. Idempotent — cancelling a run that has already finished or stopped returns its current report unchanged.
     *
     * @tags system
     * @name CancelVerification
     * @summary Cancel a running verification run
     * @request POST:/system/verification/{reportId}/cancel
     * @secure
     */
    cancelVerification: (reportId: string, params: RequestParams = {}) =>
      this.request<VerificationReport, Error>({
        path: `/system/verification/${reportId}/cancel`,
        method: "POST",
        secure: true,
        format: "json",
        ...params,
      }),
  };
}
