# Epic 3 — Ledger Insights

Goal: replace spreadsheet aggregation with correct, queryable views. All results are
calculated from non-deleted entries and never combine BRL with USD.

## LS-014 — View a monthly summary

**As a** ledger owner  
**I want** monthly income, expense, and net totals  
**So that** I can understand my cash flow without manually summing entries

**Dependencies:** LS-011, LS-013

### Acceptance criteria

- [ ] `GET /api/v1/ledger/monthly-summary?month=YYYY-MM` returns income, expense, and
      net totals grouped by currency.
- [ ] Net is calculated as income minus expense while persisted amounts remain
      positive.
- [ ] Missing months return zero totals and empty groups with `200`.
- [ ] The month is interpreted as a calendar month using date semantics, not server
      timezone-dependent timestamp boundaries.
- [ ] Deleted entries and entries owned by another user are excluded.
- [ ] Results are calculated on read from PostgreSQL; no summary table or cache is
      introduced in v1.
- [ ] Tests cover income-only, expense-only, mixed, empty, BRL/USD, rounding, and
      month-boundary cases.

---

## LS-015 — Break down a month by category and institution

**As a** ledger owner  
**I want** to see monthly totals grouped by category or institution  
**So that** I can identify where I spend money and which accounts I use

**Dependencies:** LS-014

### Acceptance criteria

- [ ] The monthly-summary endpoint or a clearly documented subordinate resource can
      group totals by category and, optionally, institution.
- [ ] Income and expense totals remain distinct and each currency is a separate
      result dimension.
- [ ] Archived categories retain their historical name in results.
- [ ] Blank institutions are represented by a stable `unassigned` bucket rather than
      silently omitted.
- [ ] Result ordering is deterministic and documented.
- [ ] The implementation avoids N+1 queries and is covered by a real PostgreSQL
      integration test.

---

## LS-016 — Compare two months

**As a** ledger owner reviewing history  
**I want** to compare category spending across two months  
**So that** I can identify meaningful increases and opportunities to reduce expenses

**Dependencies:** LS-015

### Acceptance criteria

- [ ] `GET /api/v1/ledger/monthly-comparison?base=YYYY-MM&compare=YYYY-MM` returns
      expense totals per category for both months, by currency.
- [ ] Each result includes absolute change and percentage change when mathematically
      defined.
- [ ] A category present in only one month appears with zero for the other month.
- [ ] When the base value is zero, percentage change is `null` with documented
      semantics; the API never emits `NaN` or infinity.
- [ ] Comparing a month with itself is either rejected with a stable validation error
      or returns zero change, with the selected behavior documented and tested.
- [ ] Deleted and non-owned entries are excluded, and BRL/USD are never combined.

