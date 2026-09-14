# Open items

Things either of us raised across this work that haven't been done yet. Not a
backlog of ideas — only things actually said and left open. Grouped by how they
came up, most concrete first.

_As of 2026-09-12._

## Explicitly deferred to "adjust manually later"

- **Crafted-item phasing is flat.** Every craftable item is Phase 1 right now.
  We'd originally discussed bumping a crafted item to a later phase when its
  pattern *and* a reagent are both BoP from that phase's content (Dark
  Iron/Molten/Corehound/Flarecore → MC; Chromatic/Dreamscale → BWL), but you
  said to skip that and just set everything to Phase 1, adjusting by hand if a
  specific recipe needs it. Nothing's been adjusted since — if any of those
  MC/BWL-reagent recipes should actually sit later than Phase 1, that's still
  outstanding.

## Known gaps I flagged but didn't build

- ~~**Custom class sets have no set bonuses.**~~ Done 2026-09-13: Talonclaw
  Regalia, Ursoc Armor (druid), Cataclysm Armor, The Stonefury (shaman),
  Righteous Armor (paladin) all implemented (commit `d464da4b3`). See
  [CHANGES.md](CHANGES.md) Part C for detail and reproduction steps.
- ~~**All PvP sets need their bonuses done — priority.**~~ Done earlier in the
  2026-09-13 session (commits `79cb702a3`, `590bfad43`) — every class's PvP
  sets (both blue and epic tier) had a universal 2pc/6pc bonus-value swap bug,
  fixed and verified against the server's real tooltips.
- ~~**Really, every set needs a full pass.**~~ Done 2026-09-13 (commit
  `bfcd64728`): audited every class's PvE tier/dungeon sets, one background
  agent per class, three classes at a time. Found and fixed wrong
  piece-count thresholds and/or wrong bonus values in the large majority of
  already-"implemented" sets across all 9 classes — see CHANGES.md Part C for
  the full list. Remaining known gap: **Cenarion Armor's root-cause data bug
  was fixed, but the same underlying question — why did the generic
  Wowhead-derived pipeline fail to tag `setName` for some items when
  `custom_items.json` had no override for them — was never diagnosed**, only
  patched item-by-item as it was found (Righteous Boots, the 8 Cenarion
  Armor pieces, Striker's Garb, a handful of AQ40/Naxx sets that turned out
  to be content gaps instead). Also, several sets fixed this pass have
  bonuses left as documented no-ops because the underlying spell/mechanic
  isn't simulated at all in this fork (e.g. Feral Tank abilities, Hibernate,
  Life Tap's cooldown) — those are correctly *not* bugs, but would need
  actual new sim features, not a set-bonus fix, if ever wanted.
- **T3-looking sets (Dreadnaught, Cryptstalker, etc.) aren't pinned to a
  phase.** They fall through the normal location rule, which puts them at
  Phase 1 since they have no AtlasLoot raid-instance source on this server.
  Fine as long as the server doesn't actually raid that content — worth
  revisiting only if it turns out to.

## Raised, never actually answered

- **Scarlet Monastery set completeness.** Early on you asked me to check
  whether the Scarlet Crusade set has all its level-60 pieces, and said you
  could supply the last piece yourself if not. I don't have a record of ever
  reporting back on that. Worth re-checking the 5-piece Scarlet Crusade set in
  the current Phase-6 (SM) item list and asking you for the missing piece if
  one's still short.

## Scope limits I stated out loud

- **AtlasLoot-based loot sources only cover dungeons/raids/world bosses.**
  `gen_sources.py` explicitly leaves `Crafting/`, `Factions/`, and `PvP/`
  sources on whatever the generic upstream (retail) data already had, because
  matching those confidently against real faction/spell ids felt like a
  separate, riskier piece of work. If you want those corrected too, that's a
  distinct follow-up, not something quietly finished.

## Minor / cosmetic

- **One world-boss table's display name is ugly.** `ASpiritA` (an AtlasLoot
  world-boss table) isn't in the hand-verified `WB_NAMES` alias list, so any
  item sourced only from it falls back to an auto-prettified "A Spirit A"
  instead of a real boss name. Low-stakes (I don't know what boss this
  actually is without more digging) but noted rather than silently left ugly.
- **A stale line in the pipeline doc.** `docs/private-server-item-rules.md`'s
  "Known implementation wrinkles" section still says `gen_phases.py` "only
  scans `Instances/`" — that was true before the `serverdata.py` rewrite, but
  `gen_phases.py` now goes through `serverdata.atlasloot()` and already covers
  all seven AtlasLoot folders. The prose just never got updated to say so.

## Housekeeping, not content

- **Nothing from this work is committed.** `git log` still shows the same last
  commit as when this all started; `git status` has 216 modified/new files
  covering everything from this session (item pipeline rewrite, Int→Spell
  Power, script move, determinism fixes) plus the earlier talent-tree work.
  It's all just sitting in the working tree. Say the word when you want it
  committed (and how you want it split up, if at all).
- **Stray scratch files at the repo root**, not created by me and not
  referenced by anything built this session: `_add_candidates.json`,
  `_missing_vplus_items.txt`, `_olddb.json`, `_pvp_obsolete.json`,
  `_removelist.json`, `_removelist_aqnaxx.json`, `"Start Server.bat"`. Flagged
  before, still there — delete or keep, your call.
