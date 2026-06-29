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

export interface Branch {
  name: string;
  head: string;
  /** Number of head revisions recorded for this branch. */
  history: number;
  /** Contributor of the head claim. */
  contributor: string;
  /**
   * Time the head was recorded.
   * @format date-time
   */
  time: string;
}

export interface Verification {
  branch: string;
  head: string;
  /** True if every checked claim verified. */
  ok: boolean;
  /** Number of claims verified. */
  checked: number;
  /** Ids of claims that failed verification (empty when ok). */
  failures?: string[];
}

export interface NodeInput {
  /**
   * The node type, application-defined (e.g. person, fruit).
   * @example "person"
   */
  type: string;
  /**
   * The content encoding.
   * @example "application/json"
   */
  encoding: string;
  /**
   * The node's content, base64-encoded.
   * @format byte
   */
  content: Blob;
}

export interface EdgeInput {
  /** The id of an existing node this edge points at. */
  reference: string;
  /**
   * The relation type.
   * @example "likes"
   */
  type: string;
  /**
   * Relation direction from this node's perspective.
   * @default "outgoing"
   */
  direction?: "outgoing" | "incoming";
}

/** The content of a claim to contribute. The server adds the contributor and signature. */
export interface Contribution {
  node: NodeInput;
  edges?: EdgeInput[];
}

export interface EdgeView {
  reference: string;
  type: string;
  direction: "outgoing" | "incoming";
}

export interface Claim {
  /** Content-addressed claim id. */
  id: string;
  /** Content-addressed node id. */
  node: string;
  type: string;
  encoding: string;
  content_hash: string;
  /** @format date-time */
  created_at: string;
  contributor: string;
  edges: EdgeView[];
  /**
   * The canonical signed bytes (base64); verify the claim independently from these.
   * @format byte
   */
  canonical: Blob;
}

export interface Query {
  /** A read-only Cypher query. */
  query: string;
  /** Optional named query parameters. */
  parameters?: Record<string, any>;
}

export interface QueryResult {
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
 * no tenant or archive routing in the paths. Authentication is configurable
 * (none / JWT / API key); authorization is capability-based with a simple
 * lattice **read ⊂ write ⊂ admin**:
 *
 *   - **read**  — query branches, claims, run verification, Cypher reads
 *   - **write** — contribute claims
 *   - **admin** — privileged operations (reserved)
 *
 * Claims are **content-addressed and signed**. A client *contributes* the
 * content of a claim (a node plus its edges); the server signs it with the
 * stack's configured signer (the contributor identity) and appends it. Every
 * claim is returned with its embedded **canonical signed bytes**, so any
 * response is independently verifiable — never merely reconstructable.
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
     * @description Returns every branch with its current head. Requires `read`.
     *
     * @tags read
     * @name ListBranches
     * @summary List branches
     * @request GET:/branches
     * @secure
     */
    listBranches: (params: RequestParams = {}) =>
      this.request<BranchList, Error>({
        path: `/branches`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description The branch's head, contributor, time, and revision count. Requires `read`.
     *
     * @tags read
     * @name GetBranch
     * @summary Get one branch
     * @request GET:/branches/{name}
     * @secure
     */
    getBranch: (name: string, params: RequestParams = {}) =>
      this.request<Branch, Error>({
        path: `/branches/${name}`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Runs full verification over the branch's claims (signatures, content hashes, and closure) and reports the result. Requires `read`.
     *
     * @tags read
     * @name VerifyBranch
     * @summary Verify a branch
     * @request GET:/branches/{name}/verification
     * @secure
     */
    verifyBranch: (name: string, params: RequestParams = {}) =>
      this.request<Verification, Error>({
        path: `/branches/${name}/verification`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Contribute one claim to the branch: a node (its type, encoding, and content) plus any edges to existing nodes. The server signs the claim with the stack's signer and appends it, advancing the branch head. Returns the resulting signed claim. Requires `write`. Edges reference nodes by **id**, which must already exist. To build a relation like "alice likes apple", contribute the target node first, capture its id from the response, then contribute the source node with an edge to it. (Batch / atomic-subgraph contribution is a deliberate open question — see the discussion notes.)
     *
     * @tags write
     * @name Contribute
     * @summary Contribute a claim
     * @request POST:/branches/{name}/claims
     * @secure
     */
    contribute: (
      name: string,
      data: Contribution,
      params: RequestParams = {},
    ) =>
      this.request<Claim, Error>({
        path: `/branches/${name}/claims`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),
  };
  claims = {
    /**
     * @description Returns the claim, including its embedded canonical signed bytes. Requires `read`.
     *
     * @tags read
     * @name GetClaim
     * @summary Get a claim by id
     * @request GET:/claims/{id}
     * @secure
     */
    getClaim: (id: string, params: RequestParams = {}) =>
      this.request<Claim, Error>({
        path: `/claims/${id}`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),
  };
  gql = {
    /**
     * @description Run a read-only Cypher query against the graph. Mutation is never expressed in Cypher — claims are only created via `contribute` — and a query observes the **exact same filter rules** as native reads (no superseded, contradicted, or out-of-closure claims). Requires `read`. Returns `501` if the configured storage backend provides no Cypher surface (only graph-native backends such as neo4j do).
     *
     * @tags read
     * @name Query
     * @summary Cypher query (read-only)
     * @request POST:/gql
     * @secure
     */
    query: (data: Query, params: RequestParams = {}) =>
      this.request<QueryResult, Error>({
        path: `/gql`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),
  };
}
