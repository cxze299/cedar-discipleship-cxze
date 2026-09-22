// Read the same tokens used by native controls so Ant components do not keep
// their default blue theme when the rest of the application changes.
export function readComponentTheme() {
  const style = getComputedStyle(document.documentElement);
  const token = (name) => style.getPropertyValue(name).trim();
  const pixelToken = (name) => {
    const value = Number.parseFloat(token(name));
    return Number.isFinite(value) ? value : undefined;
  };
  const borderRadius = pixelToken('--cd-radius-base');
  const controlHeight = pixelToken('--cd-control-h-desktop');

  return {
    token: {
      colorPrimary: token('--cd-primary'),
      colorSuccess: token('--cd-success'),
      colorWarning: token('--cd-warning'),
      colorError: token('--cd-danger'),
      colorInfo: token('--cd-info'),
      colorText: token('--cd-text'),
      colorTextSecondary: token('--cd-muted'),
      colorBorder: token('--cd-border'),
      colorBgLayout: token('--cd-bg'),
      fontFamily: token('--cd-font-sans'),
      ...(borderRadius === undefined ? {} : { borderRadius }),
      ...(controlHeight === undefined ? {} : { controlHeight }),
    },
  };
}
