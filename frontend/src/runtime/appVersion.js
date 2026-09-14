const reloadParameter = '_appv';

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
      if (currentURL.searchParams.has(reloadParameter)) {
        currentURL.searchParams.delete(reloadParameter);
        window.history.replaceState(null, '', currentURL);
      }
      return false;
    }

    if (currentURL.searchParams.get(reloadParameter) === latestVersion) {
      return false;
    }
    currentURL.searchParams.set(reloadParameter, latestVersion);
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
  document.addEventListener('visibilitychange', checkWhenVisible);
}
