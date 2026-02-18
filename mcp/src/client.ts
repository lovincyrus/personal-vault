import { readFileSync } from "fs";
import { join } from "path";
import { homedir } from "os";

interface VaultStatus {
  initialized: boolean;
  locked: boolean;
  field_count: number;
  categories: Record<string, number>;
}

interface FieldInfo {
  id: string;
  category: string;
  field_name: string;
  value?: string;
  sensitivity: string;
  updated_at: string;
  version: number;
}

interface ContextBundle {
  categories: Record<string, FieldInfo[]>;
}

class VaultError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "VaultError";
  }
}

function vaultDir(): string {
  return process.env.VAULT_DIR ?? join(homedir(), ".pvault");
}

function serverAddr(): string {
  return process.env.VAULT_ADDR ?? "http://127.0.0.1:7200";
}

let serviceToken: string | null = null;

function resolveToken(): string | null {
  // 1. Auto-provisioned service token (set by provisionServiceToken)
  if (serviceToken) {
    return serviceToken;
  }

  // 2. VAULT_TOKEN env var (pre-configured service tokens)
  if (process.env.VAULT_TOKEN) {
    return process.env.VAULT_TOKEN;
  }

  // 3. Session file
  try {
    const sessionPath = join(vaultDir(), ".session");
    return readFileSync(sessionPath, "utf-8").trim();
  } catch {
    return null;
  }
}

/**
 * Provision a scoped service token using the session token.
 * Called once at startup when VAULT_SCOPE is set.
 */
export async function provisionServiceToken(
  consumer: string,
  scope: string,
): Promise<void> {
  const sessionPath = join(vaultDir(), ".session");
  let sessionTok: string;
  try {
    sessionTok = readFileSync(sessionPath, "utf-8").trim();
  } catch {
    throw new VaultError("vault: no session — run 'pvault unlock'");
  }

  const url = `${serverAddr()}/vault/tokens/service`;
  let resp: Response;
  try {
    resp = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${sessionTok}`,
      },
      body: JSON.stringify({ consumer, scope, ttl: "24h" }),
    });
  } catch {
    throw new VaultError("vault: server not running — run 'pvault unlock'");
  }

  if (!resp.ok) {
    const text = await resp.text();
    throw new VaultError(`vault: failed to provision service token — ${text}`);
  }

  const result = (await resp.json()) as { token: string };
  serviceToken = result.token;
}

function mapError(status: number, body: string): VaultError {
  if (status === 401) {
    return new VaultError("vault: session expired — run 'pvault unlock'");
  }
  if (status === 403) {
    return new VaultError("vault: vault is locked — run 'pvault unlock'");
  }
  if (status === 404) {
    return new VaultError(`vault: not found — ${body}`);
  }

  // Try to extract error message from JSON response
  try {
    const parsed = JSON.parse(body);
    if (parsed.error) {
      return new VaultError(`vault: ${parsed.error}`);
    }
  } catch {
    // not JSON
  }

  return new VaultError(`vault: HTTP ${status} — ${body}`);
}

async function apiRequest(
  method: string,
  path: string,
  body?: unknown
): Promise<unknown> {
  const token = resolveToken();
  const url = `${serverAddr()}${path}`;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  let resp: Response;
  try {
    resp = await fetch(url, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });
  } catch (err) {
    throw new VaultError(
      "vault: server not running — run 'pvault unlock'"
    );
  }

  if (!resp.ok) {
    const text = await resp.text();
    throw mapError(resp.status, text);
  }

  return resp.json();
}

export async function getStatus(): Promise<VaultStatus> {
  // Public endpoint — works without auth
  const url = `${serverAddr()}/vault/status`;
  let resp: Response;
  try {
    resp = await fetch(url);
  } catch {
    throw new VaultError(
      "vault: server not running — run 'pvault unlock'"
    );
  }
  if (!resp.ok) {
    throw new VaultError(`vault: status check failed — HTTP ${resp.status}`);
  }
  return resp.json() as Promise<VaultStatus>;
}

export async function getSchema(): Promise<unknown> {
  // Public endpoint — works without auth
  const url = `${serverAddr()}/vault/schema`;
  let resp: Response;
  try {
    resp = await fetch(url);
  } catch {
    throw new VaultError(
      "vault: server not running — run 'pvault unlock'"
    );
  }
  if (!resp.ok) {
    throw new VaultError(`vault: schema fetch failed — HTTP ${resp.status}`);
  }
  return resp.json();
}

export async function getField(id: string): Promise<FieldInfo> {
  return apiRequest("GET", `/vault/fields/${id}`) as Promise<FieldInfo>;
}

export async function listFields(): Promise<FieldInfo[]> {
  return apiRequest("GET", "/vault/fields") as Promise<FieldInfo[]>;
}

export async function getContext(): Promise<ContextBundle> {
  return apiRequest("GET", "/vault/context") as Promise<ContextBundle>;
}

export async function setField(
  id: string,
  value: string,
  sensitivity?: string,
): Promise<{ status: string }> {
  const body: Record<string, string> = { value };
  if (sensitivity) {
    body.sensitivity = sensitivity;
  }
  return apiRequest("PUT", `/vault/fields/${id}`, body) as Promise<{
    status: string;
  }>;
}

export type { VaultStatus, FieldInfo, ContextBundle };
