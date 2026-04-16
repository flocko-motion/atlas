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

export interface CreateRunReq {
  worker_config_id: string;
}

export interface CreateRunResp {
  run_id: string;
}

export interface EdgeProvenanceResp {
  edge: EdgeResponse;
  run_id: string;
  worker_config_node: NodeResponse;
}

export interface EdgeResponse {
  confidence: number;
  created_at: string;
  id: string;
  run_id: string;
  source_node_id: string;
  target_node_id: string;
  type: string;
}

export interface GetEdgeProvenanceReq {
  id: string;
}

export interface GetEdgeReq {
  id: string;
}

export interface GetEntityReq {
  id: string;
  minConfidence: number;
}

export interface GetEntityResp {
  entity: NodeResponse;
  relations: RelationResponse[];
}

export interface GetEntityTimelineReq {
  id: string;
}

export interface GetEntityTimelineResp {
  relations: RelationResponse[];
}

export interface GetNodeEdgesReq {
  direction: string;
  id: string;
  limit: number;
  offset: number;
  type: string;
}

export interface GetNodeEdgesResp {
  edges: EdgeResponse[];
}

export interface GetNodeProvenanceReq {
  id: string;
}

export interface GetNodeReq {
  id: string;
}

export interface GetQueueReq {
  byClass: string;
  byConfig: string;
  byWorker: string;
  contentClass: string;
  contentType: string;
  limit: number;
}

export interface GetQueueResp {
  nodes: NodeResponse[];
}

export interface ListEdgesReq {
  limit: number;
  offset: number;
  type: string;
}

export interface ListEdgesResp {
  edges: EdgeResponse[];
}

export interface ListNodesReq {
  contentClass: string;
  contentType: string;
  level: number;
  limit: number;
  offset: number;
}

export interface ListNodesResp {
  nodes: NodeResponse[];
}

export interface ListRelationsReq {
  limit: number;
  offset: number;
  type: string;
  unresolved: boolean;
}

export interface ListRelationsResp {
  relations: RelationResponse[];
}

export interface NodeResponse {
  artifact_created_at: string;
  artifact_created_at_blur: string;
  confidence: number;
  content: string;
  content_class: string;
  content_len: number;
  content_sha256: string;
  content_type: string;
  created_at: string;
  encoding_class: string;
  encoding_format: string;
  id: string;
  level: number;
  origin: string;
  original_name: string;
  valid_from: string;
  valid_from_blur: string;
  valid_until: string;
  valid_until_blur: string;
}

export interface ProvenanceSubgraph {
  edges: EdgeResponse[];
  nodes: NodeResponse[];
}

export interface RelationResponse {
  head_edges: EdgeResponse[];
  node: NodeResponse;
  tail_edges: EdgeResponse[];
}

export interface SearchEntitiesReq {
  limit: number;
  offset: number;
  q: string;
  type: string;
}

export interface SearchEntitiesResp {
  entities: NodeResponse[];
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
  public baseUrl: string = "";
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
 * @title API
 * @version 1.0.0
 */
export class Api<
  SecurityDataType extends unknown,
> extends HttpClient<SecurityDataType> {
  api = {
    /**
     * No description
     *
     * @name GetApiEdges
     * @summary Returns edges with optional type filter.
     * @request GET:/api/edges
     */
    getApiEdges: (params: RequestParams = {}) =>
      this.request<ListEdgesResp, void>({
        path: `/api/edges`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiEdgesId
     * @summary Returns an edge by ID with its metadata.
     * @request GET:/api/edges/{id}
     */
    getApiEdgesId: (id: string, params: RequestParams = {}) =>
      this.request<EdgeResponse, void>({
        path: `/api/edges/${id}`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiEdgesIdProvenance
     * @summary Returns the provenance context of an edge: run, worker config node.
     * @request GET:/api/edges/{id}/provenance
     */
    getApiEdgesIdProvenance: (id: string, params: RequestParams = {}) =>
      this.request<EdgeProvenanceResp, void>({
        path: `/api/edges/${id}/provenance`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiEntities
     * @summary Searches entities by full-text query over names and aliases.
     * @request GET:/api/entities
     */
    getApiEntities: (params: RequestParams = {}) =>
      this.request<SearchEntitiesResp, void>({
        path: `/api/entities`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiEntitiesId
     * @summary Returns an entity with all its relations, sorted by temporal validity.
     * @request GET:/api/entities/{id}
     */
    getApiEntitiesId: (id: string, params: RequestParams = {}) =>
      this.request<GetEntityResp, void>({
        path: `/api/entities/${id}`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiEntitiesIdTimeline
     * @summary Returns all relations of an entity sorted chronologically by valid_from.
     * @request GET:/api/entities/{id}/timeline
     */
    getApiEntitiesIdTimeline: (id: string, params: RequestParams = {}) =>
      this.request<GetEntityTimelineResp, void>({
        path: `/api/entities/${id}/timeline`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiNodes
     * @summary Returns a filtered, paginated list of nodes.
     * @request GET:/api/nodes
     */
    getApiNodes: (params: RequestParams = {}) =>
      this.request<ListNodesResp, void>({
        path: `/api/nodes`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * @description L0 sources use multipart form upload; L1/L2 use JSON. L0 root creation is idempotent via content_sha256.
     *
     * @name PostApiNodes
     * @summary Creates a graph node.
     * @request POST:/api/nodes
     */
    postApiNodes: (params: RequestParams = {}) =>
      this.request<void, void>({
        path: `/api/nodes`,
        method: "POST",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiNodesId
     * @summary Returns a node by ID with full metadata and inline content.
     * @request GET:/api/nodes/{id}
     */
    getApiNodesId: (id: string, params: RequestParams = {}) =>
      this.request<NodeResponse, void>({
        path: `/api/nodes/${id}`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * @description Serves from Postgres cache if available, otherwise proxies from S3.
     *
     * @name GetApiNodesIdContent
     * @summary Returns the raw content bytes of a node.
     * @request GET:/api/nodes/{id}/content
     */
    getApiNodesIdContent: (id: string, params: RequestParams = {}) =>
      this.request<void, void>({
        path: `/api/nodes/${id}/content`,
        method: "GET",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiNodesIdEdges
     * @summary Returns edges connected to a node, filterable by direction and type.
     * @request GET:/api/nodes/{id}/edges
     */
    getApiNodesIdEdges: (id: string, params: RequestParams = {}) =>
      this.request<GetNodeEdgesResp, void>({
        path: `/api/nodes/${id}/edges`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiNodesIdProvenance
     * @summary Returns the upstream provenance chain from a node back to L0 roots.
     * @request GET:/api/nodes/{id}/provenance
     */
    getApiNodesIdProvenance: (id: string, params: RequestParams = {}) =>
      this.request<ProvenanceSubgraph, void>({
        path: `/api/nodes/${id}/provenance`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiQueue
     * @summary Returns unprocessed nodes for a given worker type.
     * @request GET:/api/queue
     */
    getApiQueue: (params: RequestParams = {}) =>
      this.request<GetQueueResp, void>({
        path: `/api/queue`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiRelations
     * @summary Returns filtered relation nodes.
     * @request GET:/api/relations
     */
    getApiRelations: (params: RequestParams = {}) =>
      this.request<ListRelationsResp, void>({
        path: `/api/relations`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name PostApiRuns
     * @summary Registers a new worker run and returns a run_id.
     * @request POST:/api/runs
     */
    postApiRuns: (data: CreateRunReq, params: RequestParams = {}) =>
      this.request<CreateRunResp, void>({
        path: `/api/runs`,
        method: "POST",
        body: data,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),
  };
}
