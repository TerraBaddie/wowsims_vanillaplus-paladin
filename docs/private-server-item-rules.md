# Private-server item database rules

How the wowsims item database is rebuilt to match the private server. This is the
authoritative spec for the `docs/parse_vplus.py` / `docs/gen_*.py` / `gen_db`
pipeline. If you port the project, this file plus the `CSV's/` inputs are what you
need.

_Last updated: 2026-09-12._

## Pipeline (run order)

The pipeline's Python scripts live in `docs/` (moved out of `tools/` for a
tidier split between one-off/legacy tooling and the actively-run private-server
pipeline — run every command below from the repo root, same as before):

```
python docs/parse_vplus.py      # custom_items.json + renumber.json
python docs/gen_include.py       # included_items.json + add_items.json
python docs/parse_vplus.py       # again: now injects add_items with full shape
python docs/gen_phases.py        # item_phases.json
python docs/gen_sources.py       # item_sources.json (needs item_phases.json + renumber.json)
go run ./tools/database/gen_db -outDir=./assets -gen=db
# then: rebuild lib.wasm, npx vite build -m development, promote sim/**/*.results.tmp,
#       cp -r assets/database/* dist/classic/assets/database/
python docs/remap_presets.py     # rewrite ui/*/gear_sets/*.gear.json through renumber.json
```

`docs/serverdata.py` holds the shared AtlasLoot / dump parsing.
`docs/gen_add_items.py` and `docs/gen_removed_items.py` are superseded by
`gen_include.py` (kept for reference; not run). `removed_items.json` is now `[]`.
The Go side (`tools/database/gen_db/`) stays under `tools/` — it's not one of
these scripts.

## Verifying changes

Two levels: a quick local sanity check right after a rebuild, then an actual
in-browser confirmation that the running dev server picked it up. Do both — the
first catches an obviously-wrong pipeline output, the second is the only way to
know the *client* actually reflects it (a stale service worker, cached
`localStorage`, or a build that silently didn't rerun will otherwise look fine
on disk while showing old data in the browser).

### 1. Local sanity check (no browser)

After a rebuild, spot-check `assets/database/db.json` directly rather than
trusting the pipeline's own "N items written" print statements alone:

```python
import json
d = {i["id"]: i for i in json.load(open("assets/database/db.json"))["items"]}
d.get(<item_id>)              # does it exist? what phase/quality/sources?
"Some Item Name" in {i["name"] for i in d.values()}   # by name instead of id
```

Also worth checking after any pipeline change: total item count and the
phase/quality histograms (`collections.Counter(i.get("phase") for i in
d.values())`) — a big unexpected swing is the fastest way to catch a filter
that's too aggressive or not aggressive enough.

### 2. In-browser confirmation (the dev server at `localhost:8080`)

The Chrome browser tools (`mcp__claude-in-chrome__*` / the Browser pane) can
drive the actual running site — this is how every item/phase/source change in
this pipeline's history was confirmed, not just inferred from the build log.
**This browser also has real internet access** (it's a Chrome extension,
`Claude in Chrome`, not a sandboxed preview) — it's not limited to
`localhost:8080`, so it can cross-reference Wowhead, GitHub, or anything else
if a rule needs checking against an outside source.

Steps, every time:

1. **Force a fresh load** — the client caches the item database client-side, so
   a plain refresh can show stale data even after `dist/` has the new
   `db.bin`. Run in the page console (or via `javascript_tool`):
   ```js
   localStorage.clear();
   const ks = await caches.keys();
   for (const k of ks) await caches.delete(k);
   location.reload(true);
   ```
2. **Navigate to a spec** — `http://localhost:8080/classic/<spec>/`, e.g.
   `shadow_priest`.
3. **Open the item picker for a relevant slot** and use its search box —
   confirms inclusion/exclusion directly (search a removed item's name; it
   should return nothing) and its **Source** column (drop location) and
   **ilvl/quality** color at a glance.
4. **Switch the phase filter dropdown** to confirm an item shows up (or
   doesn't) in the phase it's supposed to be in.
5. **Load a gear preset and read the character-sheet stat panel** (left
   sidebar) to confirm a stat-mechanic change actually applied — e.g. after the
   Intellect→Spell Power change, the Spell Damage total should visibly include
   roughly Intellect/3.
6. **Run a simulation** (Simulate button → Results tab) as a final check that
   nothing crashes and the DPS/HPS number is sane — this is the only step that
   exercises the actual Go/wasm sim engine rather than just the item DB.

For anything that isn't visible in the UI (e.g. confirming the Go build itself
is deterministic, or that `go test` passes), there's no browser substitute —
that's `go build`/`go test` inside the `wowsims-classic` Docker image, covered
in the pipeline section above and in "Determinism" below.

## Input data (in `CSV's/`)

| File | Role |
|---|---|
| `AtlasLoot/` (the server's own addon fork) | **Leading source for what exists and where it drops.** |
| `VPlusItemDB.lua` | In-game tooltip dump. **Leading source for all stats / weapon damage.** Broad extract — contains far more than the server actually uses, so it is NOT an existence signal. |
| `Items.xlsx` | Stat backup only (also a broad dump, ~8000 rows — not curated). |

## Rule 1 — Inclusion (does the item exist on the server?)

An item is included if **either**:

- **(a)** it appears in a non-AQ/Naxx AtlasLoot table
  (`Instances/`, `Crafting/`, `Factions/`, `PvP/`, `Sets/`, `WorldBosses/`,
  `WorldEvents/`), **or**
- **(b)** it is in `VPlusItemDB.lua` as an **equippable blue-or-better** item that
  AtlasLoot does **not** tag as AQ/Naxx and that has **no raid-zone drop source**
  in the base data (zones 3428 / 3429 / 3456).

Rule (b) exists because this AtlasLoot fork does not catalog quest-reward gear,
Zandalar/ZG reputation gear, or every custom item — but those are real pre-raid
content. Rule (b) still can't rescue AQ/Naxx loot: Malice Stone Pendant is tagged by
a `NAX*` table, the Blessed Qiraji weapons by `AQ40*`.

Greens (`quality < blue`) that are not in any AtlasLoot table are dropped — the
raid sim does not need leveling trash.

Details:
- `Crafting/` rows reference **spell IDs** (`"s10619"`). Resolve to the crafted item
  via `AtlasLoot/Core/Spells.lua` → `["craftItem"]`.
- `Sets/` (`sets.lua`) uses the server's **renumbered 25xxx/26xxx IDs**. The sim DB
  still carries the old base-game IDs (e.g. Arcanist Crown is `16795` in the sim,
  `25143` in AtlasLoot). Resolve Sets membership **by item name + slot**, not raw ID,
  or the tier sets get culled.
- **AQ, Naxxramas, and Alterac Valley tables are never read.** Excluded table name
  prefixes: `AQ20`, `AQ40`, `AQEnchants`, `AQOpening`, `NAX`, `Naxxramas`,
  `TheRuinsofAhnQiraj`, `TheTempleofAhnQiraj`, `STRATAtiesh` (Atiesh is a Naxx
  questline), `ScourgeInvasion`, and `AlteracValley` / `AVRep` / `Stormpike` /
  `Frostwolf` (AV is cut). `gen_include.py` also drops any item the sim DB sources
  from the AV reputations (Frostwolf Clan 729, Stormpike Guard 730) plus the
  exalted-reward weapons 19105–19109 that carry no source at all.
  An AQ/Naxx item is included **only if it also appears in a non-AQ/Naxx table**
  (e.g. a world-boss table).

### Exceptions to inclusion

- **Old sub-60 Scarlet Monastery gear**: item whose only AtlasLoot source is an
  `SM*` table and `ilvl < 60` → excluded (the server replaced these with lvl-60
  versions).
- **Not-yet-craftable gear**: a crafted item whose pattern or a required reagent is
  BoP and only obtainable from content the server does not have (AQ/Naxx) →
  excluded. Concretely, the crafted frost-resistance sets are **cut**:
  Icebane, Glacial, Polar, Icy Scale, Ramaladni's Icy Grasp.
- **Scourge Invasion** (`WorldEvents/` `ScourgeInvasionEvent1`, `ScourgeInvasionEvent2`)
  → cut, not obtainable.
- **Recipe items** (name starts with `Plans:`, `Pattern:`, `Schematic:`, `Formula:`,
  `Recipe:`, or `Design:`) → excluded (`gen_include.py` `RECIPE_PREFIXES`, per user
  2026-09-12). AtlasLoot `Crafting/` tables list the recipe drop itself as loot, which
  would otherwise pass Rule 1(a) as if it were equippable gear, alongside the actual
  crafted item it teaches. 154 such items were found leaked into the live DB (65
  `Plans:` + 89 `Pattern:`/`Schematic:`/`Formula:`) and removed by hand from
  `db.json`/`included_items.json`/`item_phases.json`; this filter stops them
  recurring on the next real `gen_include.py` run.

### AQ enchants

The AQ enchant formulas (Shadow Power / Arcane Power to gloves, etc.) ARE on the
server — dropped by world bosses. Keep them. They go through the enchant path, not
item phases, so they normally have no phase; **if one ever needs a phase, use
Phase 1.**

## Rule 2 — Stats

`VPlusItemDB.lua` wins for every stat and for weapon min/max/speed. `Items.xlsx`
only fills a gap when the dump has nothing. wowsims base data is used for nothing
except items the dump also confirms unchanged.

## Rule 3 — Phase (when does it become available?)

Six phases. An item's phase is the **earliest** (lowest-numbered) bucket any of its
AtlasLoot source tables map to.

| Phase | Name | AtlasLoot sources |
|---|---|---|
| 1 | Pre-raid | every 5-man dungeon (incl. lvl-60: Strat, Scholo, UBRS, LBRS, DM, BRD), `PvP/`, all `Crafting/`, `WorldEvents/` (except Scourge Invasion), and non-Zandalar `Factions/` rep |
| 2 | World Bosses | `WorldBosses/` — the four green dragons, Kazzak, Azuregos, custom Hederine, **and Kurinnaxx** |
| 3 | ZG | `ZG*`, `ZulGurub`, and Zandalar reputation (`Factions/` `Zandalar*`) |
| 4 | MC / Onyxia | `MC*`, `MoltenCore`, `Onyxia*` — **and all class tier sets are force-pinned here** |
| 5 | BWL | `BWL*`, `BlackwingLair` |
| 6 | SM | `SM*`, `ScarletMonastery` (lvl-60 items only — see inclusion exception) |

**Every Phase-6 item is forced to Epic quality** (`gen_db`, per user 2026-09-11) —
the old retail Scarlet Monastery gear these ids came from was green/blue, but the
server's lvl-60 rework is all epic.

**Every Phase-6 item's ilvl is forced to 78** (`gen_db`, per user 2026-09-12) — the
old retail ilvl (often sub-40) or missing ilvl (brand-new items) is overwritten.
78 = the AQ40 average (`sources.drop.zoneId 3428` in the pristine db, 111 items,
range 71-88), since SM is the tier that comes after BWL on this server. The 8 items
dropped by the Cathedral wing's final bosses (`SMMograine`/`SMWhitemane` AtlasLoot
tables) are pinned to **80** instead: Aegis of the Scarlet Commander (26302),
Scarlet Leggings (26345), Scarlet Chestpiece (26346), Helm of Zeal (26384),
Reliquary of Light (26428), Scorching Judgement (26286), Whitemane's Chapeau
(26391), Purge (26395). See `smBaseIlvl`/`smBossIlvl`/`smBossItemIds` in
`tools/database/gen_db/main.go`.

### Crafted-item phasing

- **All craftable items are Phase 1.** (Simplification per user 2026-09-11 — manual
  adjustment afterward if a specific recipe needs a later phase.)
- Items that are *not craftable at all yet* (pattern/reagent BoP behind AQ/Naxx) are
  still **excluded** entirely — currently just the frost-resistance sets listed above.

### Tier / PvP sets

- All class tier sets (T1 and T2, retail + the custom druid/shaman/paladin sets) →
  **Phase 4**, forced, regardless of drop location.
- All PvP sets → **Phase 1**.

## Rule priority (apply in order)

1. Included by Rule 1(a) OR 1(b)? If neither → **exclude**.
2. SM-only and `ilvl < 60`? → **exclude**.
3. Not-yet-craftable (frost-resist sets) / Scourge Invasion / `STRATAtiesh`? → **exclude**.
4. Renumber the item to its server ID (Rule 4).
5. Stats from `VPlusItemDB.lua`, else `Items.xlsx`.
6. Phase = earliest bucket of its AtlasLoot tables (Rule 1(b)-only items with no
   table → Phase 1); crafted → P1; tier sets → P4; PvP sets → P1;
   Zandalar rep → P3.

## Rule 4 — Item IDs match the server

Sim item IDs are **renumbered to the private server's IDs**. The server reworked
~170 items (all class tier sets, PvP sets, and reworked dungeon pieces) to new
IDs; wowsims still had them under the retail IDs. `parse_vplus.py` matches each
sim item to its dump entry (by id / name / set+slot / PvP rank+shape), emits the
override under the dump (= server) ID, and writes an old→new map
(`assets/db_inputs/renumber.json`). `gen_db` renames the base-DB items through
that map before merging. Everything that stores an item ID locally is rewritten
through the map too:

- `ui/*/gear_sets/*.gear.json` — preset gear lists.
- Preset **export/import strings** in `ui/*/presets.ts` (base64 protobuf — decode,
  remap the `items[].id` varints, re-encode).
- `sim/**/*.results` — regenerated with `make update-tests`.
- Set membership in `sim/*/item_sets_pve.go` is keyed by set **name**, not piece ID,
  so no Go changes are needed there. Any coded item *effect* keyed by a renumbered
  ID (`sim/*/items.go`, `item_effects.go`) must be remapped — none currently are.

The old retail IDs are added to `removed_items.json` so the base-game orphans don't
linger as duplicates.

## Rule 5 — Loot source (per user 2026-09-11)

The "Source" shown in the item picker is set from the server's own AtlasLoot fork,
not the generic upstream wowsims retail-AtlasLoot data (`assets/db_inputs/atlasloot_db.json`,
fetched from a different, unrelated GitHub repo and still used as a fallback for
items this rule skips).

`docs/gen_sources.py`:
- Scope: `Instances/` (dungeons + raids) and `WorldBosses/`. `Crafting/`/`Factions/`/
  `PvP/` sources are left as whatever they already were — out of scope for now.
- For each item, picks the AtlasLoot table matching its own assigned phase
  (preferring a named boss table over a `*Trash*` table), maps the table's prefix
  to a real vanilla zone id (`ZONE_PREFIXES`, ordered longest-prefix-first so e.g.
  `STRAT` matches before the generic `ST`, and `DME`/`DMN`/`DMW` before the bare `DM`
  — Deadmines and Dire Maul both use `DM*` prefixes in this fork), then tries to
  match the boss name against the sim DB's **existing** NPC roster (`db.json` npcs,
  itself wowhead-derived) by normalized name (exact, or substring with the shorter
  string covering ≥50% of the longer one, only for names ≥6 chars).
- **Never invents an NPC id.** If no confident NPC match is found, the item's
  existing source is left untouched — this is what keeps ambiguous/non-boss table
  labels (a class-set reward table, a named loot bundle like `DMTome`) from
  producing a wrong zone.
- Trash-suffixed tables (`*Trash`, `*Trash1/2`, `*RANDOMBOSSDROPS`) get
  `zone + otherName: "Trash"`, no NPC lookup.
- World bosses get a minted display-only zone (id `900001`, name "World Boss" —
  not a real WoW zone id, registered via `db.MergeZone` in `gen_db`) and a small
  hand-verified name alias table (`WB_NAMES`) since none of these bosses exist in
  the NPC ground truth (no current item referenced them with a real npc id before).
- Output `assets/db_inputs/item_sources.json` = `{server_id: {zoneId, npcId?,
  otherName?}}`. `gen_db` applies it as a **full replacement** of `item.Sources`
  (proto merge only appends repeated fields, so this must be set directly, like
  the phase overrides) — must run after the renumber/phase loads, inside the
  same per-item loop, and after `db.MergeZone` registers zone `900001`.

### Determinism

The whole pipeline is now reproducible byte-for-byte across independent runs
(verified: `docs/parse_vplus.py` → `gen_include.py` → `gen_phases.py` →
`gen_sources.py` → `gen_db` run twice back to back produces identical
`item_sources.json`/`item_phases.json`/`included_items.json`/`renumber.json`
and an identical `db.json`/`db.bin`). Two nondeterminism sources were found and
fixed (2026-09-12):

- `gen_sources.py` picked an item's AtlasLoot table from a Python `set`
  (hash-randomized string iteration order, different every process) without a
  full tie-break, so items with more than one equally-scored candidate table
  could get a different (but individually valid) source on different runs.
  Fixed by sorting the candidate list before scoring and adding the table name
  itself as the final sort key.
- `tools/database/database.go`'s `ToUIProto()` sorted `Enchants` by
  `(EffectId, Type)` only, via `slices.SortFunc` — **not a stable sort** — so
  two enchants sharing that key (same effect granted by a different item/spell
  id) could land in either relative order depending on the Go map's random
  pre-sort order. Fixed by extending the comparator to `(EffectId, Type,
  ItemId, SpellId)`, a fully-disambiguating key.
- Everything else that gets written from a Go map (`Items`, `Zones`, `Npcs`,
  `Factions`, `RandomSuffixes`, icons) was already safe — `mapToSlice` sorts by
  a single unique int32 id, so no tie can occur there.

## Known implementation wrinkles

- `gen_phases.py` currently only scans `Instances/`. It must also scan `Crafting/`
  (via `craftItem`), `Factions/` (incl. `Zandalar*` → P3), `PvP/`, `WorldBosses/`,
  `WorldEvents/`, `Sets/` (by name+slot).
- `gen_phases.py` table-prefix match: the Naxx prefix is `NAX`, not `NAXX`.
- Custom sets (Talonclaw, Ursoc, Cataclysm, Stonefury, Righteous) have stats only —
  no coded set bonuses in `sim/*/item_sets_pve.go`.
- `core.NewItemEffect` / `core.NewItemSet` silently skip missing items/sets so
  removed content does not crash every sim.
