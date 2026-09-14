const reloadParameter = '_appv';
const retryParameter = '_appv_retry';
const maxReloadAttempts = 2;
const foregroundCheckIntervalMs = 15_000;

export async function refreshForNewVersion() {
  if (import.meta.env.DEV) return false;

  const currentVersion = String(__APP_BUILD_VERSION__);
  const currentURL = new URL(window.location.href);
  try {
    const response = await fetch(`/version.json?t=${Date.now()}`, {
      cache: 'no-store',
      headers: { Accept: 'application/json' },
    });
    if (!response.ok) return false;

    const manifest = await response.json();
    const latestVersion = String(manifest?.version || '').trim();
    if (!latestVersion || latestVersion === currentVersion) {
      if (
        currentURL.searchParams.has(reloadParameter)
        || currentURL.searchParams.has(retryParameter)
      ) {
        currentURL.searchParams.delete(reloadParameter);
        currentURL.searchParams.delete(retryParameter);
        window.history.replaceState(null, '', currentURL);
      }
      return false;
    }

    const markedVersion = currentURL.searchParams.get(reloadParameter);
    const attempts = markedVersion === latestVersion
      ? Number(currentURL.searchParams.get(retryParameter) || 0)
      : 0;
    if (attempts >= maxReloadAttempts) {
      return false;
    }
    currentURL.searchParams.set(reloadParameter, latestVersion);
    currentURL.searchParams.set(retryParameter, String(attempts + 1));
    window.location.replace(currentURL.toString());
    return true;
  } catch {
    return false;
  }
}

export function installVersionRefresh() {
  let inFlight = null;
  const check = () => {
    if (!inFlight) {
      inFlight = refreshForNewVersion().finally(() => {
        inFlight = null;
      });
    }
    return inFlight;
  };
  const checkWhenVisible = () => {
    if (document.visibilityState === 'visible') void check();
  };

  void check();
  window.addEventListener('pageshow', check);
  window.addEventListener('focus', check);
  window.addEventListener('online', check);
  document.addEventListener('visibilitychange', checkWhenVisible);
  window.setInterval(() => {
    if (document.visibilityState === 'visible') void check();
  }, foregroundCheckIntervalMs);
}
