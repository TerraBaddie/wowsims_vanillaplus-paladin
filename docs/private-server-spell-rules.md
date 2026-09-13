# Private-server spell tooltip rules

How spell tooltip *values* (damage/duration text, not names or icons) are corrected
to match the private server instead of retail Classic Wowhead. Companion to
[private-server-item-rules.md](private-server-item-rules.md), which covers items.

_Last updated: 2026-09-13._

## Background

The item pipeline already pulls stats from the private server's own DBC dump
instead of Wowhead. Spells did not get the same treatment until now: spell
*mechanics* (damage/coefficients/cooldowns) were always hand-coded in the Go sim
(`sim/<class>/*.go`), never derived from any data file, and spell *tooltip text*
(the popup shown when hovering a spell in the UI) was pulled live from Wowhead
with no private-server override at all.

This pipeline fixes the second part: the displayed tooltip *description* and
*channel-cast-time* text now come from the server's own DBC dump
(`CSV's/Spell.csv`, `CSV's/SpellDuration.csv`) when available, instead of
showing retail Wowhead's numbers. It does **not** touch spell names, icons, mana
cost, range, or requirements (still Wowhead's), and it does **not** touch the
Go sim's actual combat math — see "What this does NOT fix" below.

## Pipeline (run order)

```
python tools/gen_spell_value_overrides.py       # corrects assets/db_inputs/wowhead_spell_tooltips.csv
go run ./tools/database/gen_db -outDir=./assets -gen=db
cp -r assets/database/* dist/classic/assets/database/
GOOS=js GOARCH=wasm go build -o ./dist/classic/lib.wasm ./sim/wasm/
cp dist/classic/lib.wasm dist/classic/assets/lib.wasm
npx vite build -m development                    # only needed if ui/**/*.tsx changed
```

`tools/gen_custom_spell_tooltips.py` (older, pre-existing) still separately owns
**talent** spells: it fills in tooltips for talent spell IDs Wowhead has no data
for at all, and patches per-rank `descriptions` arrays into
`ui/core/talents/trees/*.json`. `gen_spell_value_overrides.py` explicitly excludes
talent IDs and handles everything else the sim references.

### How spell IDs are discovered

`gen_spell_value_overrides.py` unions spell IDs from:

- Every literal `SpellID: N` in `sim/**/*.go` (`core.ActionID{SpellID: ...}`).
- Every integer inside a `var FooSpellId = [N]int32{0, id1, id2, ...}` array
  literal in `sim/**/*.go` — **this is the majority of ranked spells**
  (e.g. `sim/priest/mind_flay.go`'s `MindFlaySpellId`), since they're referenced
  dynamically (`MindFlaySpellId[rank]`) rather than as a literal `SpellID:` value.
  A naive `SpellID:\s*\d+` grep alone misses almost every ranked ability.
- `database.SharedSpellsIcons` (`tools/database/overrides.go`).
- All talent spell IDs (unioned in, then subtracted back out — talents stay
  owned by the older script, see above).

### What gets corrected, and how

For each discovered spell ID that's already cached from Wowhead:

1. Resolve the description from `Spell.csv`'s `EffectBasePoints`/`EffectAmplitude`/
   `ProcChance`/`StackAmount`/`DurationIndex` columns (reverse-engineered; see the
   `COL_*` constants in `tools/gen_custom_spell_tooltips.py`), substituting
   Blizzard's `$s`/`$o`/`$d`/`$h`/`$u` tokens.
2. If it differs from the cached Wowhead description, rewrite **only** the
   `<div class="q">...</div>` block inside the cached tooltip HTML.
3. If the raw DBC text ties a `$d` (duration) token to the same effect, and the
   tooltip also has a `Channeled (N sec cast)` line, correct that line's `N` to
   match the same duration — Wowhead scrapes it as a separate field, so it can
   (and did, for Mind Flay) go stale independently of the description.
4. A spell ID with no Wowhead entry at all gets a minimal DBC-only row, same
   fallback pattern the talent script already uses.

Corrections are skipped (never applied) whenever the resolved text still
contains an unresolved `$` token or a bare `X` substitution — either means the
column reverse-engineering doesn't cover that spell/effect shape, and applying
it would silently corrupt an otherwise-correct tooltip. Also skipped: `$o`
(amount-over-duration) corrections on mana-regen text specifically (validated
against SW:Pain/Renew-style periodic damage/heal effects only — spot-checked
wrong for periodic-energize auras like Drink).

## Why regenerating db.json/db.bin isn't enough — the wasm and Rotation-tab gaps

Two separate things had to be fixed beyond the CSV correction itself, both
discovered by end-to-end verification in a real browser (not just re-running
the pipeline):

1. **`db.bin` is compiled into the WASM binary.** `assets/database/loader.go`
   `go:embed`s `db.bin` at build time. Editing `db.json`/`db.bin` on disk has no
   effect on the running app until `lib.wasm` is rebuilt (`GOOS=js GOARCH=wasm
   go build -o ./dist/classic/lib.wasm ./sim/wasm/`) and re-copied to
   `dist/classic/assets/lib.wasm`.
2. **The Rotation tab's spell picker never used local tooltip data at all.**
   `ActionId.trySetLocalTooltip()` already existed (added for the item
   pipeline) and is used by the gear picker (`ui/core/components/gear_picker/item_list.tsx`)
   to prefer local DB tooltip text over the live Wowhead widget when local data
   exists. The APL rotation picker
   (`ui/core/components/individual_sim_ui/apl_helpers.tsx`) never called it —
   it only called `setBackgroundAndHref` + `setWowheadDataset`, which always
   defers to the real, live `wow.zamimg.com/js/tooltips.js` widget fetching
   from `nether.wowhead.com` directly. That means **every** spell hover tooltip
   in the Rotation tab was silently showing live retail Wowhead data, no matter
   what this repo's local DB contained — this is why the first attempt at this
   fix appeared to do nothing when checked in a real browser. Fixed by wiring
   `trySetLocalTooltip()` into `apl_helpers.tsx` with the same
   local-first-then-Wowhead-fallback pattern the item picker uses.

If a future audit adds a new local-tooltip data source and a UI location still
shows stale/live Wowhead data despite the pipeline being correct, check whether
that location actually calls `trySetLocalTooltip()` before assuming the data
pipeline is broken.

## What this does NOT fix

- **Spell names and icons** — deliberately left as Wowhead's, per this being
  scoped to values/tooltip text only.
- **Mana cost, range, requirements** — untouched (only the description div and
  the `Channeled (N sec cast)` line are corrected).
- **The Go sim's actual combat math** — `sim/<class>/*.go` hardcodes damage,
  coefficients, tick counts, etc. as Go literals, independent of any tooltip.
  A tooltip correction never changes simulated numbers. A one-off audit
  (2026-09-13, not committed as code) cross-checked a handful of DoT/channeled
  spells against `Spell.csv`:
  - Shadow Word: Pain, Moonfire — Go values match the DBC exactly (high confidence).
  - Rend (`sim/warrior/rend.go`) — Go per-tick damage is consistently ~25-30%
    below the DBC-implied value across all 4 ranks (medium confidence; not
    fixed, flagged for follow-up).
  - Flame Shock — inconsistent audit result, likely a column-mapping issue in
    the audit itself rather than a real Go bug (low confidence; not pursued).
  - No dice-sides/variance column has been reverse-engineered yet, so
    direct-damage spells with min-max ranges (Frostbolt, Fireball, Backstab,
    etc.) remain unauditable against Go with current tooling.

## Reproducing / extending this

To re-run after `CSV's/Spell.csv` or the sim's spell IDs change:

```
python tools/gen_spell_value_overrides.py --dry-run   # preview counts first
python tools/gen_spell_value_overrides.py
go run ./tools/database/gen_db -outDir=./assets -gen=db
cp -r assets/database/* dist/classic/assets/database/
GOOS=js GOARCH=wasm go build -o ./dist/classic/lib.wasm ./sim/wasm/
cp dist/classic/lib.wasm dist/classic/assets/lib.wasm
go build ./...          # sanity check
go test ./sim/...        # check for unexpected result diffs (tooltip changes
                          # should never change these - if they do, investigate)
```

Then verify end-to-end in an actual browser (not just by inspecting
`db.json`) — hover the spell in the Rotation tab and confirm both the
description text and the `Channeled (N sec cast)` line, since a pipeline-level
"looks correct" check on `db.json` alone would have missed both gaps described
above.
