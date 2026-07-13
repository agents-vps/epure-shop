/**
 * shop/js/admin.js — E-commerce Admin Dashboard
 * Vanilla ES6. All components targeted via data-* attributes.
 *
 * Components:
 *   [data-component="admin-sidebar"]      — collapsible sidebar (toggle icon-only mode)
 *   [data-component="table-select"]       — table row checkboxes, select-all, shift-click range
 *   [data-component="bulk-action"]        — bulk action trigger (delete, status change, etc.)
 *   [data-component="notification-count"] — notification badge with dynamic count
 *   [data-component="search-filter"]      — live search filter for <table> rows
 *   [data-component="skeleton"]           — skeleton loading placeholder region
 */

(function () {
  'use strict';

  /* ------------------------------------------------------------------ */
  /*  Utility helpers                                                    */
  /* ------------------------------------------------------------------ */

  /**
   * Find the closest ancestor matching a selector (IE-safe fallback).
   */
  function closest(el, selector) {
    if (!el) return null;
    if (el.closest) return el.closest(selector);
    while (el && el !== document) {
      if (el.matches && el.matches(selector)) return el;
      el = el.parentNode;
    }
    return null;
  }

  /**
   * Return all checkbox inputs inside a container (table row / thead / tbody).
   */
  function getCheckboxes(container) {
    return container.querySelectorAll(
      'input[type="checkbox"][data-row-checkbox]'
    );
  }

  /**
   * Dispatch a custom DOM event on an element.
   */
  function emit(el, name, detail) {
    el.dispatchEvent(
      new CustomEvent(name, { bubbles: true, cancelable: true, detail: detail || {} })
    );
  }

  /* ================================================================== */
  /*  1. ADMIN SIDEBAR  ([data-component="admin-sidebar"])               */
  /* ================================================================== */

  function initSidebar(root) {
    var sidebar = root.querySelector('[data-component="admin-sidebar"]');
    if (!sidebar) return;

    // Look for a toggle button inside (or sibling) the sidebar.
    // Convention: the toggle button carries [data-action="sidebar-toggle"].
    var toggleBtn =
      sidebar.querySelector('[data-action="sidebar-toggle"]') ||
      document.querySelector('[data-action="sidebar-toggle"]');

    // Re-apply saved state from localStorage.
    var collapsed = localStorage.getItem('shop_admin_sidebar_collapsed') === 'true';
    if (collapsed) {
      sidebar.setAttribute('data-collapsed', 'true');
      sidebar.classList.add('is-collapsed');
    }

    if (toggleBtn) {
      toggleBtn.addEventListener('click', function (e) {
        e.preventDefault();
        var isCollapsed = sidebar.getAttribute('data-collapsed') === 'true';
        if (isCollapsed) {
          sidebar.removeAttribute('data-collapsed');
          sidebar.classList.remove('is-collapsed');
          localStorage.setItem('shop_admin_sidebar_collapsed', 'false');
        } else {
          sidebar.setAttribute('data-collapsed', 'true');
          sidebar.classList.add('is-collapsed');
          localStorage.setItem('shop_admin_sidebar_collapsed', 'true');
        }
        emit(sidebar, 'sidebar:toggle', { collapsed: !isCollapsed });
      });
    }
  }

  /* ================================================================== */
  /*  2. TABLE ROW SELECTION  ([data-component="table-select"])         */
  /*     — select-all checkbox (in thead)                                */
  /*     — shift-click range selection                                   */
  /*     — checkbox state drives a [data-selected] attribute on <tr>    */
  /* ================================================================== */

  var _lastCheckedIdx = null; // per-table map tracked via table reference

  function getTableFromEl(el) {
    return closest(el, 'table');
  }

  /**
   * Return an array of all data-row-checkbox checkboxes in document order.
   */
  function rowCheckboxes(table) {
    return Array.from(table.querySelectorAll('input[type="checkbox"][data-row-checkbox]'));
  }

  /**
   * Return the select-all checkbox — expects [data-select-all] on the input itself.
   */
  function selectAllCheckbox(table) {
    return table.querySelector('input[type="checkbox"][data-select-all]');
  }

  /**
   * Reflect selected state on each row via data-selected attribute + class.
   */
  function syncRowState(checkbox) {
    var row = closest(checkbox, 'tr');
    if (!row) return;
    if (checkbox.checked) {
      row.setAttribute('data-selected', 'true');
      row.classList.add('is-selected');
    } else {
      row.removeAttribute('data-selected');
      row.classList.remove('is-selected');
    }
  }

  /**
   * Update the select-all checkbox to reflect whether all/none/some rows are checked.
   */
  function updateSelectAll(table) {
    var allCb = selectAllCheckbox(table);
    if (!allCb) return;
    var boxes = rowCheckboxes(table);
    var checked = boxes.filter(function (cb) { return cb.checked; });
    if (checked.length === 0) {
      allCb.checked = false;
      allCb.indeterminate = false;
    } else if (checked.length === boxes.length) {
      allCb.checked = true;
      allCb.indeterminate = false;
    } else {
      allCb.checked = false;
      allCb.indeterminate = true;
    }
  }

  /**
   * Emit a summary event with selected row count + optional IDs.
   */
  function emitSelectionChange(table) {
    var boxes = rowCheckboxes(table).filter(function (cb) { return cb.checked; });
    var ids = boxes.map(function (cb) {
      return cb.getAttribute('data-id') || cb.value || null;
    }).filter(Boolean);
    emit(table, 'table:selection', { count: boxes.length, ids: ids });
  }

  function initTableSelect(root) {
    var tables = root.querySelectorAll('[data-component="table-select"]');
    tables.forEach(function (table) {
      // Reset last-checked index on table interaction.
      table._lastCheckedIdx = null;

      // --- select-all ---
      var allCb = selectAllCheckbox(table);
      if (allCb) {
        allCb.addEventListener('change', function () {
          var boxes = rowCheckboxes(table);
          boxes.forEach(function (cb, i) {
            cb.checked = allCb.checked;
            syncRowState(cb);
          });
          table._lastCheckedIdx = null;
          updateSelectAll(table);
          emitSelectionChange(table);
        });
      }

      // --- individual checkbox click (with shift support) ---
      table.addEventListener('click', function (e) {
        var cb = e.target;
        if (cb.type !== 'checkbox' || !cb.hasAttribute('data-row-checkbox')) return;

        var boxes = rowCheckboxes(table);
        var idx = boxes.indexOf(cb);

        // Shift-click range selection
        if (e.shiftKey && table._lastCheckedIdx != null && idx !== -1) {
          var start = Math.min(table._lastCheckedIdx, idx);
          var end = Math.max(table._lastCheckedIdx, idx);
          for (var i = start; i <= end; i++) {
            boxes[i].checked = cb.checked;
            syncRowState(boxes[i]);
          }
        }

        syncRowState(cb);
        table._lastCheckedIdx = idx;
        updateSelectAll(table);
        emitSelectionChange(table);
      });

      // Initial sync
      rowCheckboxes(table).forEach(syncRowState);
      updateSelectAll(table);
    });
  }

  /* ================================================================== */
  /*  3. BULK ACTIONS  ([data-component="bulk-action"])                  */
  /*     Reads selected checkboxes from the closest [data-component=     */
  /*     "table-select"] ancestor/sibling and applies the action.        */
  /* ================================================================== */

  function initBulkActions(root) {
    var actions = root.querySelectorAll('[data-component="bulk-action"]');
    actions.forEach(function (actionEl) {
      // Determine which table this bulk action targets.
      var tableId = actionEl.getAttribute('data-table');
      var table;
      if (tableId) {
        table = document.getElementById(tableId);
      } else {
        // Try sibling or parent lookup
        table =
          closest(actionEl, '[data-component="table-select"]') ||
          actionEl.closest('[data-component="table-select"]');
      }
      if (!table) {
        // Fallback: first table-select on the page
        table = document.querySelector('[data-component="table-select"]');
      }

      // Action type: "delete" | "status" | "archive" | custom string
      var actionType = actionEl.getAttribute('data-action') || 'default';

      actionEl.addEventListener('click', function (e) {
        var boxes = table
          ? rowCheckboxes(table).filter(function (cb) { return cb.checked; })
          : [];
        var ids = boxes.map(function (cb) {
          return cb.getAttribute('data-id') || cb.value || null;
        }).filter(Boolean);

        if (ids.length === 0) {
          emit(actionEl, 'bulk:empty', { action: actionType });
          return;
        }

        emit(actionEl, 'bulk:apply', {
          action: actionType,
          ids: ids,
          count: ids.length,
        });

        // If data-confirm is set, require confirmation before proceeding.
        // Otherwise fire immediately.
        var confirmMsg = actionEl.getAttribute('data-confirm');
        if (confirmMsg && !window.confirm(confirmMsg)) {
          emit(actionEl, 'bulk:cancelled', { action: actionType });
          return;
        }

        // Execute: dispatch a final event so parent code can hook in.
        emit(actionEl, 'bulk:execute', {
          action: actionType,
          ids: ids,
          count: ids.length,
        });

        // Optionally show a skeleton / loading state for the table.
        if (actionEl.hasAttribute('data-show-skeleton')) {
          showSkeleton(table);
        }

        // Optionally reload the page after an action.
        if (actionEl.hasAttribute('data-reload')) {
          var delay = parseInt(actionEl.getAttribute('data-reload-delay'), 10) || 1500;
          setTimeout(function () {
            window.location.reload();
          }, delay);
        }
      });
    });
  }

  /* ================================================================== */
  /*  4. SEARCH FILTER  ([data-component="search-filter"])              */
  /*     Live-filter table rows by visible text content.                 */
  /*     Expects [data-search-target] on the search input pointing to    */
  /*     a table id, or defaults to the closest [data-component="table-  */
  /*     select"] table.                                                 */
  /* ================================================================== */

  function initSearchFilter(root) {
    var filters = root.querySelectorAll('[data-component="search-filter"]');
    filters.forEach(function (input) {
      if (input.tagName !== 'INPUT') return;

      var debounceMs = parseInt(input.getAttribute('data-debounce'), 10) || 200;
      var targetId = input.getAttribute('data-search-target');
      var table;
      if (targetId) {
        table = document.getElementById(targetId);
      } else {
        table = closest(input, '[data-component="table-select"]') ||
                (input.closest && input.closest('[data-component="table-select"]')) ||
                document.querySelector('[data-component="table-select"]');
      }

      if (!table) return;

      var timer = null;

      input.addEventListener('input', function () {
        clearTimeout(timer);
        timer = setTimeout(function () {
          var query = input.value.trim().toLowerCase();
          var rows = table.querySelectorAll('tbody tr');
          var visibleCount = 0;

          rows.forEach(function (row) {
            // Skip skeleton rows / empty-state rows.
            if (
              row.hasAttribute('data-skeleton-row') ||
              row.hasAttribute('data-empty-row')
            ) {
              return;
            }
            var text = (row.textContent || '').toLowerCase();
            if (!query || text.indexOf(query) !== -1) {
              row.style.display = '';
              row.removeAttribute('hidden');
              visibleCount++;
            } else {
              row.style.display = 'none';
              row.setAttribute('hidden', '');
            }
          });

          emit(input, 'search:filter', {
            query: query,
            visible: visibleCount,
            total: rows.length,
          });

          // Show/hide "no results" row if provided.
          var noResults = table.querySelector('[data-empty-row="search"]');
          if (noResults) {
            noResults.style.display = query && visibleCount === 0 ? '' : 'none';
          }
        }, debounceMs);
      });

      // Clear button support: [data-search-clear] inside or sibling.
      var clearBtn =
        input.parentNode.querySelector('[data-search-clear]') ||
        document.querySelector('[data-search-clear][data-for="' + (input.id || input.name || '') + '"]');
      if (clearBtn) {
        clearBtn.addEventListener('click', function () {
          input.value = '';
          input.dispatchEvent(new Event('input', { bubbles: true }));
          input.focus();
        });
      }
    });
  }

  /* ================================================================== */
  /*  5. SKELETON LOADING  ([data-component="skeleton"])                */
  /*     Shows animated placeholder rows in a table body.                */
  /*     Usage:                                                          */
  /*       showSkeleton(table)      — displays skeleton rows             */
  /*       hideSkeleton(table)      — removes skeleton rows              */
  /*       toggleSkeleton(table)    — flips                              */
  /*     Data attributes on the skeleton container:                      */
  /*       data-rows="5"            — number of skeleton rows           */
  /*       data-columns="4"         — number of columns per row         */
  /* ================================================================== */

  function getSkeletonContainer(table) {
    return table.querySelector('[data-component="skeleton"]');
  }

  function buildSkeletonRow(cols) {
    var tr = document.createElement('tr');
    tr.setAttribute('data-skeleton-row', 'true');
    tr.setAttribute('aria-hidden', 'true');
    for (var i = 0; i < cols; i++) {
      var td = document.createElement('td');
      // Randomize width slightly for realism
      var w = 60 + Math.floor(Math.random() * 35);
      td.innerHTML =
        '<span class="skeleton-bar" style="width:' +
        w +
        '%;" aria-hidden="true">&nbsp;</span>';
      tr.appendChild(td);
    }
    return tr;
  }

  /**
   * Show skeleton loading rows for a table.
   */
  window.showSkeleton = function (table) {
    if (typeof table === 'string') table = document.querySelector(table);
    if (!table) return;

    var container = getSkeletonContainer(table);
    var tbody = table.querySelector('tbody');
    if (!tbody) return;

    var rowCount = container
      ? parseInt(container.getAttribute('data-rows'), 10) || 5
      : 5;
    var colCount = container
      ? parseInt(container.getAttribute('data-columns'), 10) || 4
      : tbody.querySelector('tr')
        ? tbody.querySelector('tr').querySelectorAll('td, th').length
        : 4;

    // Hide real rows.
    tbody.querySelectorAll('tr:not([data-skeleton-row])').forEach(function (r) {
      r.style.display = 'none';
      r.setAttribute('hidden', '');
    });

    // Remove old skeletons.
    tbody.querySelectorAll('[data-skeleton-row]').forEach(function (r) {
      r.remove();
    });

    // Insert new skeleton rows.
    for (var i = 0; i < rowCount; i++) {
      tbody.appendChild(buildSkeletonRow(colCount));
    }

    if (container) {
      container.setAttribute('data-visible', 'true');
      container.classList.add('is-visible');
    }
    emit(table, 'skeleton:show', { rows: rowCount, columns: colCount });
  };

  /**
   * Hide skeleton rows and restore real rows.
   */
  window.hideSkeleton = function (table) {
    if (typeof table === 'string') table = document.querySelector(table);
    if (!table) return;

    var container = getSkeletonContainer(table);
    var tbody = table.querySelector('tbody');
    if (!tbody) return;

    // Remove skeleton rows.
    tbody.querySelectorAll('[data-skeleton-row]').forEach(function (r) {
      r.remove();
    });

    // Restore real rows.
    tbody.querySelectorAll('tr[hidden]').forEach(function (r) {
      r.style.display = '';
      r.removeAttribute('hidden');
    });

    if (container) {
      container.removeAttribute('data-visible');
      container.classList.remove('is-visible');
    }
    emit(table, 'skeleton:hide', {});
  };

  /**
   * Simulate an async load: show skeleton → wait → hide.
   */
  window.simulateLoad = function (table, durationMs) {
    if (typeof table === 'string') table = document.querySelector(table);
    if (!table) return;
    durationMs = durationMs || 1500;
    showSkeleton(table);
    setTimeout(function () {
      hideSkeleton(table);
      emit(table, 'skeleton:complete', { duration: durationMs });
    }, durationMs);
  };

  /* ================================================================== */
  /*  6. NOTIFICATION BADGE  ([data-component="notification-count"])    */
  /*     Displays and updates a count badge.                             */
  /*     Methods:                                                        */
  /*       setNotificationCount(elOrSelector, count)                     */
  /*       incrementNotification(elOrSelector, delta)                    */
  /*     Data attributes:                                                */
  /*       data-max="99"       — max displayed value (shows "99+" after) */
  /* ================================================================== */

  /**
   * Set the notification count on a badge element.
   * @param {Element|string} el  The badge or a selector.
   * @param {number}         count
   */
  window.setNotificationCount = function (el, count) {
    if (typeof el === 'string') el = document.querySelector(el);
    if (!el) return;
    var max = parseInt(el.getAttribute('data-max'), 10) || 99;
    var display = count > max ? max + '+' : String(count);

    // Animate if value changed.
    var prev = el.textContent;
    if (prev !== display) {
      el.textContent = display;
      el.classList.remove('pulse');
      // Force reflow for restart.
      void el.offsetWidth;
      el.classList.add('pulse');
    }

    el.setAttribute('data-count', count);
    el.style.display = count === 0 ? 'none' : '';

    emit(el, 'notification:update', { count: count, display: display });
  };

  /**
   * Increment (or decrement) the notification badge by a delta.
   */
  window.incrementNotification = function (el, delta) {
    if (typeof el === 'string') el = document.querySelector(el);
    if (!el) return;
    var cur = parseInt(el.getAttribute('data-count'), 10) || 0;
    setNotificationCount(el, Math.max(0, cur + (delta || 1)));
  };

  /* ================================================================== */
  /*  7. BOOTSTRAP                                                       */
  /* ================================================================== */

  function init(root) {
    root = root || document;
    initSidebar(root);
    initTableSelect(root);
    initBulkActions(root);
    initSearchFilter(root);

    // Initialize notification badges with their starting counts.
    root.querySelectorAll('[data-component="notification-count"]').forEach(function (badge) {
      var initial = parseInt(badge.getAttribute('data-count'), 10);
      if (!isNaN(initial)) {
        setNotificationCount(badge, initial);
      } else if (badge.textContent.trim() !== '') {
        var parsed = parseInt(badge.textContent.trim(), 10);
        if (!isNaN(parsed)) {
          setNotificationCount(badge, parsed);
        }
      }
    });
  }

  // Auto-bootstrap on DOM ready.
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function () { init(); });
  } else {
    init();
  }
})();
