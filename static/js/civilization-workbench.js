/* Progressive enhancement: every workbench action also has a native form or link. */
(() => {
  const shell = document.querySelector('.civilization-shell');
  if (!shell) return;
  // History restoration can execute this script again on the same document.
  window.civilizationWorkbenchController?.abort();
  const controller = new AbortController();
  window.civilizationWorkbenchController = controller;
  const listeners = { signal: controller.signal };
  const theme = document.querySelector('[data-workbench-theme]');
  try { theme.value = localStorage.getItem('civilization-appearance') || 'system'; } catch (_) { /* Storage can be disabled. */ }
  const applyTheme = () => { shell.dataset.theme = theme.value; };
  if (!theme.value) theme.value = 'system';
  applyTheme();
  theme.addEventListener('change', () => {
    applyTheme();
    try { localStorage.setItem('civilization-appearance', theme.value); } catch (_) { /* Theme still applies for this page. */ }
  }, listeners);

  const drafts = new Map();
  const expanded = new Map();
  const mobile = matchMedia('(max-width: 760px)');
  const stateOf = () => {
    const detail = document.getElementById('workbench-detail');
    return detail ? [detail.dataset.workId, detail.dataset.workState, detail.dataset.workOwner].join('|') : '';
  };
  let previousState = stateOf();
  const rememberSelection = () => {
    const field = document.querySelector('[name="selected_work"]');
    const detail = document.getElementById('workbench-detail');
    if (field && detail) field.value = detail.dataset.workId;
  };
  rememberSelection();
  const setPicker = () => {
    const picker = document.getElementById('workbench-picker');
    if (picker) picker.open = !mobile.matches;
  };
  setPicker();
  mobile.addEventListener('change', setPicker, listeners);
  document.addEventListener('input', event => {
    if (event.target.matches('[data-workbench-draft]')) drafts.set(event.target.dataset.workbenchDraft, event.target.value);
  }, listeners);
  document.addEventListener('toggle', event => {
    if (event.target.matches('details[data-workbench-details]')) expanded.set(event.target.id, event.target.open);
  }, { ...listeners, capture: true });
  document.addEventListener('htmx:beforeSwap', event => {
    if (!event.detail.target?.closest('.wb-workbench')) return;
    const picker = document.getElementById('workbench-picker');
    if (picker) expanded.set('workbench-picker', picker.open);
  }, listeners);
  document.addEventListener('htmx:afterSwap', event => {
    // outerHTML removes detail.target; HTMX dispatches on the new event target.
    if (!event.target?.closest('.wb-workbench')) return;
    const source = event.detail.requestConfig?.elt;
    const selection = source?.matches('[data-work-select]');
    const submission = source?.matches('form');
    document.querySelectorAll('[data-workbench-draft]').forEach(field => {
      if (drafts.has(field.dataset.workbenchDraft)) field.value = drafts.get(field.dataset.workbenchDraft);
    });
    document.querySelectorAll('details[data-workbench-details]').forEach(detail => {
      if (expanded.has(detail.id)) detail.open = expanded.get(detail.id);
    });
    const picker = document.getElementById('workbench-picker');
    if (picker) picker.open = !mobile.matches || (!selection && !!expanded.get('workbench-picker'));
    const currentState = stateOf();
    rememberSelection();
    if (currentState !== previousState) {
      const detail = document.getElementById('workbench-detail');
      document.getElementById('workbench-announcement').textContent = detail
        ? `${detail.querySelector('h2').textContent}. ${detail.querySelector('.wb-badge').textContent}. Current owner: ${detail.dataset.workOwner}.`
        : 'Work progress updated.';
      previousState = currentState;
    }
    if (selection || submission) {
      const notice = document.querySelector('#civilization-work-list [role="alert"]');
      const focusTarget = notice || document.getElementById('workbench-detail-title');
      if (focusTarget) { focusTarget.tabIndex = -1; focusTarget.focus({ preventScroll: !mobile.matches }); }
    }
  }, listeners);
  const requestFailed = event => {
    if (!event.detail.elt?.closest('.wb-workbench')) return;
    let notice = document.getElementById('workbench-network-error');
    if (!notice) {
      notice = document.createElement('p');
      notice.id = 'workbench-network-error';
      notice.className = 'wb-notice';
      notice.setAttribute('role', 'alert');
      document.getElementById('civilization-work-list').prepend(notice);
    }
    notice.textContent = 'Connection interrupted. Your input is still here. Refresh progress before retrying an action.';
  };
  document.addEventListener('htmx:responseError', requestFailed, listeners);
  document.addEventListener('htmx:sendError', requestFailed, listeners);
})();
