// Pure data access. No React, no hooks. The only layer that talks HTTP.

async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(path, { headers: { Accept: "application/json" } });
  if (!res.ok) {
    throw new Error(`${res.status} ${res.statusText}`);
  }
  return (await res.json()) as T;
}

async function getText(path: string): Promise<string> {
  const res = await fetch(path);
  if (!res.ok) {
    throw new Error(`${res.status} ${res.statusText}`);
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
    // surface the server's plain-text error message (e.g. the 409 credentials note)
    throw new Error((await res.text().catch(() => "")) || `${res.status} ${res.statusText}`);
  }
  return (await res.json()) as T;
}

export const api = { getJSON, getText, postJSON };
