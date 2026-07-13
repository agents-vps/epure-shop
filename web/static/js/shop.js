/**
 * Shop.js — Vanilla JS for boutique Épure (client-side)
 *
 * Covers:
 *   - Simulated cart: add / remove / update quantity, localStorage persistence
 *   - Product filters simulation: price range, sort
 *   - Product gallery: thumbnail switching
 *   - Favorites: toggle with localStorage persistence
 *   - Cart badge counter: live update
 *   - Promo code: simulated discount application
 *
 * All DOM targeting uses data-* attributes exclusively.
 * Zero framework dependencies.
 *
 * @version 1.0.0
 */

(function () {
  'use strict';

  /* ───────────────────────────────────────────
     STORAGE HELPERS
     ─────────────────────────────────────────── */

  var STORAGE_CART = 'epure_cart';
  var STORAGE_FAVORITES = 'epure_favorites';

  /**
   * Retrieve cart from localStorage.
   * @returns {Array} Array of cart items: { id, name, price, image, quantity }
   */
  function getCart() {
    try {
      return JSON.parse(localStorage.getItem(STORAGE_CART)) || [];
    } catch (e) {
      return [];
    }
  }

  /**
   * Persist cart to localStorage.
   * @param {Array} cart
   */
  function setCart(cart) {
    try {
      localStorage.setItem(STORAGE_CART, JSON.stringify(cart));
    } catch (e) {
      // Storage full or unavailable — silently degrade
    }
  }

  /**
   * Retrieve favorites from localStorage.
   * @returns {Array} Array of product IDs (strings).
   */
  function getFavorites() {
    try {
      return JSON.parse(localStorage.getItem(STORAGE_FAVORITES)) || [];
    } catch (e) {
      return [];
    }
  }

  /**
   * Persist favorites to localStorage.
   * @param {Array} favorites
   */
  function setFavorites(favorites) {
    try {
      localStorage.setItem(STORAGE_FAVORITES, JSON.stringify(favorites));
    } catch (e) {
      // Storage full or unavailable
    }
  }

  /**
   * Format a price (number) to a localized currency string.
   * @param {number} cents — price in cents
   * @returns {string}
   */
  function formatPrice(cents) {
    return new Intl.NumberFormat('fr-FR', {
      style: 'currency',
      currency: 'EUR',
    }).format(cents / 100);
  }

  /**
   * Parse a price from a DOM element's data attribute or text content.
   * Expects raw number in cents stored in data-price (or falls back to text).
   * @param {HTMLElement} el
   * @returns {number} price in cents
   */
  function parsePrice(el) {
    var raw = el.getAttribute('data-price');
    if (raw !== null) return parseInt(raw, 10) || 0;
    // Fallback: strip non-digits from text
    return parseInt(el.textContent.replace(/[^\d]/g, ''), 10) || 0;
  }

  /* ═══════════════════════════════════════════
     CART MODULE
     ═══════════════════════════════════════════ */

  var cart = getCart();
  var PROMO_DISCOUNTS = {
    'BIENVENUE10': { type: 'percent', value: 10, label: '–10 % (Bienvenue)' },
    'LIVRAISON': { type: 'fixed', value: 499, label: 'Livraison offerte' },
    'ETE25': { type: 'percent', value: 25, label: '–25 % (Soldes d\'été)' },
  };
  var activePromo = null; // { code, discount }

  /**
   * Find a cart item by product ID.
   * @param {string} id
   * @returns {object|undefined}
   */
  function findCartItem(id) {
    return cart.find(function (item) {
      return item.id === id;
    });
  }

  /**
   * Add a product to the cart, or increase quantity if already present.
   * @param {string} id
   * @param {string} name
   * @param {number} price  — in cents
   * @param {string} image  — URL
   * @param {number} [qty=1]
   */
  function addToCart(id, name, price, image, qty) {
    qty = qty || 1;
    var existing = findCartItem(id);
    if (existing) {
      existing.quantity += qty;
    } else {
      cart.push({
        id: id,
        name: name,
        price: price,
        image: image,
        quantity: qty,
      });
    }
    setCart(cart);
    updateCartCount();
    renderCartItems();
    updateCartTotals();
    flashCartBadge();
    showToast('success', 'Ajouté au panier', name);
  }

  /**
   * Remove an item from the cart.
   * @param {string} id
   */
  function removeFromCart(id) {
    var item = findCartItem(id);
    cart = cart.filter(function (item) {
      return item.id !== id;
    });
    setCart(cart);
    updateCartCount();
    renderCartItems();
    updateCartTotals();
    if (item) {
      showToast('info', 'Retiré du panier', item.name);
    }
  }

  /**
   * Update quantity for a cart item. Remove if quantity <= 0.
   * @param {string} id
   * @param {number} quantity
   */
  function updateQuantity(id, quantity) {
    quantity = Math.max(0, parseInt(quantity, 10) || 0);
    if (quantity === 0) {
      removeFromCart(id);
      return;
    }
    var item = findCartItem(id);
    if (item) {
      item.quantity = quantity;
      setCart(cart);
      updateCartCount();
      renderCartItems();
      updateCartTotals();
    }
  }

  /**
   * Compute cart subtotal (in cents).
   * @returns {number}
   */
  function cartSubtotal() {
    return cart.reduce(function (sum, item) {
      return sum + item.price * item.quantity;
    }, 0);
  }

  /**
   * Compute discount (in cents) based on active promo.
   * @returns {number}
   */
  function computeDiscount(subtotal) {
    if (!activePromo) return 0;
    var d = activePromo.discount;
    if (d.type === 'percent') {
      return Math.round(subtotal * (d.value / 100));
    }
    if (d.type === 'fixed') {
      return Math.min(d.value, subtotal); // never go negative
    }
    return 0;
  }

  /**
   * Get cart total item count.
   * @returns {number}
   */
  function cartItemCount() {
    return cart.reduce(function (sum, item) {
      return sum + item.quantity;
    }, 0);
  }

  /**
   * Update all [data-component="cart-count"] badges.
   */
  function updateCartCount() {
    var badges = document.querySelectorAll('[data-component="cart-count"]');
    var count = cartItemCount();
    badges.forEach(function (badge) {
      badge.textContent = count;
      // Toggle visibility class (may be .badge)
      if (count > 0) {
        badge.removeAttribute('hidden');
        badge.style.display = '';
      } else {
        badge.setAttribute('hidden', '');
        badge.style.display = 'none';
      }
    });
    // Also update cart-count on any aria-label for screen readers
    badges.forEach(function (badge) {
      if (count > 0) {
        badge.setAttribute('aria-label', count + ' article' + (count > 1 ? 's' : '') + ' dans le panier');
      }
    });
  }

  /**
   * Brief pulse animation on the cart badge.
   */
  function flashCartBadge() {
    var badges = document.querySelectorAll('[data-component="cart-count"]');
    badges.forEach(function (badge) {
      badge.style.transition = 'transform 150ms cubic-bezier(0.34, 1.56, 0.64, 1)';
      badge.style.transform = 'scale(1.35)';
      setTimeout(function () {
        badge.style.transform = 'scale(1)';
      }, 180);
    });
  }

  /**
   * Render cart items in the cart page / drawer.
   * Targets [data-component="cart-item"] containers.
   */
  function renderCartItems() {
    var containers = document.querySelectorAll('[data-component="cart-item"]');
    containers.forEach(function (container) {
      var itemId = container.getAttribute('data-product-id');
      if (!itemId) return;
      var item = findCartItem(itemId);
      if (!item) {
        // Item not in cart — hide the row
        container.setAttribute('data-state', 'removed');
        container.style.display = 'none';
        return;
      }
      container.removeAttribute('data-state');
      container.style.display = '';

      // Quantity input
      var qtyInput = container.querySelector('[data-quantity]');
      if (qtyInput) {
        qtyInput.value = item.quantity;
      }

      // Quantity display
      var qtyDisplay = container.querySelector('[data-cart-quantity]');
      if (qtyDisplay) {
        qtyDisplay.textContent = item.quantity;
      }

      // Line total
      var lineTotal = container.querySelector('[data-cart-line-total]');
      if (lineTotal) {
        lineTotal.textContent = formatPrice(item.price * item.quantity);
        lineTotal.setAttribute('data-price', item.price * item.quantity);
      }
    });

    // Also handle any empty-state visibility
    var emptyStates = document.querySelectorAll('[data-component="cart-empty"]');
    var cartSections = document.querySelectorAll('[data-component="cart-content"]');
    var isEmpty = cart.length === 0;
    emptyStates.forEach(function (el) {
      el.style.display = isEmpty ? '' : 'none';
      if (isEmpty) el.removeAttribute('hidden');
      else el.setAttribute('hidden', '');
    });
    cartSections.forEach(function (el) {
      el.style.display = isEmpty ? 'none' : '';
      if (isEmpty) el.setAttribute('hidden', '');
      else el.removeAttribute('hidden');
    });
  }

  /**
   * Update cart totals (subtotal, discount, shipping, total).
   * Targets elements with [data-cart-subtotal], [data-cart-discount],
   * [data-cart-shipping], [data-cart-total].
   */
  function updateCartTotals() {
    var subtotal = cartSubtotal();
    var discount = computeDiscount(subtotal);
    // Free shipping above 75 € (7500 cents), otherwise 4.90 €
    var shipping = subtotal >= 7500 ? 0 : (subtotal > 0 ? 490 : 0);
    var total = subtotal - discount + shipping;

    var subEls = document.querySelectorAll('[data-cart-subtotal]');
    var discEls = document.querySelectorAll('[data-cart-discount]');
    var discLabelEls = document.querySelectorAll('[data-cart-discount-label]');
    var shipEls = document.querySelectorAll('[data-cart-shipping]');
    var totalEls = document.querySelectorAll('[data-cart-total]');

    subEls.forEach(function (el) {
      el.textContent = formatPrice(subtotal);
      el.setAttribute('data-price', subtotal);
    });
    discEls.forEach(function (el) {
      el.textContent = discount > 0 ? '−' + formatPrice(discount) : formatPrice(0);
      el.setAttribute('data-price', discount);
    });
    discLabelEls.forEach(function (el) {
      el.textContent = activePromo ? activePromo.discount.label : '';
    });
    shipEls.forEach(function (el) {
      if (shipping === 0 && subtotal > 0) {
        el.textContent = 'Offerte';
      } else {
        el.textContent = formatPrice(shipping);
      }
      el.setAttribute('data-price', shipping);
    });
    totalEls.forEach(function (el) {
      el.textContent = formatPrice(total);
      el.setAttribute('data-price', total);
    });
  }

  /**
   * Initialize cart: bind events, hydrate from localStorage.
   */
  function initCart() {
    // Update badge on load
    updateCartCount();

    /* ── Add to cart buttons ── */
    document.addEventListener('click', function (e) {
      var btn = e.target.closest('[data-component="add-to-cart"]');
      if (!btn) return;
      if (btn.disabled || btn.getAttribute('aria-disabled') === 'true') return;

      var productId = btn.getAttribute('data-product-id');
      var name = btn.getAttribute('data-product-name');
      var priceRaw = btn.getAttribute('data-product-price');
      var image = btn.getAttribute('data-product-image');
      var qtyInput = document.querySelector('[data-component="quantity-input"][data-product-id="' + productId + '"]');
      var qty = qtyInput ? parseInt(qtyInput.value, 10) || 1 : 1;

      if (!productId || !name || priceRaw === null) return;

      var price = parseInt(priceRaw, 10) || 0;

      addToCart(productId, name, price, image, qty);

      // Brief loading state feedback
      btn.classList.add('btn--loading');
      setTimeout(function () {
        btn.classList.remove('btn--loading');
      }, 600);
    });

    /* ── Quantity steppers inside cart items ── */
    document.addEventListener('click', function (e) {
      var btn = e.target.closest('[data-action]');
      if (!btn) return;

      var cartItem = btn.closest('[data-component="cart-item"]');
      if (!cartItem) return;

      var productId = cartItem.getAttribute('data-product-id');
      if (!productId) return;

      var action = btn.getAttribute('data-action');
      var item = findCartItem(productId);
      if (!item) return;

      if (action === 'increase' || action === 'increment') {
        updateQuantity(productId, item.quantity + 1);
      } else if (action === 'decrease' || action === 'decrement') {
        updateQuantity(productId, item.quantity - 1);
      } else if (action === 'remove') {
        removeFromCart(productId);
      }
    });

    /* ── Direct quantity input changes ── */
    document.addEventListener('change', function (e) {
      var input = e.target.closest('[data-quantity]');
      if (!input) return;

      var cartItem = input.closest('[data-component="cart-item"]');
      if (!cartItem) return;

      var productId = cartItem.getAttribute('data-product-id');
      if (!productId) return;

      var qty = parseInt(input.value, 10);
      if (isNaN(qty) || qty < 1) {
        qty = 1;
        input.value = 1;
      }
      updateQuantity(productId, qty);
    });

    // Render items on load
    renderCartItems();
    updateCartTotals();
  }

  /* ═══════════════════════════════════════════
     FILTERS MODULE
     ═══════════════════════════════════════════ */

  /**
   * Initialize filters: price range and sort.
   * Targets [data-component="filters"].
   */
  function initFilters() {
    var filterContainer = document.querySelector('[data-component="filters"]');
    if (!filterContainer) return;

    // Price range inputs
    var minInput = filterContainer.querySelector('[data-filter-price-min]');
    var maxInput = filterContainer.querySelector('[data-filter-price-max]');

    // Sort selector
    var sortSelect = filterContainer.querySelector('[data-filter-sort]');

    // Product grid to filter
    var productGrid = document.querySelector('[data-component="product-grid"]');

    /**
     * Apply all active filters to the product grid.
     */
    function applyFilters() {
      if (!productGrid) return;

      var minPrice = minInput ? parseFloat(minInput.value) || 0 : 0;
      var maxPrice = maxInput ? parseFloat(maxInput.value) || Infinity : Infinity;
      var sortValue = sortSelect ? sortSelect.value : 'default';

      var cards = productGrid.querySelectorAll('[data-product-card]');
      var visibleCount = 0;

      // First pass: gather visible cards with their sort values
      var visibleCards = [];

      cards.forEach(function (card) {
        var priceEl = card.querySelector('[data-product-price]');
        var price = priceEl ? parsePrice(priceEl) : 0;
        var priceEuros = price / 100;

        if (priceEuros >= minPrice && priceEuros <= maxPrice) {
          card.removeAttribute('hidden');
          card.style.display = '';
          visibleCount++;
          visibleCards.push({ card: card, price: price });
        } else {
          card.setAttribute('hidden', '');
          card.style.display = 'none';
        }
      });

      // Sort visible cards
      if (sortValue !== 'default' && sortValue !== '') {
        visibleCards.sort(function (a, b) {
          switch (sortValue) {
            case 'price-asc':
              return a.price - b.price;
            case 'price-desc':
              return b.price - a.price;
            case 'name-asc':
              var nameA = (a.card.getAttribute('data-product-name') || '').toLowerCase();
              var nameB = (b.card.getAttribute('data-product-name') || '').toLowerCase();
              return nameA.localeCompare(nameB);
            case 'name-desc':
              var nameC = (b.card.getAttribute('data-product-name') || '').toLowerCase();
              var nameD = (a.card.getAttribute('data-product-name') || '').toLowerCase();
              return nameC.localeCompare(nameD);
            case 'newest':
              // Assume data-product-date stores ISO date; newer first
              var dateA = a.card.getAttribute('data-product-date') || '';
              var dateB = b.card.getAttribute('data-product-date') || '';
              return dateB.localeCompare(dateA);
            default:
              return 0;
          }
        });

        // Reorder in DOM
        visibleCards.forEach(function (item) {
          productGrid.appendChild(item.card);
        });
      }

      // Update result count
      var countEl = filterContainer.querySelector('[data-filter-result-count]');
      if (countEl) {
        countEl.textContent = visibleCount + ' produit' + (visibleCount !== 1 ? 's' : '');
      }

      // Show/hide empty state
      var noResults = document.querySelector('[data-component="no-results"]');
      if (noResults) {
        noResults.style.display = visibleCount === 0 ? '' : 'none';
        if (visibleCount === 0) noResults.removeAttribute('hidden');
        else noResults.setAttribute('hidden', '');
      }
    }

    // Bind events
    if (minInput) minInput.addEventListener('input', applyFilters);
    if (maxInput) maxInput.addEventListener('input', applyFilters);
    if (sortSelect) sortSelect.addEventListener('change', applyFilters);

    // Apply on load (in case of preset values)
    applyFilters();
  }

  /* ═══════════════════════════════════════════
     PRODUCT GALLERY MODULE
     ═══════════════════════════════════════════ */

  /**
   * Initialize product gallery: thumbnail → main image switching.
   * Targets [data-component="product-gallery"].
   */
  function initGallery() {
    var galleries = document.querySelectorAll('[data-component="product-gallery"]');

    galleries.forEach(function (gallery) {
      var mainImage = gallery.querySelector('[data-gallery-main]');
      var thumbnails = gallery.querySelectorAll('[data-gallery-thumbnail]');

      if (!mainImage || thumbnails.length === 0) return;

      /**
       * Switch main image to the given source and mark active thumbnail.
       * @param {string} src
       * @param {HTMLElement} activeThumb
       */
      function switchImage(src, activeThumb) {
        if (!mainImage) return;

        // Fade transition effect
        mainImage.style.opacity = '0';
        mainImage.style.transition = 'opacity 150ms ease-out';

        setTimeout(function () {
          mainImage.src = src;
          mainImage.alt = activeThumb ? (activeThumb.getAttribute('alt') || '') : '';
          mainImage.style.opacity = '1';
        }, 150);

        // Update active thumbnail state
        thumbnails.forEach(function (thumb) {
          thumb.setAttribute('aria-current', thumb === activeThumb ? 'true' : 'false');
          thumb.classList.toggle('product-gallery__thumb--active', thumb === activeThumb);
        });
      }

      // Click handler
      gallery.addEventListener('click', function (e) {
        var thumb = e.target.closest('[data-gallery-thumbnail]');
        if (!thumb) return;

        var fullSrc = thumb.getAttribute('data-gallery-full');
        var src = fullSrc || thumb.src;
        if (!src) return;

        switchImage(src, thumb);
      });

      // Keyboard support (Enter / Space on thumbnails)
      gallery.addEventListener('keydown', function (e) {
        if (e.key === 'Enter' || e.key === ' ') {
          var thumb = e.target.closest('[data-gallery-thumbnail]');
          if (!thumb) return;
          e.preventDefault();

          var fullSrc = thumb.getAttribute('data-gallery-full');
          var src = fullSrc || thumb.src;
          if (!src) return;

          switchImage(src, thumb);
        }
      });

      // Mark first thumbnail as active on load
      if (thumbnails.length > 0) {
        thumbnails[0].setAttribute('aria-current', 'true');
        thumbnails[0].classList.add('product-gallery__thumb--active');
      }
    });
  }

  /* ═══════════════════════════════════════════
     FAVORITES MODULE
     ═══════════════════════════════════════════ */

  var favorites = getFavorites();

  /**
   * Initialize favorite toggle buttons.
   * Targets [data-component="favorite-toggle"].
   */
  function initFavorites() {
    // Hydrate visible favorite buttons on load
    updateFavoriteButtons();

    document.addEventListener('click', function (e) {
      var btn = e.target.closest('[data-component="favorite-toggle"]');
      if (!btn) return;
      if (btn.disabled) return;

      var productId = btn.getAttribute('data-product-id');
      if (!productId) return;

      toggleFavorite(productId);
      updateFavoriteButton(btn, productId);
    });
  }

  /**
   * Toggle a product in the favorites array.
   * @param {string} productId
   */
  function toggleFavorite(productId) {
    var idx = favorites.indexOf(productId);
    if (idx > -1) {
      favorites.splice(idx, 1);
    } else {
      favorites.push(productId);
    }
    setFavorites(favorites);
  }

  /**
   * Update a single favorite button's visual state.
   * Expects the button to use data-state="active" / data-state="inactive".
   * @param {HTMLElement} btn
   * @param {string} productId
   */
  function updateFavoriteButton(btn, productId) {
    if (!btn) return;

    var isFavorite = favorites.indexOf(productId) > -1;

    btn.setAttribute('data-state', isFavorite ? 'active' : 'inactive');
    btn.setAttribute('aria-pressed', isFavorite ? 'true' : 'false');

    // Toggle a CSS class for heart icon styling
    btn.classList.toggle('favorite-toggle--active', isFavorite);

    // Update accessible label
    var label = btn.getAttribute('aria-label');
    if (!label || label.indexOf('Ajouter') === 0 || label.indexOf('Retirer') === 0) {
      btn.setAttribute(
        'aria-label',
        isFavorite ? 'Retirer des favoris' : 'Ajouter aux favoris'
      );
    }
  }

  /**
   * Hydrate all favorite buttons on the page.
   */
  function updateFavoriteButtons() {
    var buttons = document.querySelectorAll('[data-component="favorite-toggle"]');
    buttons.forEach(function (btn) {
      var productId = btn.getAttribute('data-product-id');
      if (productId) {
        updateFavoriteButton(btn, productId);
      }
    });
  }

  /* ═══════════════════════════════════════════
     PROMO CODE MODULE
     ═══════════════════════════════════════════ */

  /**
   * Initialize promo code input + apply button.
   * Targets [data-component="promo-code"].
   */
  function initPromoCode() {
    var containers = document.querySelectorAll('[data-component="promo-code"]');

    containers.forEach(function (container) {
      var input = container.querySelector('[data-promo-input]');
      var applyBtn = container.querySelector('[data-promo-apply]');
      var removeBtn = container.querySelector('[data-promo-remove]');
      var feedback = container.querySelector('[data-promo-feedback]');

      if (!input) return;

      /**
       * Attempt to apply a promo code.
       */
      function applyCode() {
        var code = input.value.trim().toUpperCase();
        if (!code) {
          showFeedback('Veuillez entrer un code.', 'error');
          return;
        }

        var discount = PROMO_DISCOUNTS[code];
        if (!discount) {
          showFeedback('Code promo invalide.', 'error');
          input.setAttribute('aria-invalid', 'true');
          return;
        }

        activePromo = { code: code, discount: discount };
        showFeedback(discount.label + ' appliqué !', 'success');
        input.setAttribute('aria-invalid', 'false');
        input.value = '';
        input.disabled = true;
        if (applyBtn) applyBtn.setAttribute('hidden', '');
        if (removeBtn) removeBtn.removeAttribute('hidden');
        updateCartTotals();
        renderCartItems();
      }

      /**
       * Remove the currently active promo code.
       */
      function removeCode() {
        activePromo = null;
        input.disabled = false;
        input.value = '';
        input.setAttribute('aria-invalid', 'false');
        if (applyBtn) applyBtn.removeAttribute('hidden');
        if (removeBtn) removeBtn.setAttribute('hidden', '');
        if (feedback) feedback.textContent = '';
        updateCartTotals();
        renderCartItems();
      }

      /**
       * Display feedback message.
       * @param {string} message
       * @param {'success'|'error'} type
       */
      function showFeedback(message, type) {
        if (!feedback) return;
        feedback.textContent = message;
        feedback.setAttribute('data-state', type);
        feedback.style.display = '';
        // Auto-hide after 4s
        clearTimeout(feedback._timeout);
        feedback._timeout = setTimeout(function () {
          feedback.textContent = '';
          feedback.removeAttribute('data-state');
        }, 4000);
      }

      if (applyBtn) {
        applyBtn.addEventListener('click', function (e) {
          e.preventDefault();
          applyCode();
        });
      }

      if (removeBtn) {
        removeBtn.addEventListener('click', function (e) {
          e.preventDefault();
          removeCode();
        });
      }

      // Apply on Enter key
      input.addEventListener('keydown', function (e) {
        if (e.key === 'Enter') {
          e.preventDefault();
          applyCode();
        }
      });
    });
  }

  /* ═══════════════════════════════════════════
     TOAST NOTIFICATIONS  (minimal inline)
     ═══════════════════════════════════════════ */

  /**
   * Show a simple toast notification.
   * Expects a [data-component="toast-container"] somewhere in the DOM.
   * @param {'success'|'error'|'warning'|'info'} type
   * @param {string} title
   * @param {string} [message]
   */
  function showToast(type, title, message) {
    var container = document.querySelector('[data-component="toast-container"]');
    if (!container) return;

    var toast = document.createElement('div');
    toast.className = 'toast toast--' + type;
    toast.setAttribute('role', 'status');
    toast.setAttribute('aria-live', 'polite');
    toast.setAttribute('data-component', 'toast');

    // Icon placeholder (character-based)
    var icons = {
      success: '✓',
      error: '✕',
      warning: '⚠',
      info: 'ℹ',
    };

    toast.innerHTML =
      '<span class="toast__icon">' + (icons[type] || '') + '</span>' +
      '<div class="toast__content">' +
      '<p class="toast__title">' + escapeHtml(title) + '</p>' +
      (message ? '<p class="toast__message">' + escapeHtml(message) + '</p>' : '') +
      '</div>' +
      '<button class="toast__close" aria-label="Fermer" data-action="dismiss-toast">×</button>';

    container.appendChild(toast);

    // Dismiss on close click
    toast.querySelector('[data-action="dismiss-toast"]').addEventListener('click', function () {
      removeToast(toast);
    });

    // Auto-remove after 4 seconds
    setTimeout(function () {
      removeToast(toast);
    }, 4000);
  }

  /**
   * Animate out and remove a toast element.
   * @param {HTMLElement} toast
   */
  function removeToast(toast) {
    if (!toast || !toast.parentNode) return;
    toast.style.opacity = '0';
    toast.style.transform = 'translateX(100%)';
    toast.style.transition = 'opacity 200ms ease-in, transform 200ms ease-in';
    setTimeout(function () {
      if (toast.parentNode) {
        toast.parentNode.removeChild(toast);
      }
    }, 220);
  }

  /**
   * Escape HTML to prevent XSS in toast content.
   * @param {string} str
   * @returns {string}
   */
  function escapeHtml(str) {
    var div = document.createElement('div');
    div.appendChild(document.createTextNode(str));
    return div.innerHTML;
  }

  /* ═══════════════════════════════════════════
     BOOTSTRAP
     ═══════════════════════════════════════════ */

  /**
   * Main initialization — runs on DOMContentLoaded.
   */
  function bootstrap() {
    initCart();
    initFilters();
    initGallery();
    initFavorites();
    initPromoCode();

    // Second cart-count update (after all modules hydrated)
    updateCartCount();
  }

  // Kick off when DOM is ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', bootstrap);
  } else {
    bootstrap();
  }
})();
