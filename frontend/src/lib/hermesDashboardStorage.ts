const HERMES_PTY_ATTACH_KEY = "hermes.pty.token.chat";
const CLAWMANAGER_HERMES_INSTANCE_KEY = "clawmanager.hermes.instanceId";

function clearHermesPtyAttach(storage: Storage) {
  storage.removeItem(HERMES_PTY_ATTACH_KEY);
}

/**
 * Hermes dashboard persists its PTY attach id in localStorage under a fixed key.
 * When embedded via the ClawManager same-origin proxy, that key is shared with the
 * parent page and across instance visits — reusing a stale attach causes dual
 * chat channels to fight and WebSocket 1006 reconnect loops.
 *
 * Clear the attach token before each Hermes iframe load so the dashboard mints a
 * fresh attach id (mirrors OpenClaw's prepareOpenClawControlUIStorage pattern).
 */
export function prepareHermesDashboardStorage(instanceId: number, embedUrl: string) {
  if (typeof window === "undefined") {
    return embedUrl;
  }

  try {
    const storage = window.localStorage;
    clearHermesPtyAttach(storage);
    storage.setItem(CLAWMANAGER_HERMES_INSTANCE_KEY, String(instanceId));
  } catch {
    return embedUrl;
  }

  return embedUrl;
}

export function clearHermesDashboardStorage() {
  if (typeof window === "undefined") {
    return;
  }
  try {
    clearHermesPtyAttach(window.localStorage);
  } catch {
    // ignore quota / privacy mode failures
  }
}
