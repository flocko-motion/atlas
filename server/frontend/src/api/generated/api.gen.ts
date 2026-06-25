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

export interface AdmitUserReq {
  subject: string;
  tenant: string;
}

export interface AdmitUserResp {
  subject: string;
  tenant: string;
}

export interface ArchiveReadReq {
  ra: string;
  tenant: string;
}

export interface BranchReq {
  name: string;
  ra: string;
  tenant: string;
}

export interface BranchResp {
  contributor: string;
  head: string;
  history: number;
  name: string;
  time: string;
}

export interface BranchSummary {
  head: string;
  name: string;
}

export interface ClaimReq {
  id: string;
  ra: string;
  tenant: string;
}

export interface ClaimView {
  canonical: string;
  content_hash: string;
  contributor: string;
  created_at: string;
  edges: EdgeView[];
  encoding: string;
  id: string;
  size: number;
  type: string;
}

export interface CreateArchiveReq {
  ra: string;
  sequencer: StackSequencer;
  storage: StackStorage;
  tenant: string;
  title: string;
}

export interface DeleteArchiveReq {
  ra: string;
  tenant: string;
}

export interface DeleteArchiveResp {
  deleted: boolean;
}

export interface EdgeView {
  direction: string;
  reference: string;
  type: string;
}

export interface GetArchiveReq {
  ra: string;
  tenant: string;
}

export interface GetArchiveResp {
  current: string;
  ra: string;
  target: string;
  tenant: string;
  title: string;
}

export interface GrantRoleReq {
  ra: string;
  role: string;
  subject: string;
  tenant: string;
}

export interface GrantRoleResp {
  ra: string;
  role: string;
  subject: string;
}

export interface ListBranchesResp {
  branches: BranchSummary[];
}

export type ListSubjectsReq = object;

export interface ListSubjectsResp {
  subjects: SubjectView[];
}

export interface ListTenantUsersReq {
  tenant: string;
}

export interface ListTenantUsersResp {
  users: TenantUser[];
}

export interface PatchArchiveReq {
  ra: string;
  target: string;
  tenant: string;
}

export interface RoleEntry {
  ra: string;
  role: string;
}

export interface SetUserDisabledReq {
  disabled: boolean;
  subject: string;
}

export interface StackSequencer {
  backend: string;
  dsn: string;
  key: string;
  path: string;
}

export interface StackStorage {
  backend: string;
  dir: string;
  dsn: string;
}

export interface SubjectView {
  disabled: boolean;
  id: string;
}

export interface TenantUser {
  roles: RoleEntry[];
  subject: string;
}

export interface VerificationResp {
  error: string;
  valid: boolean;
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
     * @description lifecycle state. Works in any state (a stopped/failed archive still reports); hidden (404) from a subject with no visibility into the tenant.
     *
     * @name GetApiArchivesTenantRa
     * @summary Returns an archive's status: its title and current + target
     * @request GET:/api/archives/{tenant}/{ra}
     * @secure
     */
    getApiArchivesTenantRa: (
      tenant: string,
      ra: string,
      params: RequestParams = {},
    ) =>
      this.request<GetArchiveResp, void>({
        path: `/api/archives/${tenant}/${ra}`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description set read-only) — declarative: it writes the target and core reconciles toward it. Gated by ra.control. Returns the archive's updated status.
     *
     * @name PatchApiArchivesTenantRa
     * @summary Sets an archive's target lifecycle state (start / stop /
     * @request PATCH:/api/archives/{tenant}/{ra}
     * @secure
     */
    patchApiArchivesTenantRa: (
      tenant: string,
      ra: string,
      data: PatchArchiveReq,
      params: RequestParams = {},
    ) =>
      this.request<GetArchiveResp, void>({
        path: `/api/archives/${tenant}/${ra}`,
        method: "PATCH",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiArchivesTenantRaBranches
     * @summary Lists an archive's branches with their head ids.
     * @request GET:/api/archives/{tenant}/{ra}/branches
     * @secure
     */
    getApiArchivesTenantRaBranches: (
      tenant: string,
      ra: string,
      params: RequestParams = {},
    ) =>
      this.request<ListBranchesResp, void>({
        path: `/api/archives/${tenant}/${ra}/branches`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiArchivesTenantRaBranchesName
     * @summary Returns one branch: its head, binding time, contributor, and history depth.
     * @request GET:/api/archives/{tenant}/{ra}/branches/{name}
     * @secure
     */
    getApiArchivesTenantRaBranchesName: (
      tenant: string,
      ra: string,
      name: string,
      params: RequestParams = {},
    ) =>
      this.request<BranchResp, void>({
        path: `/api/archives/${tenant}/${ra}/branches/${name}`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Verification failure is a result (200, valid=false), not an HTTP error; a missing branch is 404.
     *
     * @name GetApiArchivesTenantRaBranchesNameVerification
     * @summary Runs the §5.10 verification across a branch's provenance.
     * @request GET:/api/archives/{tenant}/{ra}/branches/{name}/verification
     * @secure
     */
    getApiArchivesTenantRaBranchesNameVerification: (
      tenant: string,
      ra: string,
      name: string,
      params: RequestParams = {},
    ) =>
      this.request<VerificationResp, void>({
        path: `/api/archives/${tenant}/${ra}/branches/${name}/verification`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiArchivesTenantRaClaimsId
     * @summary Returns a claim: a readable projection plus its canonical bytes.
     * @request GET:/api/archives/{tenant}/{ra}/claims/{id}
     * @secure
     */
    getApiArchivesTenantRaClaimsId: (
      tenant: string,
      ra: string,
      id: string,
      params: RequestParams = {},
    ) =>
      this.request<ClaimView, void>({
        path: `/api/archives/${tenant}/${ra}/claims/${id}`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description in its stack speaks Cypher. HandleRaw is used so the no-capability case can return a precise 501 (schemaf has no 501 sentinel). Stub: always 501 after the access/lifecycle gate, until ranke-go ships the capability.
     *
     * @name PostApiArchivesTenantRaGql
     * @summary Runs a read-only Cypher/GQL query against an archive — if a layer
     * @request POST:/api/archives/{tenant}/{ra}/gql
     * @secure
     */
    postApiArchivesTenantRaGql: (
      tenant: string,
      ra: string,
      params: RequestParams = {},
    ) =>
      this.request<void, void>({
        path: `/api/archives/${tenant}/${ra}/gql`,
        method: "POST",
        secure: true,
        ...params,
      }),

    /**
     * @description brings it up. The stack (storage + sequencer backends) is chosen at runtime by the caller — this is what lets a test suite drive any backend. Gated by tenant-admin.
     *
     * @name PostApiTenantsTenantArchives
     * @summary Defines a new archive and its persistence stack, then
     * @request POST:/api/tenants/{tenant}/archives
     * @secure
     */
    postApiTenantsTenantArchives: (
      tenant: string,
      data: CreateArchiveReq,
      params: RequestParams = {},
    ) =>
      this.request<GetArchiveResp, void>({
        path: `/api/tenants/${tenant}/archives`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name DeleteApiTenantsTenantArchivesRa
     * @summary Stops an archive and removes its definition. Gated by tenant-admin.
     * @request DELETE:/api/tenants/{tenant}/archives/{ra}
     * @secure
     */
    deleteApiTenantsTenantArchivesRa: (
      tenant: string,
      ra: string,
      params: RequestParams = {},
    ) =>
      this.request<DeleteArchiveResp, void>({
        path: `/api/tenants/${tenant}/archives/${ra}`,
        method: "DELETE",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description tenant-admin; scoped — only this tenant's grants, never other affiliations.
     *
     * @name GetApiTenantsTenantUsers
     * @summary Lists the tenant's users and their roles. Gated by
     * @request GET:/api/tenants/{tenant}/users
     * @secure
     */
    getApiTenantsTenantUsers: (tenant: string, params: RequestParams = {}) =>
      this.request<ListTenantUsersResp, void>({
        path: `/api/tenants/${tenant}/users`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description ("enter as valid"). Gated by tenant-admin.
     *
     * @name PostApiTenantsTenantUsers
     * @summary Admits a subject into a tenant — the baseline membership
     * @request POST:/api/tenants/{tenant}/users
     * @secure
     */
    postApiTenantsTenantUsers: (
      tenant: string,
      data: AdmitUserReq,
      params: RequestParams = {},
    ) =>
      this.request<AdmitUserResp, void>({
        path: `/api/tenants/${tenant}/users`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description intact (grants are additive). Gated by tenant-admin.
     *
     * @name DeleteApiTenantsTenantUsersSubjectGrants
     * @summary Revokes one role from a tenant user, leaving other roles
     * @request DELETE:/api/tenants/{tenant}/users/{subject}/grants
     * @secure
     */
    deleteApiTenantsTenantUsersSubjectGrants: (
      tenant: string,
      subject: string,
      params: RequestParams = {},
    ) =>
      this.request<GrantRoleResp, void>({
        path: `/api/tenants/${tenant}/users/${subject}/grants`,
        method: "DELETE",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description empty, else an RA role). Gated by tenant-admin.
     *
     * @name PostApiTenantsTenantUsersSubjectGrants
     * @summary Grants a role to a tenant user (tenant role when ra is
     * @request POST:/api/tenants/{tenant}/users/{subject}/grants
     * @secure
     */
    postApiTenantsTenantUsersSubjectGrants: (
      tenant: string,
      subject: string,
      data: GrantRoleReq,
      params: RequestParams = {},
    ) =>
      this.request<GrantRoleResp, void>({
        path: `/api/tenants/${tenant}/users/${subject}/grants`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiUsers
     * @summary Lists every known subject and its disabled state. Root only.
     * @request GET:/api/users
     * @secure
     */
    getApiUsers: (params: RequestParams = {}) =>
      this.request<ListSubjectsResp, void>({
        path: `/api/users`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name PatchApiUsersSubject
     * @summary Enables or disables a subject globally. Root only.
     * @request PATCH:/api/users/{subject}
     * @secure
     */
    patchApiUsersSubject: (
      subject: string,
      data: SetUserDisabledReq,
      params: RequestParams = {},
    ) =>
      this.request<SubjectView, void>({
        path: `/api/users/${subject}`,
        method: "PATCH",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),
  };
}
