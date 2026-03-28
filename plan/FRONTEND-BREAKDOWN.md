# Frontend Breakdown

Flock is a modular e-commerce operations platform. The frontend is a single Next.js application (App Router) communicating with the Go API via JWT-authenticated REST endpoints. Below is the page/feature inventory, grouped by domain module.

---

## 1 — Authentication & Layout

| Page/Feature | Route | Backend Endpoints | Notes |
|---|---|---|---|
| Login | `/login` | External OIDC/Auth0 | Redirect to IdP; callback stores JWT |
| Layout shell | `(app)/layout` | `GET /check-auth` | Sidebar + header; guarded by `Require` |
| Permission gate | — | `GET /users/malformation` | Show warnings for cross-domain violations |

---

## 2 — Contacts

| Page/Feature | Route | Backend Endpoints | Notes |
|---|---|---|---|
| Contact list | `/contacts` | `GET /contacts` | Paginated table. Filters: name, email, city |
| Contact detail | `/contacts/[id]` | `GET /contacts/:id` | Read-only overview |
| Create contact | `/contacts/new` | `POST /contacts` | Form: name, email, phone, address, city, metadata |
| Edit contact | `/contacts/[id]/edit` | `PATCH /contacts/:id` | Same form, prefilled |
| Delete contact | confirm dialog | `DELETE /contacts/:id` | Destructive, requires confirmation |
| Membership status | panel in detail | `GET /membership/status/:contactId` | Shows opt-in/opt-out per channel |
| RFM score | panel in detail | `GET /analytics/rfm/contacts/:contactId/score` | R/F/M individual scores |
| Affinity profile | panel in detail | `GET /analytics/affinity/contacts/:contactId` | Tag + category + variation affinities |
| Recommendations | panel in detail | `GET /analytics/recommendations/contacts/:contactId` | Product carousel with seed param |

---

## 3 — Products

| Page/Feature | Route | Backend Endpoints | Notes |
|---|---|---|---|
| Product list | `/products` | `GET /products` | Table with SKU, name, price, status |
| Product detail | `/products/[id]` | `GET /products/:id` | Product info + variations |
| Create product | `/products/new` | `POST /products` | Form: name, SKU, description, tags, categories, price |
| Edit product | `/products/[id]/edit` | `PATCH /products/:id` | Same form, prefilled |
| Delete product | confirm dialog | `DELETE /products/:id` | |
| Find by SKU | search bar | `GET /products/sku/:sku` | Quick-lookup flow |
| Variation CRUD | sub-tabs | `CRUD /variations` | Inline table for product variations |
| Category tree | `/categories` | `GET /categories`, `GET /categories/:id/children` | Tree view with expand/collapse |
| Create/edit category | dialog | `POST/PATCH /categories/:id` | |
| Tag management | `/tags` | `GET /tags`, `DELETE /tags/:name` | List + delete |
| Tag correlations | `/tags/correlations` | `CRUD /tags/correlations` | Table: source to target with strength |

---

## 4 — Orders

| Page/Feature | Route | Backend Endpoints | Notes |
|---|---|---|---|
| Order list | `/orders` | `GET /orders` | Table: order#, contact, total, status, date |
| Order detail | `/orders/[id]` | `GET /orders/:id` | Line items, status timeline, comments |
| Create order | `/orders/new` | `POST /orders` | Product picker, contact selector |
| Edit order | `/orders/[id]/edit` | `PATCH /orders/:id` | |
| Status change | inline action | `PATCH /orders/:id/status` | Dropdown or modal with status options |
| Comments thread | panel in detail | `POST/PATCH/DELETE /orders/:id/comments/:commentId` | Inline thread |

---

## 5 — Assets

| Page/Feature | Route | Backend Endpoints | Notes |
|---|---|---|---|
| Asset browser | `/assets` | `GET /assets`, `GET /assets/folders/tree` | Two-pane: folder tree + asset grid |
| Upload | dialog | `POST /assets` | Multipart upload |
| Asset detail | drawer | `GET /assets/:id` | Preview + metadata |
| Edit asset | drawer | `PATCH /assets/:id` | Rename, move folder |
| Folder CRUD | context menu | `CRUD /assets/folders` | |
| JPG worker | action button | `POST /assets/workers/jpg/run` | Trigger bulk transcode |

---

## 6 — Marketing: Campaigns

| Page/Feature | Route | Backend Endpoints | Notes |
|---|---|---|---|
| Campaign list | `/campaigns` | `GET /campaigns` | Table: name, segment, status, sent date |
| Campaign detail | `/campaigns/[id]` | `GET /campaigns/:id` | Overview, deliveries |
| Create campaign | `/campaigns/new` | `POST /campaigns` | Form: name, segment, template, subject, from |
| Edit campaign | `/campaigns/[id]/edit` | `PATCH /campaigns/:id` | |
| Delete campaign | confirm dialog | `DELETE /campaigns/:id` | Only PLANNED status |
| Send campaign | action button | `POST /campaigns/:id/send` | Confirm dialog |
| Test send | action button | `POST /campaigns/:id/test` | Prompt for test email |
| Delivery log | sub-tab | `GET /campaigns/:id/deliveries` | Table: contact, status, timestamp |

---

## 7 — Marketing: Segments

| Page/Feature | Route | Backend Endpoints | Notes |
|---|---|---|---|
| Segment list | `/segments` | `GET /segments` | Table: name, slug, filter count |
| Segment detail | `/segments/[id]` | `GET /segments/:id` | Filter list, parent info |
| Create segment | `/segments/new` | `POST /segments` | Name, slug, channel, parent dropdown, filter builder |
| Edit segment | `/segments/[id]/edit` | `PATCH /segments/:id` | |
| Delete segment | confirm dialog | `DELETE /segments/:id` | |
| Preview count | inline | `POST /segments/preview/count` | Live counter while editing filters |
| Resolve contacts | action | `POST /segments/:id/resolve` | Modal with contact list |
| Count | badge | `GET /segments/:id/count` | Shown as badge on list/detail |

### Filter Builder Component

The segment filter builder is the most complex UI component. Each filter type has its own form:

| Filter Type | UI Controls |
|---|---|
| `email_opt_in` | Checkbox |
| `city` / `city_code_in` | Multi-select city picker |
| `min_total_spend` | Currency input |
| `purchased_sku` | Multi-text (SKU list) |
| `order_recency` / `no_order_recency` | Number input (days) |
| `category` | Category tree picker |
| `top_spenders` | Number input (limit or percentage) |
| `first_purchase_only` | Toggle |
| `subscribed_no_buy` | Toggle |
| `opt_in_status` | Channel dropdown + status dropdown |
| `metadata` | Key text + value text |
| `order_status` | Multi-select status picker |
| `rfm_group` | Dropdown from `GET /analytics/rfm/groups` |
| `rfm_score` | Min/max number inputs |
| `rfm_range` | R/F/M min/max number inputs |
| `min_order_count` | Number input |
| `tag_affinity` | Tag picker + percentage slider + related tags |
| `category_affinity` | Category picker + percentage slider |
| `variation_affinity` | Name + value text + percentage slider |
| `mail_open_rate` | Min/max percentage sliders (0-100) |

Each filter row has an **exclude** toggle and a **remove** button. Filters can be added via a dropdown selector.

---

## 8 — Marketing: RFM

| Page/Feature | Route | Backend Endpoints | Notes |
|---|---|---|---|
| RFM overview | `/analytics/rfm` | `GET /analytics/rfm/bands`, `GET /analytics/rfm/groups` | Band config table + group list |
| Band config editor | inline | `PUT /analytics/rfm/bands/:dimension` | Edit thresholds for R, F, M |
| Group CRUD | dialog | `CRUD /analytics/rfm/groups` | Name, slug, conditions (R/F/M min/max) |
| Contact score | panel | `GET /analytics/rfm/contacts/:contactId/score` | Used in contact detail |
| Batch score | action | `POST /analytics/rfm/contacts/score-batch` | Upload contact IDs, get scores |
| Refresh | action button | `POST /analytics/rfm/refresh` | Trigger MV refresh |

---

## 9 — Marketing: Email

| Page/Feature | Route | Backend Endpoints | Notes |
|---|---|---|---|
| Delivery list | `/email/deliveries` | `GET /email/deliveries?email=` | Table: subject, status, timestamp |
| Delivery detail | `/email/deliveries/[id]` | `GET /email/deliveries/:id` | Full delivery info + status history |
| Send email | dialog | `POST /email/send` | Manual send (debugging) |

---

## 10 — Shipping

| Page/Feature | Route | Backend Endpoints | Notes |
|---|---|---|---|
| Quotation request | `/shipping/quotations/new` | `POST /shipping/quotations` | Origin/destination/weight form |
| Quotation list | `/shipping/quotations` | `GET /shipping/quotations` | Table: carrier, price, transit time |
| Shipping mark list | `/shipping/marks` | `GET /shipping/marks` | Table: tracking#, status, carrier |
| Mark detail | `/shipping/marks/[id]` | `GET /shipping/marks/:id` | Full details |
| Create mark | dialog | `POST /shipping/marks` | Quote ID, order ID, contact info |
| Void mark | action | `PATCH /shipping/marks/:id/void` | |
| Batch management | `/shipping/batches` | `GET /shipping/batches` | Table: id, status, mark count |
| Batch detail | `/shipping/batches/[id]` | `GET /shipping/batches/:id` | Mark list + add/remove actions |
| Close batch | action | `PATCH /shipping/batches/:id/close` | |
| Download manifest | action | `GET /shipping/batches/:id/manifest-document` | PDF download |
| Tracking lookup | search | `GET /shipping/tracking/:trackingNumber` | |
| Carrier list | reference | `GET /shipping/carriers` | |
| Order dispatch | panel in order detail | `GET /shipping/orders/:orderID/dispatch` | |

---

## 11 — Sync & Integrations

| Page/Feature | Route | Backend Endpoints | Notes |
|---|---|---|---|
| Sync dashboard | `/sync` | `GET /syncrecord/stats` | KPIs: total runs, last sync, errors |
| Sync run list | `/sync/runs` | `GET /syncrecord/runs` | Table: start, end, module, status |
| Sync run detail | `/sync/runs/[id]` | `GET /syncrecord/runs/:id` | Items processed, errors |
| WooCommerce sync | action buttons | `POST /woo/sync/contacts`, `POST /woo/sync/orders` | Manual trigger |
| Falabella sync | `/falabella` | `POST /falabella/sync/products`, feed/execution status | |
| Falabella brands | reference | `GET /falabella/brands` | |
| Feed status | detail | `GET /falabella/sync/status/feed/:feedId` | |

---

## 12 — Analytics

| Page/Feature | Route | Backend Endpoints | Notes |
|---|---|---|---|
| Analytics status | `/analytics` | `GET /analytics/status` | ClickHouse connection health |
| Seed data | action | `POST /analytics/seed` | Backfill ClickHouse from MySQL |
| Affinity refresh | action | `POST /analytics/affinity/refresh` | Refresh MVs |

---

## Shared Components

| Component | Used By | Notes |
|---|---|---|
| Paginated table | All list pages | Page, limit, total, sort controls |
| Permission gate | All pages | Hide/show actions based on JWT scopes |
| Confirm dialog | Delete actions | Standard destructive-action confirmation |
| Sidebar navigation | Layout | Module-grouped menu items |
| Contact picker | Orders, Campaigns | Search-as-you-type for contacts |
| Product picker | Orders, Campaigns | Search-as-you-type for products |
| Category tree | Products, Segments | Recursive tree with checkboxes |
| Date range picker | Orders, Sync runs | Standard date filter |
| Toast notifications | All mutations | Success/error feedback |

---

## Permission-to-Page Mapping

| Permission | Pages |
|---|---|
| `contact:view` | Contact list, contact detail |
| `contact:manage` | Contact create/edit/delete |
| `product:view` | Product list, detail, categories, falabella brands |
| `product:edit` | Product create/edit |
| `product:manage` | Product delete, variations, categories, falabella sync |
| `product:tags` | Tag list |
| `order:view` | Order list, detail |
| `order:manage` | Order create/edit |
| `order:triage` | Order status change, comments |
| `order:sync` | WooCommerce order sync |
| `contact:sync` | WooCommerce contact sync |
| `assets:view` | Asset browser |
| `assets:manage` | Upload, edit, delete assets/folders |
| `shipping:quotations` | Quotation + mark list/detail, carriers |
| `shipping:generate` | Create marks, batches, manifest |
| `shipping:manage` | Void marks |
| `marketing:manage` | Campaigns, segments, email, RFM, analytics, membership, tags |

---

## Technical Stack Recommendation

| Concern | Choice | Rationale |
|---|---|---|
| Framework | Next.js 15 (App Router) | SSR + client components, file-based routing |
| State | TanStack Query | Server-state cache, optimistic updates |
| UI | shadcn/ui + Tailwind | Composable, accessible, consistent |
| Forms | react-hook-form + zod | Validation tied to API schemas |
| Tables | TanStack Table | Sorting, filtering, pagination |
| Auth | next-auth or Auth.js | JWT passthrough to API |
| Charts | Recharts | RFM visualizations, sync stats |
| Rich text | Monaco or CodeMirror | Email template editor (Go templates) |

---

## Estimated Page Count

| Module | Pages | Complexity |
|---|---|---|
| Auth + Layout | 2 | Low |
| Contacts | 4 | Medium (affinity/RFM panels) |
| Products | 6 | Medium (category tree, variations) |
| Orders | 4 | Medium (status lifecycle, comments) |
| Assets | 2 | Medium (file browser UX) |
| Marketing: Campaigns | 4 | High (template editor, delivery log) |
| Marketing: Segments | 3 | High (filter builder is complex) |
| Marketing: RFM | 2 | Medium |
| Marketing: Email | 2 | Low |
| Shipping | 6 | Medium (batches, manifest) |
| Sync & Integrations | 4 | Low-Medium |
| Analytics | 1 | Low |
| **Total** | **~40 pages** | |
