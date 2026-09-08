/* Keep inspection details open while fresh evidence replaces the overview. */
(() => {
  let expanded = [];
  document.addEventListener('htmx:beforeSwap', event => {
    if (event.detail.target?.id !== 'civilization-mission-control') return;
    expanded = Array.from(event.detail.target.querySelectorAll('details[data-mission-detail][open]'), detail => detail.dataset.missionDetail);
  });
  document.addEventListener('htmx:afterSwap', event => {
    if (event.detail.target?.id !== 'civilization-mission-control') return;
    document.querySelectorAll('#civilization-mission-control details[data-mission-detail]').forEach(detail => {
      detail.open = expanded.includes(detail.dataset.missionDetail);
    });
  });
})();
