# Epic 2 — Core Ledger

Goal: deliver the complete entry-management workflow against PostgreSQL. Each write
is scoped to the current owner and every money rule is enforced in both the domain
and database where practical.

## LS-007 — Create and list categories

**As a** ledger owner  
**I want** to create and list my categories  
**So that** I can organize entries using my own vocabulary

**Dependencies:** LS-004, LS-005

### Acceptance criteria

- [ ] `POST /api/v1/ledger/categories` creates an active category owned by the
      current user and returns `201` with its resource representation.
- [ ] A category has an ID, trimmed non-empty name, optional group/type, status, and
      timestamps.
- [ ] Active category names are unique per owner using case-insensitive comparison;
      duplicates return `409 CONFLICT`.
- [ ] `GET /api/v1/ledger/categories` returns only the current owner's categories,
      ordered predictably, with an option to include archived records.
- [ ] Input failures use the standard error envelope and stable field-level details.
- [ ] Repository integration tests prove owner isolation and database constraints.

---

## LS-008 — Edit and archive categories

**As a** ledger owner  
**I want** to rename or archive a category  
**So that** my classification evolves without breaking history

**Dependencies:** LS-007

### Acceptance criteria

- [ ] `PUT /api/v1/ledger/categories/{id}` updates an owned category using the same
      naming validation and uniqueness rules as creation.
- [ ] `DELETE /api/v1/ledger/categories/{id}` archives rather than physically deletes
      the category and is idempotent.
- [ ] Archived categories remain linked to historical entries but cannot be selected
      for newly created or recategorized entries.
- [ ] Accessing a missing or another owner's category returns `404` without revealing
      whether it exists for another owner.
- [ ] Concurrent conflicting updates produce a deterministic conflict or a documented
      last-write-wins result; behavior is covered by tests.

---

## LS-009 — Create a ledger entry

**As a** ledger owner  
**I want** to record income or an expense  
**So that** my ledger captures the financial fact when it occurs

**Dependencies:** LS-002, LS-004, LS-007

### Acceptance criteria

- [ ] `POST /api/v1/ledger/entries` accepts date, description, positive amount,
      currency (`BRL` or `USD`), type (`income` or `expense`), category ID, optional
      subcategory, institution, and card invoice month.
- [ ] Decimal money is parsed without binary floating-point and stored as
      `NUMERIC(14,2)`; zero, negative, over-precision, and overflow values are rejected.
- [ ] The selected category exists, belongs to the current owner, and is active.
- [ ] The service never converts currencies and never persists a negative amount.
- [ ] Creation and category validation occur in one transaction where required to
      prevent invalid references.
- [ ] Success returns `201`, a `Location` header, and the canonical representation.
- [ ] Validation errors return `422`; malformed JSON returns `400`; unsupported
      content types return `415`.
- [ ] Unit tests cover domain rules and integration/HTTP tests cover persistence,
      owner isolation, constraints, and error responses.

---

## LS-010 — Retrieve an entry by ID

**As a** ledger owner  
**I want** to retrieve one entry  
**So that** I can inspect it before editing or displaying its details

**Dependencies:** LS-009

### Acceptance criteria

- [ ] `GET /api/v1/ledger/entries/{id}` returns the complete canonical entry.
- [ ] Invalid UUID syntax returns `400` and an absent or non-owned ID returns `404`.
- [ ] Archived category metadata remains readable through historical entries.
- [ ] The query always includes owner scope and does not rely only on filtering in the
      HTTP layer.

---

## LS-011 — List and filter ledger entries

**As a** ledger owner  
**I want** to browse and filter my entries  
**So that** I can investigate where my money came from and went

**Dependencies:** LS-009

### Acceptance criteria

- [ ] `GET /api/v1/ledger/entries` supports combinable filters for month/date range,
      category, institution, currency, and type.
- [ ] Results are ordered by entry date descending and then ID descending for stable
      ordering.
- [ ] Cursor pagination supports a bounded configurable page size, returns a next
      cursor only when more data exists, and rejects invalid cursors.
- [ ] Cursors are opaque to clients and preserve stable traversal when multiple
      records share a date.
- [ ] Soft-deleted entries are excluded from all ordinary results.
- [ ] Queries are parameterized and an integration test verifies expected query plans
      use the owner/date index for the primary access path.
- [ ] Empty results return `200` with an empty collection, not `404`.

---

## LS-012 — Edit a ledger entry

**As a** ledger owner  
**I want** to correct an existing entry  
**So that** mistakes do not remain in my financial history

**Dependencies:** LS-009, LS-010

### Acceptance criteria

- [ ] `PUT /api/v1/ledger/entries/{id}` replaces all editable fields and applies the
      same validation as creation.
- [ ] An entry may be edited regardless of its age; monthly closure is a report, not
      a write lock.
- [ ] The new category must be active and owned by the same user.
- [ ] The response returns the updated canonical resource and an `updated_at` value.
- [ ] Missing, deleted, or non-owned entries return `404` without information leakage.
- [ ] Concurrent updates cannot silently overwrite a newer version: an explicit
      version/ETag precondition or equivalent optimistic-locking strategy is tested.

---

## LS-013 — Soft-delete a ledger entry

**As a** ledger owner  
**I want** to remove an incorrect or duplicate entry  
**So that** it no longer affects my ledger totals while remaining auditable

**Dependencies:** LS-009, LS-010

### Acceptance criteria

- [ ] `DELETE /api/v1/ledger/entries/{id}` sets `deleted_at` and does not physically
      remove the row.
- [ ] Repeating deletion is idempotent and returns the documented success status.
- [ ] Deleted entries do not appear in retrieval, listing, summaries, or comparisons.
- [ ] Missing and non-owned IDs return `404` without revealing cross-owner data.
- [ ] The operation records enough metadata for a future restore/audit capability;
      implementing restore is explicitly out of scope for v1.

