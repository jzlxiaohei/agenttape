// Pure data access. No React, no hooks. The only layer that talks HTTP.

async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(path, { headers: { Accept: "application/json" } });
  if (!res.ok) {
    throw await httpError(res);
  }
  return (await res.json()) as T;
}

async function getText(path: string): Promise<string> {
  const res = await fetch(path);
  if (!res.ok) {
    throw await httpError(res);
  }
  return res.text();
}

async function postJSON<T>(path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method: "POST",
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (!res.ok) {
    throw await httpError(res);
  }
  return (await res.json()) as T;
}

async function patchJSON(path: string, body?: unknown): Promise<void> {
  const res = await fetch(path, {
    method: "PATCH",
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (!res.ok) {
    throw await httpError(res);
  }
}

async function del(path: string, body?: unknown): Promise<void> {
  const res = await fetch(path, {
    method: "DELETE",
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (!res.ok) {
    throw await httpError(res);
  }
}

async function httpError(res: Response): Promise<Error> {
  const status = `${res.status} ${res.statusText}`.trim();
  const detail = await responseErrorDetail(res);
  return new Error(detail ? `${status}: ${detail}` : status);
}

async function responseErrorDetail(res: Response): Promise<string> {
  const text = (await res.text().catch(() => "")).trim();
  if (!text) return "";
  if (!res.headers.get("content-type")?.includes("application/json")) {
    return text;
  }
  try {
    return jsonErrorDetail(JSON.parse(text)) || text;
  } catch {
    return text;
  }
}

function jsonErrorDetail(value: unknown): string {
  if (!value || typeof value !== "object") return "";
  const obj = value as Record<string, unknown>;
  for (const key of ["error", "message", "detail"]) {
    const v = obj[key];
    if (typeof v === "string" && v.trim()) return v.trim();
  }
  return "";
}

export const api = { getJSON, getText, postJSON, patchJSON, del };
