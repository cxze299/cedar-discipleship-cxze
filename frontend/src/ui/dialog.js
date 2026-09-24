import { reactive } from 'vue';

export const dialogState = reactive({
  open: false,
  mode: 'confirm',
  title: '',
  message: '',
  confirmLabel: '确认',
  cancelLabel: '取消',
  tone: 'default',
  value: '',
  placeholder: '',
});

let resolver = null;

function showDialog(options) {
  if (resolver) {
    resolver(dialogState.mode === 'prompt' ? null : false);
  }
  const mode = options.mode || 'confirm';
  const defaultTitle = mode === 'prompt' ? '请输入' : (mode === 'alert' ? '提示' : '请确认');

  Object.assign(dialogState, {
    open: true,
    mode,
    title: options.title || defaultTitle,
    message: options.message || '',
    confirmLabel: options.confirmLabel || (mode === 'alert' ? '知道了' : '确认'),
    cancelLabel: options.cancelLabel || '取消',
    tone: options.tone || (options.danger ? 'danger' : 'default'),
    value: options.defaultValue || '',
    placeholder: options.placeholder || '',
  });

  return new Promise((resolve) => {
    resolver = resolve;
  });
}

export function alertDialog(messageOrOptions) {
  const options = typeof messageOrOptions === 'string' ? { message: messageOrOptions } : messageOrOptions;
  return showDialog({ ...options, mode: 'alert' });
}

export function confirmDialog(messageOrOptions) {
  const options = typeof messageOrOptions === 'string' ? { message: messageOrOptions } : messageOrOptions;
  return showDialog({ ...options, mode: 'confirm' });
}

export function promptDialog(options) {
  const opts = typeof options === 'string' ? { message: options } : options;
  return showDialog({ ...opts, mode: 'prompt' });
}

export function resolveDialog(confirmed) {
  const done = resolver;
  resolver = null;
  const result = dialogState.mode === 'prompt'
    ? (confirmed ? dialogState.value.trim() : null)
    : Boolean(confirmed);
  dialogState.open = false;
  if (done) done(result);
}

if (typeof window !== 'undefined') {
  window.dialogState = dialogState;
  window.confirmDialog = confirmDialog;
  window.promptDialog = promptDialog;
  window.alertDialog = alertDialog;
  window.resolveDialog = resolveDialog;
}
