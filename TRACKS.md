## Objectif
Backend Go + Unpoly pour la boutique Épure — architecture hexagonale, SQLite, html/template, stdlib.

## Contraintes
- Go 1.26+, stdlib uniquement sauf `modernc.org/sqlite` + `golang.org/x/crypto/bcrypt`
- Pas de framework web, pas d'ORM
- Tout le front-end sous `web/`, rien ailleurs
- CSRF via `http.CrossOriginProtection` (pas de tokens)
- Unpoly v3.14.x vendored dans `web/static/vendor/`

## Décisions actées
- ID génération: package `internal/core/domain/id` (pas de `crypto/uuid` dispo avant Go 1.27)
- `RequireAdmin` redirige vers `/admin/login` (pas 403) pour les non-Unpoly
- SessionStore hache le token en interne — la middleware passe le token brut
- `wrapData()` dans le renderer fournit des safe defaults pour les layouts
- CSP `unsafe-inline` pour scripts (nécessaire pour Unpoly)

## Tracks

| Tâche | Agent | Branche | Statut | Bloqué par |
|---|---|---|---|---|
| core-domain-ports | hermes | — | done | — |
| sqlite-adapters | hermes | — | done | core-domain-ports |
| core-services | hermes | — | done | core-domain-ports |
| templates-conversion | hermes | — | done | — |
| unpoly-middleware | hermes | — | done | core-domain-ports |
| http-handlers | hermes | — | done | sqlite-adapters, templates-conversion, unpoly-middleware |
| main-wiring | hermes | — | done | http-handlers |
| security-audit | hermes | — | done | main-wiring |
| gateway-memory-diag | hermes | — | completed | — |
| disk-usage-audit | hermes | — | completed | — |

### upload-product-images
**À livrer** : Upload d'image dans le formulaire produit admin (MaxBytesReader, DetectContentType, renommage)
**Critères d'acceptation** :
- [ ] POST /admin/products accepte multipart/form-data avec image
- [ ] Limite 5 Mo, types jpeg/png/webp uniquement
- [ ] Nom régénéré aléatoirement
- [ ] Erreur propre si mauvais type ou trop gros
