/* ═══════════════════════════════════════════
   UI — Vanilla JS interactivity
   Épure — Boutique minimaliste
   All selectors target [data-*] attributes,
   never CSS classes.
   ═══════════════════════════════════════════ */

;(function () {
  'use strict';

  /* ─── Shared helpers ─── */

  const $ = (sel, ctx) => (ctx || document).querySelector(sel);
  const $$ = (sel, ctx) => [...(ctx || document).querySelectorAll(sel)];

  /** Return all focusable descendants of `el`. */
  function getFocusable(el) {
    const sel = [
      'a[href]',
      'button:not([disabled])',
      'input:not([disabled])',
      'select:not([disabled])',
      'textarea:not([disabled])',
      '[tabindex]:not([tabindex="-1"])',
    ].join(',');
    return $$(sel, el).filter(function (n) {
      return n.offsetParent !== null || n === document.activeElement;
    });
  }

  /** Trap focus inside `container`, wrapping around. */
  function trapFocus(container, e) {
    if (e.key !== 'Tab') return;
    const focusable = getFocusable(container);
    if (!focusable.length) {
      e.preventDefault();
      return;
    }
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (e.shiftKey) {
      if (document.activeElement === first) {
        e.preventDefault();
        last.focus();
      }
    } else {
      if (document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
    }
  }

  /** Restore focus to the element stored in data-restore-focus. */
  function restoreFocus(container) {
    const id = container.getAttribute('data-restore-focus');
    if (id) {
      const el = document.getElementById(id);
      if (el) {
        el.focus();
        container.removeAttribute('data-restore-focus');
      }
    }
  }

  /* ═══════════════════════════════════════
     Mobile Menu
     ═══════════════════════════════════════ */

  function initMobileMenu() {
    const menu = $('[data-component="mobile-menu"]');
    if (!menu) return;

    const openTriggers = $$('[data-action="open-mobile-menu"]');
    const closeTriggers = $$('[data-action="close-mobile-menu"]', menu);

    function open() {
      menu.setAttribute('data-state', 'open');
      menu.setAttribute('aria-hidden', 'false');
      // Store trigger focus for later restoration
      if (document.activeElement) {
        menu.setAttribute('data-restore-focus', document.activeElement.id || '');
        if (!document.activeElement.id) {
          const uid = 'mf-' + Math.random().toString(36).slice(2, 9);
          document.activeElement.id = uid;
          menu.setAttribute('data-restore-focus', uid);
        }
      }
      // Move focus into menu
      const firstFocusable = getFocusable(menu)[0];
      if (firstFocusable) firstFocusable.focus();
      document.addEventListener('keydown', onKeyDown);
    }

    function close() {
      menu.setAttribute('data-state', 'closed');
      menu.setAttribute('aria-hidden', 'true');
      restoreFocus(menu);
      document.removeEventListener('keydown', onKeyDown);
    }

    function toggle() {
      menu.getAttribute('data-state') === 'open' ? close() : open();
    }

    function onKeyDown(e) {
      if (e.key === 'Escape') {
        close();
        e.stopPropagation();
      }
      trapFocus(menu, e);
    }

    openTriggers.forEach(function (btn) {
      btn.addEventListener('click', function (e) {
        e.preventDefault();
        open();
      });
    });

    closeTriggers.forEach(function (btn) {
      btn.addEventListener('click', function (e) {
        e.preventDefault();
        close();
      });
    });

    // Click backdrop to close
    menu.addEventListener('click', function (e) {
      if (e.target === menu) close();
    });
  }

  /* ═══════════════════════════════════════
     Cart Drawer
     ═══════════════════════════════════════ */

  function initCartDrawer() {
    const drawer = $('[data-component="cart-drawer"]');
    if (!drawer) return;

    const openTriggers = $$('[data-action="open-cart"]');
    const closeTriggers = $$('[data-action="close-cart"]', drawer);

    // The drawer-backdrop wraps the .drawer panel in CSS;
    // the JS component is the backdrop element.
    const isBackdrop = drawer.classList.contains('drawer-backdrop');

    function open() {
      drawer.setAttribute('data-state', 'open');
      drawer.setAttribute('aria-hidden', 'false');
      if (document.activeElement) {
        drawer.setAttribute('data-restore-focus', document.activeElement.id || '');
        if (!document.activeElement.id) {
          const uid = 'cd-' + Math.random().toString(36).slice(2, 9);
          document.activeElement.id = uid;
          drawer.setAttribute('data-restore-focus', uid);
        }
      }
      const firstFocusable = getFocusable(drawer)[0];
      if (firstFocusable) firstFocusable.focus();
      document.addEventListener('keydown', onKeyDown);
      document.body.style.overflow = 'hidden';
    }

    function close() {
      drawer.setAttribute('data-state', 'closed');
      drawer.setAttribute('aria-hidden', 'true');
      restoreFocus(drawer);
      document.removeEventListener('keydown', onKeyDown);
      document.body.style.overflow = '';
    }

    function onKeyDown(e) {
      if (e.key === 'Escape') {
        close();
        e.stopPropagation();
      }
      trapFocus(drawer, e);
    }

    openTriggers.forEach(function (btn) {
      btn.addEventListener('click', function (e) {
        e.preventDefault();
        open();
      });
    });

    closeTriggers.forEach(function (btn) {
      btn.addEventListener('click', function (e) {
        e.preventDefault();
        close();
      });
    });

    // Click backdrop to close
    if (isBackdrop) {
      drawer.addEventListener('click', function (e) {
        if (e.target === drawer) close();
      });
    }
  }

  /* ═══════════════════════════════════════
     Modal (with focus trap)
     ═══════════════════════════════════════ */

  const modalStack = [];

  function initModals() {
    const modals = $$('[data-component="modal"]');

    modals.forEach(function (modal) {
      const closeTriggers = $$('[data-action="close-modal"]', modal);

      function open() {
        // Close any already-open modal on the stack first
        if (modalStack.length) {
          closeModal(modalStack[modalStack.length - 1], true);
        }
        modalStack.push(modal);

        modal.setAttribute('data-state', 'open');
        modal.setAttribute('aria-hidden', 'false');
        modal.setAttribute('role', 'dialog');
        modal.setAttribute('aria-modal', 'true');

        if (document.activeElement) {
          modal.setAttribute('data-restore-focus', document.activeElement.id || '');
          if (!document.activeElement.id) {
            const uid = 'mdl-' + Math.random().toString(36).slice(2, 9);
            document.activeElement.id = uid;
            modal.setAttribute('data-restore-focus', uid);
          }
        }

        // Focus first focusable element, or the modal itself
        const focusable = getFocusable(modal);
        (focusable[0] || modal).focus();

        document.addEventListener('keydown', onKeyDown);
        document.body.style.overflow = 'hidden';
      }

      function close() {
        closeModal(modal, false);
      }

      function onKeyDown(e) {
        if (e.key === 'Escape') {
          close();
          e.stopPropagation();
        }
        trapFocus(modal, e);
      }

      closeTriggers.forEach(function (btn) {
        btn.addEventListener('click', function (e) {
          e.preventDefault();
          close();
        });
      });

      // Click backdrop to close
      modal.addEventListener('click', function (e) {
        if (e.target === modal) close();
      });

      // Store open/close on the element
      modal._modalOpen = open;
      modal._modalClose = close;
    });

    // Delegate: any [data-action="open-modal"] with data-target="#id"
    document.addEventListener('click', function (e) {
      const trigger = e.target.closest('[data-action="open-modal"]');
      if (!trigger) return;
      e.preventDefault();
      const targetId = trigger.getAttribute('data-target');
      if (!targetId) return;
      const modal = document.querySelector(targetId);
      if (modal && modal._modalOpen) modal._modalOpen();
    });
  }

  function closeModal(modal, silent) {
    const idx = modalStack.indexOf(modal);
    if (idx !== -1) modalStack.splice(idx, 1);

    modal.setAttribute('data-state', 'closed');
    modal.setAttribute('aria-hidden', 'true');
    modal.removeAttribute('aria-modal');

    restoreFocus(modal);

    if (!silent) {
      document.removeEventListener('keydown', modal._onKeyDown);
    }

    // If no more modals open, re-enable body scroll
    if (!modalStack.length) {
      document.body.style.overflow = '';
    }
  }

  /* ═══════════════════════════════════════
     Tabs
     ═══════════════════════════════════════ */

  function initTabs() {
    const tabContainers = $$('[data-component="tabs"]');

    tabContainers.forEach(function (container) {
      const triggers = $$('[data-action="switch-tab"]', container);
      const panels = $$('[data-tab-panel]', container);
      const tabMap = {};

      // Build a map of panel id → panel element
      panels.forEach(function (panel) {
        const id = panel.getAttribute('id');
        if (id) tabMap[id] = panel;
      });

      triggers.forEach(function (trigger) {
        trigger.addEventListener('click', function (e) {
          e.preventDefault();
          const panelId = trigger.getAttribute('data-target');
          if (!panelId) return;

          // Deactivate all triggers in this container
          triggers.forEach(function (t) {
            t.setAttribute('aria-selected', 'false');
            t.setAttribute('tabindex', '-1');
          });

          // Activate this trigger
          trigger.setAttribute('aria-selected', 'true');
          trigger.setAttribute('tabindex', '0');

          // Deactivate all panels in this container
          panels.forEach(function (p) {
            p.setAttribute('data-state', 'inactive');
          });

          // Activate target panel
          const targetPanel = tabMap[panelId] || document.getElementById(panelId);
          if (targetPanel) {
            targetPanel.setAttribute('data-state', 'active');
          }
        });

        // Keyboard: arrow keys
        trigger.addEventListener('keydown', function (e) {
          const idx = triggers.indexOf(trigger);
          let next;

          if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
            e.preventDefault();
            next = triggers[(idx + 1) % triggers.length];
          } else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
            e.preventDefault();
            next = triggers[(idx - 1 + triggers.length) % triggers.length];
          } else if (e.key === 'Home') {
            e.preventDefault();
            next = triggers[0];
          } else if (e.key === 'End') {
            e.preventDefault();
            next = triggers[triggers.length - 1];
          }

          if (next) {
            next.focus();
            next.click();
          }
        });
      });
    });
  }

  /* ═══════════════════════════════════════
     Toast Notifications
     ═══════════════════════════════════════ */

  const toastTimers = {};

  function initToast() {
    // Ensure a toast container exists
    let container = $('[data-component="toast"]');
    if (!container) {
      container = document.createElement('div');
      container.setAttribute('data-component', 'toast');
      container.className = 'toast-container';
      container.setAttribute('aria-live', 'polite');
      container.setAttribute('aria-atomic', 'false');
      document.body.appendChild(container);
    }

    // Delegate close button clicks inside the container
    container.addEventListener('click', function (e) {
      const closeBtn = e.target.closest('[data-action="close-toast"]');
      if (!closeBtn) return;
      const toast = closeBtn.closest('[data-toast-id]');
      if (toast) dismissToast(toast);
    });
  }

  function dismissToast(el) {
    const id = el.getAttribute('data-toast-id');
    if (id && toastTimers[id]) {
      clearTimeout(toastTimers[id]);
      delete toastTimers[id];
    }
    el.addEventListener('transitionend', function () {
      if (el.parentNode) el.parentNode.removeChild(el);
    }, { once: true });
    el.style.opacity = '0';
    el.style.transform = 'translateX(100%)';
    // Force removal after transition in case transitionend doesn't fire
    setTimeout(function () {
      if (el.parentNode) el.parentNode.removeChild(el);
    }, 400);
  }

  /**
   * showToast({ type: 'success'|'error'|'warning'|'info', title, message, duration? })
   * duration in ms (default 5000). Pass 0 to disable auto-dismiss.
   */
  function showToast(opts) {
    const container = $('[data-component="toast"]');
    if (!container) return;

    const type = opts.type || 'info';
    const duration = opts.duration !== undefined ? opts.duration : 5000;
    const id = 'tst-' + Math.random().toString(36).slice(2, 9);

    // Icon paths (simple inline SVG placeholders)
    const icons = {
      success: '<svg viewBox="0 0 20 20" fill="currentColor" class="toast__icon"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"/></svg>',
      error: '<svg viewBox="0 0 20 20" fill="currentColor" class="toast__icon"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd"/></svg>',
      warning: '<svg viewBox="0 0 20 20" fill="currentColor" class="toast__icon"><path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd"/></svg>',
      info: '<svg viewBox="0 0 20 20" fill="currentColor" class="toast__icon"><path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd"/></svg>',
    };

    const toast = document.createElement('div');
    toast.className = 'toast toast--' + type;
    toast.setAttribute('data-toast-id', id);
    toast.setAttribute('role', 'status');
    toast.innerHTML =
      '<div class="toast__icon-wrap">' + (icons[type] || icons.info) + '</div>' +
      '<div class="toast__content">' +
        (opts.title ? '<div class="toast__title">' + escapeHTML(opts.title) + '</div>' : '') +
        (opts.message ? '<div class="toast__message">' + escapeHTML(opts.message) + '</div>' : '') +
      '</div>' +
      '<button class="toast__close" data-action="close-toast" aria-label="Fermer">' +
        '<svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M3 3l8 8M11 3l-8 8"/></svg>' +
      '</button>';

    container.appendChild(toast);

    if (duration > 0) {
      toastTimers[id] = setTimeout(function () {
        dismissToast(toast);
      }, duration);
    }
  }

  function escapeHTML(str) {
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }

  /* ═══════════════════════════════════════
     Accordion
     ═══════════════════════════════════════ */

  function initAccordion() {
    const accordions = $$('[data-component="accordion"]');

    accordions.forEach(function (acc) {
      const triggers = $$('[data-action="toggle-accordion"]', acc);

      triggers.forEach(function (trigger) {
        trigger.addEventListener('click', function () {
          const panelId = trigger.getAttribute('data-target');
          const panel = panelId
            ? (acc.querySelector('#' + panelId) || document.getElementById(panelId))
            : trigger.nextElementSibling;

          if (!panel) return;

          const isOpen = panel.getAttribute('data-state') === 'open';

          if (isOpen) {
            panel.setAttribute('data-state', 'closed');
            trigger.setAttribute('aria-expanded', 'false');
          } else {
            // Optionally close siblings (accordion behavior)
            // For now, toggle independently unless data-accordion-type="single"
            if (acc.getAttribute('data-accordion-type') === 'single') {
              $$('[data-action="toggle-accordion"]', acc).forEach(function (t) {
                t.setAttribute('aria-expanded', 'false');
              });
              $$('[data-tab-panel]', acc).forEach(function (p) {
                p.setAttribute('data-state', 'closed');
              });
              // Also handle panels targeted by aria-controls pattern
              $$('[data-state="open"]', acc).forEach(function (p) {
                if (p !== panel) p.setAttribute('data-state', 'closed');
              });
            }

            panel.setAttribute('data-state', 'open');
            trigger.setAttribute('aria-expanded', 'true');
          }
        });

        // Keyboard: Enter/Space to toggle
        trigger.addEventListener('keydown', function (e) {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            trigger.click();
          }
        });
      });
    });
  }

  /* ═══════════════════════════════════════
     Dropdown
     ═══════════════════════════════════════ */

  function initDropdown() {
    // Close any open dropdown when clicking outside or pressing Escape
    document.addEventListener('click', function (e) {
      const openDropdowns = $$('[data-component="dropdown"]');
      openDropdowns.forEach(function (dd) {
        // If click is outside this dropdown, close it
        if (!dd.contains(e.target)) {
          const menu = $('[data-dropdown-menu]', dd);
          if (menu && menu.getAttribute('data-state') === 'open') {
            menu.setAttribute('data-state', 'closed');
            const trigger = $('[data-action="toggle-dropdown"][aria-expanded="true"]', dd);
            if (trigger) trigger.setAttribute('aria-expanded', 'false');
          }
        }
      });
    });

    document.addEventListener('keydown', function (e) {
      if (e.key === 'Escape') {
        const openDropdowns = $$('[data-component="dropdown"]');
        openDropdowns.forEach(function (dd) {
          const menu = $('[data-dropdown-menu]', dd);
          if (menu && menu.getAttribute('data-state') === 'open') {
            menu.setAttribute('data-state', 'closed');
            const trigger = $('[data-action="toggle-dropdown"][aria-expanded="true"]', dd);
            if (trigger) {
              trigger.setAttribute('aria-expanded', 'false');
              trigger.focus();
            }
          }
        });
      }
    });

    // Attach toggle handlers to each dropdown trigger
    const dropdowns = $$('[data-component="dropdown"]');

    dropdowns.forEach(function (dd) {
      const triggers = $$('[data-action="toggle-dropdown"]', dd);
      const menu = $('[data-dropdown-menu]', dd);

      if (!menu || !triggers.length) return;

      triggers.forEach(function (trigger) {
        trigger.addEventListener('click', function (e) {
          e.preventDefault();
          e.stopPropagation();

          const isOpen = menu.getAttribute('data-state') === 'open';

          // Close all other dropdowns first
          $$('[data-dropdown-menu][data-state="open"]').forEach(function (m) {
            if (m !== menu) m.setAttribute('data-state', 'closed');
          });
          $$('[data-action="toggle-dropdown"][aria-expanded="true"]').forEach(function (t) {
            if (t !== trigger) t.setAttribute('aria-expanded', 'false');
          });

          if (isOpen) {
            menu.setAttribute('data-state', 'closed');
            trigger.setAttribute('aria-expanded', 'false');
          } else {
            menu.setAttribute('data-state', 'open');
            trigger.setAttribute('aria-expanded', 'true');

            // Focus first item
            const firstItem = $('[data-dropdown-item]', menu);
            if (firstItem) firstItem.focus();
          }
        });

        // Keyboard controls when trigger is focused
        trigger.addEventListener('keydown', function (e) {
          if (e.key === 'ArrowDown' || e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            if (menu.getAttribute('data-state') !== 'open') {
              trigger.click();
            } else {
              // Focus first item
              const firstItem = $('[data-dropdown-item]', menu);
              if (firstItem) firstItem.focus();
            }
          }
        });
      });

      // Keyboard navigation within dropdown menu
      menu.addEventListener('keydown', function (e) {
        const items = $$('[data-dropdown-item]', menu);

        if (e.key === 'ArrowDown') {
          e.preventDefault();
          const idx = items.indexOf(document.activeElement);
          const next = items[(idx + 1) % items.length];
          if (next) next.focus();
        } else if (e.key === 'ArrowUp') {
          e.preventDefault();
          const idx = items.indexOf(document.activeElement);
          const next = items[(idx - 1 + items.length) % items.length];
          if (next) next.focus();
        } else if (e.key === 'Escape') {
          menu.setAttribute('data-state', 'closed');
          const t = $('[data-action="toggle-dropdown"]', dd);
          if (t) {
            t.setAttribute('aria-expanded', 'false');
            t.focus();
          }
        } else if (e.key === 'Tab') {
          // Close on tab out
          menu.setAttribute('data-state', 'closed');
          const t = $('[data-action="toggle-dropdown"]', dd);
          if (t) t.setAttribute('aria-expanded', 'false');
        }
      });
    });
  }

  /* ═══════════════════════════════════════
     Boot
     ═══════════════════════════════════════ */

  function boot() {
    initMobileMenu();
    initCartDrawer();
    initModals();
    initTabs();
    initToast();
    initAccordion();
    initDropdown();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', boot);
  } else {
    boot();
  }

  /* ═══════════════════════════════════════
     Public API
     ═══════════════════════════════════════ */

  window.Epure = window.Epure || {};
  window.Epure.showToast = showToast;
  window.Epure.dismissToast = dismissToast;

})();
