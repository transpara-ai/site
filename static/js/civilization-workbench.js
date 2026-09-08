/* Progressive enhancement. Automatic updates preserve drafts and keyboard focus. */
(() => {
  const shell = document.querySelector('.civilization-shell');
  if (!shell) return;
  window.civilizationWorkbenchController?.abort();
  const controller = new AbortController();
  window.civilizationWorkbenchController = controller;
  const listeners = { signal: controller.signal };
  let theme = 'system';
  try { theme = localStorage.getItem('civilization-appearance') || 'system'; } catch (_) { /* Storage can be disabled. */ }
  const applyTheme = value => {
    theme = ['light', 'dark', 'system'].includes(value) ? value : 'system';
    shell.dataset.theme = theme;
    document.querySelectorAll('[data-workbench-theme]').forEach(button => {
      button.setAttribute('aria-pressed', String(button.dataset.workbenchTheme === theme));
    });
  };
  applyTheme(theme);
  document.querySelectorAll('[data-workbench-theme]').forEach(button => button.addEventListener('click', () => {
    applyTheme(button.dataset.workbenchTheme);
    try { localStorage.setItem('civilization-appearance', theme); } catch (_) { /* The theme still applies. */ }
  }, listeners));

  const drafts = new Map();
  const expanded = new Map();
  const polls = new WeakMap();
  const focusBeforeSwap = new WeakMap();
  const pendingForms = new Set();
  const dirtyForms = new Set();
  const mobile = matchMedia('(max-width: 760px)');
  let revision = 0;
  const sourceOf = event => event.detail.requestConfig?.elt || event.detail.elt;
  const isWorkbench = source => source?.closest('.wb-workbench') && !source.closest('#workbench-runtime');
  const stateOf = () => {
    const detail = document.getElementById('workbench-detail');
    if (detail) return [detail.dataset.workId, detail.dataset.workState, detail.dataset.workOwner].join('|');
    return [...document.querySelectorAll('[data-team-work]')].map(card => card.textContent.replace(/Updated [\s\S]*$/, '').trim()).join('|');
  };
  let previousState = stateOf();
  const rememberView = () => {
    const board = document.getElementById('civilization-work-list');
    const form = document.getElementById('workbench-intake-form');
    if (!form || !board) return;
    const detail = document.getElementById('workbench-detail');
    form.elements.selected_work.value = detail?.dataset.workId || board.dataset.selectedWork || '';
    form.elements.view.value = board.dataset.view;
    form.elements.filter_repository.value = document.getElementById('workbench-repository-filter')?.value || '';
    form.elements.operator.value = document.getElementById('workbench-operator-filter')?.value || '';
    form.elements.responsible.value = document.getElementById('workbench-responsible-filter')?.value || '';
  };
  rememberView();
  const setPicker = () => {
    const picker = document.getElementById('workbench-picker');
    if (picker) picker.open = !mobile.matches;
  };
  setPicker();
  mobile.addEventListener('change', setPicker, listeners);
  document.addEventListener('input', event => {
    if (event.target.matches('[data-workbench-draft]')) drafts.set(event.target.dataset.workbenchDraft, event.target.value);
    const form = event.target.closest('[data-workbench-draft-form]');
    if (form) dirtyForms.add(form.id);
  }, listeners);
  document.addEventListener('toggle', event => {
    if (event.target.matches('details[data-workbench-details]')) expanded.set(event.target.id, event.target.open);
  }, { ...listeners, capture: true });

  const connection = message => {
    const notice = document.getElementById('workbench-connection');
    if (!notice) return;
    if (notice.textContent !== message) notice.textContent = message;
    notice.className = message ? 'wb-notice' : '';
    const live = document.querySelector('.wb-live');
    if (live && message) live.textContent = 'Reconnecting…';
  };
  const captureFocus = () => {
    const active = document.activeElement;
    if (!active?.closest('#civilization-work-list')) return null;
    return { id: active.id, parent: active.closest('[id]')?.id, tag: active.tagName.toLowerCase(), name: active.name,
      start: active.selectionStart, end: active.selectionEnd, scroll: active.scrollTop };
  };
  const restoreFocus = saved => {
    if (!saved) return;
    const element = saved.id ? document.getElementById(saved.id) : document.getElementById(saved.parent)?.querySelector(saved.name ? `${saved.tag}[name="${CSS.escape(saved.name)}"]` : saved.tag);
    if (!element) return;
    element.focus({ preventScroll: true });
    if (saved.start != null && element.setSelectionRange) element.setSelectionRange(saved.start, saved.end);
    element.scrollTop = saved.scroll;
  };
  document.addEventListener('htmx:beforeRequest', event => {
    const source = sourceOf(event);
    if (!isWorkbench(source)) return;
    if (source.id === 'civilization-work-list') polls.set(event.detail.xhr, revision);
    else {
      revision++;
      if (source.matches('form') && event.detail.requestConfig?.verb !== 'get') { pendingForms.add(source.id); dirtyForms.delete(source.id); }
    }
  }, listeners);
  document.addEventListener('htmx:beforeSwap', event => {
    const source = sourceOf(event);
    if (!isWorkbench(source)) return;
    // Bind the new controls in the same turn so rapid filter changes are not
    // lost during HTMX's default settle delay.
    if (['civilization-work-list', 'civilization-workbench'].includes(event.detail.target?.id)) event.detail.swapOverride = 'outerHTML settle:0';
    const xhr = event.detail.xhr;
    const isPoll = polls.has(xhr);
    if (isPoll && polls.get(xhr) !== revision) { event.detail.shouldSwap = false; return; }
    if (!isPoll) revision++;
    const picker = document.getElementById('workbench-picker');
    if (picker) expanded.set('workbench-picker', picker.open);
    if (!isPoll) return;
    const fragment = document.createElement('template');
    fragment.innerHTML = event.detail.serverResponse;
    const board = fragment.content.querySelector('#civilization-work-list');
    if (!board || board.dataset.available !== 'true') {
      event.detail.shouldSwap = false;
      connection('Progress is temporarily unavailable. Reconnecting automatically; your inputs are preserved.');
      return;
    }
    focusBeforeSwap.set(xhr, captureFocus());
    // Preserve local notices, fetched runtime evidence, and forms still submitting.
    const preserve = ['workbench-runtime'];
    if (document.activeElement?.closest('#workbench-filters')) preserve.push('workbench-filters');
    if (document.getElementById('workbench-action-notice')?.dataset.actionError === 'true') preserve.push('workbench-action-notice');
    for (const id of preserve) {
      if (document.getElementById(id)) fragment.content.querySelector(`#${id}`)?.setAttribute('hx-preserve', 'true');
    }
    for (const id of dirtyForms) fragment.content.querySelector(`#${CSS.escape(id)}`)?.setAttribute('hx-preserve', 'true');
    for (const id of pendingForms) {
      const detail = document.getElementById(id)?.closest('#workbench-detail');
      const incoming = fragment.content.querySelector('#workbench-detail');
      if (detail && incoming?.dataset.workId === detail.dataset.workId) incoming.setAttribute('hx-preserve', 'true');
    }
    event.detail.serverResponse = fragment.innerHTML;
  }, listeners);
  document.addEventListener('htmx:afterSwap', event => {
    if (!isWorkbench(event.target)) return;
    const source = sourceOf(event);
    const isPoll = polls.has(event.detail.xhr);
    const selection = source?.matches('[data-work-select]');
    const submission = source?.matches('form') && source.id !== 'workbench-filters';
    document.querySelectorAll('[data-workbench-draft]').forEach(field => {
      if (drafts.has(field.dataset.workbenchDraft)) field.value = drafts.get(field.dataset.workbenchDraft);
    });
    document.querySelectorAll('details[data-workbench-details]').forEach(detail => {
      if (expanded.has(detail.id)) detail.open = expanded.get(detail.id);
    });
    const picker = document.getElementById('workbench-picker');
    if (picker) picker.open = !mobile.matches || (!selection && !!expanded.get('workbench-picker'));
    rememberView();
    const currentState = stateOf();
    if (currentState !== previousState) {
      const detail = document.getElementById('workbench-detail');
      document.getElementById('workbench-announcement').textContent = detail
        ? `${detail.querySelector('h2').textContent}. ${detail.querySelector('.wb-badge').textContent}. Current owner: ${detail.dataset.workOwner}.`
        : document.querySelector('#workbench-list-title')?.textContent.trim() + '. Workbench updated.';
      previousState = currentState;
    }
    if (isPoll) { connection(''); restoreFocus(focusBeforeSwap.get(event.detail.xhr)); }
    else if (selection || submission) {
      const target = document.querySelector('#workbench-action-notice [role="alert"]') || document.getElementById('workbench-detail-title') || document.getElementById('workbench-list-title');
      if (target) { target.tabIndex = -1; target.focus({ preventScroll: !mobile.matches }); }
    }
  }, listeners);
  document.addEventListener('htmx:afterRequest', event => {
    const source = sourceOf(event);
    if (source && pendingForms.delete(source.id)) revision++;
  }, listeners);
  const requestFailed = event => {
    if (isWorkbench(sourceOf(event))) connection('Connection interrupted. Reconnecting automatically. Check progress before retrying an action; your inputs are preserved.');
  };
  document.addEventListener('htmx:responseError', requestFailed, listeners);
  document.addEventListener('htmx:sendError', requestFailed, listeners);
})();
