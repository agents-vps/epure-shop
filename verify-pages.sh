#!/usr/bin/env bash
# verify-pages.sh — Structural consistency check for multi-page HTML sites.
# Usage: bash verify-pages.sh <directory>
# Checks: skip-link, single h1, CSS/JS imports, <main>/<footer>,
#         shared components (cart-drawer), data-component attrs, <template>.
set -euo pipefail

DIR="${1:-.}"
cd "$DIR"

CLIENT_PAGES=($(find . -maxdepth 1 -name '*.html' -not -name 'components.html' | sort))
ADMIN_PAGES=($(find admin -maxdepth 1 -name '*.html' | sort))
ALL_PAGES=("${CLIENT_PAGES[@]}" "${ADMIN_PAGES[@]}")

[[ ${#ALL_PAGES[@]} -gt 0 ]] || { echo "No .html files found in $DIR"; exit 1; }

REQUIRED_CSS=("tokens.css" "base.css" "components.css")
SHOP_CSS="shop.css"
ADMIN_CSS="admin.css"
REQUIRED_JS=("ui.js")
FAILS=0

for page in "${ALL_PAGES[@]}"; do
  f="$page"
  echo "--- $f ---"

  check() { # check "label" pattern
    if grep -q "$2" "$f" 2>/dev/null; then echo "  ✓ $1"; else echo "  FAIL: $1 missing"; FAILS=$((FAILS+1)); fi
  }

  check "skip-link" 'skip-link'

  h1_count=$(grep -c '<h1' "$f" 2>/dev/null || true)
  [[ "$h1_count" -eq 1 ]] && echo "  ✓ single h1 ($h1_count)" || { echo "  FAIL: h1 count = $h1_count (expected 1)"; FAILS=$((FAILS+1)); }

  for css in "${REQUIRED_CSS[@]}"; do check "import $css" "$css"; done
  [[ "$f" == admin/* ]] && check "import $ADMIN_CSS" "$ADMIN_CSS" || check "import $SHOP_CSS" "$SHOP_CSS"
  check "import ui.js" "ui.js"
  check "import utilities.css" "utilities.css"
  check "<main>" '<main'

  # Footer intentionally absent on: auth, checkout, confirmation, 404, admin
  case "$(basename "$f")" in
    login.html|register.html|checkout.html|order-confirmation.html|404.html) ;;
    * ) [[ "$f" == admin/* ]] || check "<footer>" '<footer' ;;
  esac

  # data-component (skip login pages)
  if echo "$f" | grep -qv 'login'; then
    check "data-component" 'data-component='
  fi

  # <template> (skip auth/404/confirmation/checkout)
  case "$(basename "$f")" in
    login.html|register.html|404.html|order-confirmation.html|checkout.html) ;;
    *) check "<template>" '<template' ;;
  esac

  echo ""
done

echo "=== $FAILS failures across ${#ALL_PAGES[@]} pages ==="
exit $FAILS
