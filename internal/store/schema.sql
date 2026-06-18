-- tracelab module 2 schema (version 1).
-- Design: a shared "spine" (events) for the unified session timeline, with
-- type-specific detail tables (http_exchanges / hook_events) so the two very
-- different data sources evolve independently without NULL-soup. Raw bytes live
-- on the filesystem (kept as a user-facing feature); the DB only points to them.

CREATE TABLE IF NOT EXISTS schema_meta (
  key   TEXT PRIMARY KEY,
  value TEXT
);

CREATE TABLE IF NOT EXISTS sessions (
  id         TEXT PRIMARY KEY,
  client     TEXT,
  upstream   TEXT,
  label      TEXT,
  started_at TEXT,
  ended_at   TEXT,
  meta_json  TEXT
);

-- Spine: the minimal set every event shares (timeline + correlation + source).
CREATE TABLE IF NOT EXISTS events (
  id             TEXT PRIMARY KEY,
  session_id     TEXT REFERENCES sessions(id),
  kind           TEXT,                 -- http_exchange | hook_event
  source_adapter TEXT,
  source_mode    TEXT,
  client         TEXT,
  turn_id        TEXT,
  parent_id      TEXT,
  request_id     TEXT,
  started_at     TEXT,
  completed_at   TEXT,
  duration_ms    INTEGER,
  created_at      TEXT
);

-- HTTP-specific detail (1:1 with events where kind=http_exchange).
CREATE TABLE IF NOT EXISTS http_exchanges (
  event_id        TEXT PRIMARY KEY REFERENCES events(id),
  method          TEXT,
  url             TEXT,
  target          TEXT,
  response_status INTEGER,
  is_completion   INTEGER,             -- 1=completion call  0=control/probe
  provider        TEXT,
  model           TEXT,
  wire_api        TEXT,
  normalize_error TEXT,
  input_tokens    INTEGER,
  output_tokens   INTEGER,
  total_tokens    INTEGER,
  cached_tokens   INTEGER,
  reasoning_tokens INTEGER,
  normalized_json TEXT                  -- block.Raw stripped (full original is on disk)
);

-- Hook-specific detail (1:1 with events where kind=hook_event).
CREATE TABLE IF NOT EXISTS hook_events (
  event_id   TEXT PRIMARY KEY REFERENCES events(id),
  runtime    TEXT,
  event_name TEXT,
  tool_call_id TEXT
);

-- Pointers to raw bytes on disk (filesystem storage, kept as a feature).
CREATE TABLE IF NOT EXISTS raw_files (
  id               TEXT PRIMARY KEY,
  event_id         TEXT REFERENCES events(id),
  role             TEXT,               -- request_body | response_body | hook_payload
  path             TEXT,               -- relative to data dir; user can open directly
  media_type       TEXT,
  content_encoding TEXT,
  body_encoding    TEXT,
  size_bytes       INTEGER,
  redaction_applied INTEGER
);

-- Tags: structural facts, suspected heuristics, and (later) manual labels.
CREATE TABLE IF NOT EXISTS tags (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  event_id   TEXT REFERENCES events(id),
  tag        TEXT,
  confidence REAL,
  suspected  INTEGER,                  -- 1 = shown as 疑似
  source     TEXT,                     -- structural | heuristic | manual
  evidence   TEXT,
  created_at TEXT
);

-- Per-section token stats (http only) for queries like "tools > 50% of tokens".
CREATE TABLE IF NOT EXISTS sections (
  event_id      TEXT REFERENCES events(id),
  name          TEXT,
  bytes         INTEGER,
  approx_tokens INTEGER
);

-- Full-text search over the meaningful prompt sections (next.md 3.2).
CREATE VIRTUAL TABLE IF NOT EXISTS events_fts USING fts5(
  event_id UNINDEXED,
  system_text,
  messages_text,
  final_text,
  tool_args_text,
  tokenize='unicode61'
);

CREATE INDEX IF NOT EXISTS idx_events_session ON events(session_id, started_at);
CREATE INDEX IF NOT EXISTS idx_http_provider  ON http_exchanges(provider, model);
CREATE INDEX IF NOT EXISTS idx_tags_tag        ON tags(tag);
CREATE INDEX IF NOT EXISTS idx_tags_event      ON tags(event_id);
