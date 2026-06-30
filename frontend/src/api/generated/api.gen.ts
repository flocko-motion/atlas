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

export interface Health {
  /** @example "ok" */
  status: string;
  /**
   * The contributor identity this stack signs with.
   * @example "did:key:z6Mk..."
   */
  signer?: string;
}

export interface BranchSummary {
  name: string;
  /** The content-addressed head claim id. */
  head: string;
}

export interface BranchList {
  branches: BranchSummary[];
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
 * Parameters for a verification run — the same shape whether declared in
 * the stack config (scheduled/automatic) or posted ad-hoc.
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
   * Max content size, in bytes, to re-read and re-hash per claim. Larger
   * content is skipped this run (the claim's signature is still checked).
   * Omit to verify all content regardless of size.
   */
  contentThreshold?: number;
}

/** A point-in-time record of a verification run; embeds the config that produced it. */
export interface VerificationReport {
  id: string;
  /**
   * Parameters for a verification run — the same shape whether declared in
   * the stack config (scheduled/automatic) or posted ad-hoc.
   */
  config: VerificationConfig;
  status: "running" | "complete" | "stopped" | "error";
  /** @format date-time */
  startedAt: string;
  /** @format date-time */
  completedAt?: string;
  claimsChecked?: number;
  bytesRead?: number;
  /** True when no failures were found in this run. */
  ok: boolean;
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
   * invalid-content — the claim itself doesn't validate (e.g. bad
   * signature); unrepairable.
   */
  mode: "corrupt-bytes" | "invalid-content";
  /** The layer where the failure was observed. */
  layer: string;
  detail?: string;
}

/** The result of a contribution transaction — the ids (handles) of the appended claims. */
export interface ContributionResult {
  /** Content-addressed ids of the contributed claims, in order. */
  ids: string[];
}

export interface GqlQuery {
  /** A read-only Cypher query. */
  query: string;
  /** Optional named query parameters. */
  parameters?: Record<string, any>;
}

export interface GqlResult {
  columns: string[];
  /** Row-major result; each row aligns with columns. */
  rows: any[][];
}

export interface Error {
  /** Human-readable message. */
  error: string;
  /** On 403, the subject id to grant in config. */
  subject?: string;
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
 * @version 0.1.0
 * @baseUrl /
 *
 * REST API for a single **ranke-db** stack: contribute signed claims and read
 * the verifiable graph they form.
 *
 * A server hosts exactly **one stack**, launched from a config file — there is
 * no tenant or archive routing in the paths.
 *
 * **Authentication is a pluggable adapter** (e.g. NoAuth, JWT, API key — and
 * extensible), chosen by the stack's config; this contract therefore pins no
 * auth scheme. Send whatever credential the configured adapter expects — the
 * endpoints, the capability model, and the `401`/`403` outcomes are identical
 * under any auth mode. Authorization is capability-based with a simple lattice
 * **read ⊂ write ⊂ admin**:
 *
 *   - **read**  — list branches, fetch claims within a closure, Cypher reads
 *   - **write** — contribute claims
 *   - **admin** — verification runs, storage-layer introspection, other privileged ops
 *
 * Claims are **signed CBOR** objects, content-addressed and referenced by their
 * id (the handle). The API carries claims as their raw CBOR bytes
 * (`application/cbor`) in both directions — there is **no JSON projection** of a
 * claim. A client *contributes* one claim (PUT) or a set (POST, an atomic
 * transaction) by sending the CBOR; a read returns the claim's CBOR, which is
 * the canonical signed form — independently verifiable, never reconstructed.
 *
 * **Every read is bounded by a branch closure.** All access to claims and
 * content goes through a branch name; you can only fetch what lies in that
 * branch's closure. There is no branch-free access by raw id — that is an
 * internal system operation and is never exposed. A claim outside the named
 * branch's closure is `404`, indistinguishable from one that does not exist.
 */
export class Api<
  SecurityDataType extends unknown,
> extends HttpClient<SecurityDataType> {
  health = {
    /**
     * No description
     *
     * @tags system
     * @name Health
     * @summary Liveness and readiness of the stack
     * @request GET:/health
     */
    health: (params: RequestParams = {}) =>
      this.request<Health, any>({
        path: `/health`,
        method: "GET",
        format: "json",
        ...params,
      }),
  };
  branches = {
    /**
     * @description Lists the branch names that exist, each with its current head id — a point-in-time snapshot (the head moves on every contribution). This is the entry point for discovery: a head is just a claim, so to inspect it fetch it via `GET /branches/{name}/claims/{id}`. Requires `read`.
     *
     * @tags read
     * @name ListBranches
     * @summary List branch names with their current head
     * @request GET:/branches
     */
    listBranches: (params: RequestParams = {}) =>
      this.request<BranchList, Error>({
        path: `/branches`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * @description Contribute a **set** of claims to branch `{name}` as a single **atomic transaction** — all are appended, or none are. The request body is the claims as **signed CBOR** in ranke's bundle format (a subgraph; claims reference each other by their content-addressed ids). Content-addressed, so it is **idempotent**: re-contributing yields the same ids with no duplicates. Returns the ids (handles) of the contributed claims. For a single claim, **PUT it by id** instead. Requires `write`.
     *
     * @tags write
     * @name Contribute
     * @summary Contribute a set of claims (bulk)
     * @request POST:/branches/{name}/claims
     */
    contribute: (name: string, data: File, params: RequestParams = {}) =>
      this.request<ContributionResult, Error>({
        path: `/branches/${name}/claims`,
        method: "POST",
        body: data,
        format: "json",
        ...params,
      }),

    /**
     * @description Returns claim `{id}` as its **signed CBOR** bytes, **only if it lies in branch `{name}`'s closure** — the fundamental access guarantee. A claim that is superseded, contradicted, or otherwise outside the closure returns `404`, indistinguishable from one that does not exist. Raw by-id access (unbounded by a branch) is an internal system operation, never exposed. Requires `read`.
     *
     * @tags read
     * @name GetClaim
     * @summary Fetch a claim within a branch's closure
     * @request GET:/branches/{name}/claims/{id}
     */
    getClaim: (name: string, id: string, params: RequestParams = {}) =>
      this.request<File, Error>({
        path: `/branches/${name}/claims/${id}`,
        method: "GET",
        ...params,
      }),

    /**
     * @description Contribute a single claim to branch `{name}`, idempotently. The request body is the claim's **signed CBOR** bytes; its content-addressed id must equal `{id}` (otherwise `400`). Re-putting the same claim is a no-op. For more than one claim, POST the set to the collection as a transaction. Requires `write`.
     *
     * @tags write
     * @name PutClaim
     * @summary Contribute a single claim
     * @request PUT:/branches/{name}/claims/{id}
     */
    putClaim: (
      name: string,
      id: string,
      data: File,
      params: RequestParams = {},
    ) =>
      this.request<void, Error>({
        path: `/branches/${name}/claims/${id}`,
        method: "PUT",
        body: data,
        ...params,
      }),

    /**
     * @description Streams the content blob addressed by `{hash}` — **only if a claim in branch `{name}`'s closure references it** (same closure guarantee as claims; out-of-closure or unknown → `404`). The bytes verify against `{hash}` and size as they stream. This is the "source content" step of a read: walk claims by id, then pull the bytes you need. Requires `read`.
     *
     * @tags read
     * @name GetContent
     * @summary Fetch content bytes within a branch's closure
     * @request GET:/branches/{name}/contents/{hash}
     */
    getContent: (name: string, hash: string, params: RequestParams = {}) =>
      this.request<Blob, Error>({
        path: `/branches/${name}/contents/${hash}`,
        method: "GET",
        ...params,
      }),

    /**
     * @description Run a read-only Cypher query against branch `{name}`'s closure. This is an **optional feature**, available only when the stack is configured with a graph-native storage layer (e.g. neo4j) — otherwise it returns `501`. It is not the primary read path (see the `read` endpoints). Mutation is never expressed in Cypher (claims are created only via contribute), and a query observes the **exact same filter rules** as native reads (no superseded, contradicted, or out-of-closure claims). Requires `read`.
     *
     * @tags read
     * @name Gql
     * @summary Cypher query over a branch (optional)
     * @request POST:/branches/{name}/gql
     */
    gql: (name: string, data: GqlQuery, params: RequestParams = {}) =>
      this.request<GqlResult, Error>({
        path: `/branches/${name}/gql`,
        method: "POST",
        body: data,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),
  };
  storage = {
    /**
     * @description Lists the stack's storage layers (read-through tiers) by **name and type only** — never connection details or secrets. Naming layers is what lets a verification run target one directly (a read-through view can mask the loss of an object on a deeper layer). Requires `admin`.
     *
     * @tags system
     * @name ListStorageLayers
     * @summary List storage layers
     * @request GET:/storage/layers
     */
    listStorageLayers: (params: RequestParams = {}) =>
      this.request<StorageLayerList, Error>({
        path: `/storage/layers`,
        method: "GET",
        format: "json",
        ...params,
      }),
  };
  verification = {
    /**
     * @description Verification runs, newest first. Each report is a **point-in-time record**: a layer repaired externally (outage recovered, files restored) shows clean in a later run, so reports accumulate rather than overwrite. Requires `admin`.
     *
     * @tags verify
     * @name ListVerifications
     * @summary List verification runs
     * @request GET:/verification
     */
    listVerifications: (params: RequestParams = {}) =>
      this.request<VerificationReportList, Error>({
        path: `/verification`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * @description Start a run from a `VerificationConfig`: walk the closure rooted at `closure`, reading the named `layer` directly, re-checking each claim's signature and re-hashing its content up to `contentThreshold` bytes (larger content is skipped this run). Async — returns the running report immediately. May root at any branch or id: it returns *findings*, not contents, so it never leaks across closures. Requires `admin`.
     *
     * @tags verify
     * @name StartVerification
     * @summary Start a verification run
     * @request POST:/verification
     */
    startVerification: (data: VerificationConfig, params: RequestParams = {}) =>
      this.request<VerificationReport, Error>({
        path: `/verification`,
        method: "POST",
        body: data,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description The report for a run (poll until `status` is `complete`). Requires `admin`.
     *
     * @tags verify
     * @name GetVerification
     * @summary Show a verification run
     * @request GET:/verification/{reportId}
     */
    getVerification: (reportId: string, params: RequestParams = {}) =>
      this.request<VerificationReport, Error>({
        path: `/verification/${reportId}`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * @description Stop a running verification; returns the report with `status` `stopped`. Requires `admin`.
     *
     * @tags verify
     * @name StopVerification
     * @summary Stop a verification run
     * @request DELETE:/verification/{reportId}
     */
    stopVerification: (reportId: string, params: RequestParams = {}) =>
      this.request<VerificationReport, Error>({
        path: `/verification/${reportId}`,
        method: "DELETE",
        format: "json",
        ...params,
      }),
  };
}
