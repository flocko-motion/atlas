/**
 * Server connections — the explorer's address book. Headless.
 *
 * The explorer is a pure client in the Banana-Cake-Pop sense: it runs from a static
 * bundle, needs no server of its own, works with none at all (mock data), and can
 * hold several ranke-db instances at once, switching between them. Nothing here is
 * persisted server-side because there is no server side.
 *
 * The auth kinds mirror the server's own adapters and the security schemes in
 * `openapi/openapi.yaml`:
 *
 *   none      → the noauth adapter; no header sent
 *   apikey    → `X-API-Key: <key>`
 *   jwt       → `Authorization: Bearer <token>`
 *   macaroon  → `Authorization: Macaroon <token>`
 *
 * **Secrets are held in memory by default and are not written to disk.** Anything in
 * `localStorage` is readable by any script that gets into the page, so a token there
 * is a token exposed to the next XSS. `remember` opts in per connection, and only
 * then is the secret persisted.
 */

import { create } from 'zustand';

export type AuthKind = 'none' | 'apikey' | 'jwt' | 'macaroon';

export const AUTH_LABELS: Record<AuthKind, string> = {
  none: 'no auth',
  apikey: 'API key',
  jwt: 'JWT bearer',
  macaroon: 'macaroon',
};

/** How a secret is described in the UI, per kind. */
export const AUTH_SECRET_LABELS: Record<AuthKind, string | null> = {
  none: null,
  apikey: 'API key',
  jwt: 'bearer token',
  macaroon: 'macaroon',
};

/**
 * A mock connection's "server details": what archive the generator stands for.
 *
 * `seed` and `claimsPerContribution` define *which* archive this is; `claims` is how
 * large it can be — a ceiling, not a request. How much of it a read returns is the
 * query's `limit`, which is a separate thing and lives in the Query tab.
 */
export interface MockParams {
  /** Ceiling on the archive's size. */
  claims: number;
  claimsPerContribution: number;
  seed: number;
}

export interface Connection {
  id: string;
  name: string;
  /**
   * `mock` is a generator standing in for an instance, configured the same way and
   * read through the same port; `rest` is a real ranke-db.
   */
  kind: 'mock' | 'rest';
  /** Base URL of a ranke-db instance, e.g. `http://localhost:8080`. Unused when mock. */
  baseUrl: string;
  authKind: AuthKind;
  /** Persist the secret to localStorage. Off by default, deliberately. */
  remember: boolean;
  /** Generator parameters, meaningful when `kind` is `mock`. */
  mock: MockParams;
}

export const DEFAULT_MOCK: MockParams = { claims: 100000, claimsPerContribution: 10, seed: 0x5eed };

/** builtInMock is the connection the explorer starts with: a generator, no server. */
export function builtInMock(): Connection {
  return {
    id: 'mock-local',
    name: 'mock archive',
    kind: 'mock',
    baseUrl: '',
    authKind: 'none',
    remember: false,
    mock: { ...DEFAULT_MOCK },
  };
}

export type ProbeState = 'unknown' | 'probing' | 'ok' | 'failed';

export interface ProbeResult {
  state: ProbeState;
  /** Round-trip of the health request, in ms. */
  latencyMs?: number;
  detail?: string;
  /** Whatever `/health` reported, verbatim. */
  body?: string;
}

interface ConnectionsState {
  connections: Connection[];
  /** The connection in use. A mock generator counts as one. */
  activeId: string | null;
  probes: Record<string, ProbeResult>;

  addConnection: (c: Omit<Connection, 'id'>) => string;
  updateConnection: (id: string, patch: Partial<Connection>) => void;
  removeConnection: (id: string) => void;
  activate: (id: string | null) => void;
  setSecret: (id: string, secret: string) => void;
  secretOf: (id: string) => string;
  setProbe: (id: string, probe: ProbeResult) => void;
}

const STORE_KEY = 'ranke-explorer/connections';
const SECRET_KEY = 'ranke-explorer/secrets';

/** In-memory secrets, keyed by connection id. Never written unless `remember`. */
const secrets = new Map<string, string>();

interface Persisted {
  connections: Connection[];
  activeId: string | null;
}

function loadPersisted(): Persisted {
  try {
    const raw = localStorage.getItem(STORE_KEY);
    const parsed = raw ? (JSON.parse(raw) as Persisted) : null;
    const remembered = JSON.parse(localStorage.getItem(SECRET_KEY) ?? '{}') as Record<string, string>;
    for (const [id, secret] of Object.entries(remembered)) secrets.set(id, secret);
    if (parsed && Array.isArray(parsed.connections) && parsed.connections.length > 0) {
      // Older persisted entries predate `kind`/`mock`; fill them in rather than
      // discarding somebody's configured instances.
      const connections = parsed.connections.map((c) => ({
        ...c,
        kind: c.kind ?? 'rest',
        mock: c.mock ?? { ...DEFAULT_MOCK },
      }));
      return { connections, activeId: parsed.activeId ?? connections[0].id };
    }
  } catch {
    // A corrupt or unavailable store is not worth failing the app over.
  }
  const mock = builtInMock();
  return { connections: [mock], activeId: mock.id };
}

function persist(state: ConnectionsState): void {
  try {
    const payload: Persisted = { connections: state.connections, activeId: state.activeId };
    localStorage.setItem(STORE_KEY, JSON.stringify(payload));
    const remembered: Record<string, string> = {};
    for (const c of state.connections) {
      if (c.remember) {
        const secret = secrets.get(c.id);
        if (secret) remembered[c.id] = secret;
      }
    }
    localStorage.setItem(SECRET_KEY, JSON.stringify(remembered));
  } catch {
    // Private-mode browsers refuse writes; the session still works in memory.
  }
}

let counter = 0;

const initial = loadPersisted();

export const useConnections = create<ConnectionsState>((set, get) => ({
  connections: initial.connections,
  activeId: initial.activeId,
  probes: {},

  addConnection: (c) => {
    const id = `conn-${Date.now().toString(36)}-${++counter}`;
    set((s) => ({ connections: [...s.connections, { ...c, id }] }));
    persist(get());
    return id;
  },

  updateConnection: (id, patch) => {
    set((s) => ({
      connections: s.connections.map((c) => (c.id === id ? { ...c, ...patch } : c)),
    }));
    persist(get());
  },

  removeConnection: (id) => {
    secrets.delete(id);
    set((s) => ({
      connections: s.connections.filter((c) => c.id !== id),
      activeId: s.activeId === id ? null : s.activeId,
    }));
    persist(get());
  },

  activate: (activeId) => {
    set({ activeId });
    persist(get());
  },

  setSecret: (id, secret) => {
    secrets.set(id, secret);
    persist(get());
  },

  secretOf: (id) => secrets.get(id) ?? '',

  setProbe: (id, probe) => set((s) => ({ probes: { ...s.probes, [id]: probe } })),
}));

/** activeConnection returns the connection in use, falling back to the first. */
export function activeConnection(): Connection | null {
  const { connections, activeId } = useConnections.getState();
  return connections.find((c) => c.id === activeId) ?? connections[0] ?? null;
}

/** authHeaders builds the request headers a connection's auth kind calls for. */
export function authHeaders(connection: Connection, secret: string): Record<string, string> {
  switch (connection.authKind) {
    case 'apikey':
      return secret ? { 'X-API-Key': secret } : {};
    case 'jwt':
      return secret ? { Authorization: `Bearer ${secret}` } : {};
    case 'macaroon':
      return secret ? { Authorization: `Macaroon ${secret}` } : {};
    case 'none':
      return {};
  }
}

/** endpoint joins a connection's base URL with a path, tolerating trailing slashes. */
export function endpoint(connection: Connection, path: string): string {
  const base = connection.baseUrl.replace(/\/+$/, '');
  return `${base}${path.startsWith('/') ? path : `/${path}`}`;
}

/**
 * probe checks a connection by asking for `/health`, which every instance serves.
 * It is the only request the explorer makes before the user asks for data, and it
 * reports what came back rather than interpreting it.
 */
export async function probe(connection: Connection, secret: string): Promise<ProbeResult> {
  const t0 = performance.now();
  try {
    const response = await fetch(endpoint(connection, '/health'), {
      headers: authHeaders(connection, secret),
      mode: 'cors',
    });
    const latencyMs = performance.now() - t0;
    const body = (await response.text()).slice(0, 400);
    if (!response.ok) {
      return { state: 'failed', latencyMs, detail: `HTTP ${response.status}`, body };
    }
    return { state: 'ok', latencyMs, body };
  } catch (err) {
    // A CORS rejection and a dead host are indistinguishable from here; say so
    // rather than guessing, because the fix differs completely.
    return {
      state: 'failed',
      latencyMs: performance.now() - t0,
      detail: `unreachable, or blocked by CORS (${String(err)})`,
    };
  }
}
