"""Assign every included item a phase from WHERE IT DROPS (AtlasLoot data).

See docs/private-server-item-rules.md. Six phases, earliest drop location wins:

  1 Pre-raid    : 5-man dungeons, Crafting, PvP, WorldEvents, non-ZG rep
  2 World Bosses: green dragons, Kazzak, Azuregos, Hederine, Kurinnaxx
  3 ZG          : Zul'Gurub + Zandalar reputation
  4 MC / Onyxia : Molten Core + Onyxia  (+ ALL class tier sets, forced)
  5 BWL         : Blackwing Lair
  6 SM          : Scarlet Monastery (lvl-60 items only)

Writes assets/db_inputs/item_phases.json = {server_id: phase}. Applied last in gen_db.
"""
import collections
import importlib.util
import json
import os

import serverdata as sd

REPO = sd.REPO
OUT = os.path.join(REPO, "assets/db_inputs/item_phases.json")
RENUMBER = os.path.join(REPO, "assets/db_inputs/renumber.json")

_spec = importlib.util.spec_from_file_location("pv", os.path.join(REPO, "docs/parse_vplus.py"))
pv = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(pv)

# Every class tier set (T1 + T2 + custom) is pinned to the Molten Core phase.
TIER_SETS = set(pv.TIER1_SETS) | set(pv.NEW_SETS) | {
    "Netherwind Regalia", "Nemesis Raiment", "Bloodfang Armor", "Stormrage Raiment",
    "Dragonstalker Armor", "The Ten Storms", "Vestments of Transcendence",
    "Battlegear of Wrath", "Judgement Armor", "Cenarion Armor",
}
TIER_PHASE = 4


def main():
    pristine = sd.pristine_db()
    id_tables, _ = sd.atlasloot()
    blocks = sd.dump_blocks()

    try:
        renum = {int(k): v for k, v in json.load(open(RENUMBER)).items()}
    except OSError:
        renum = {}

    # {id: (setName, ilvl)} for everything that can end up in the DB: pristine items
    # plus every dump item (covers custom/new + renumbered pieces).
    # ilvl of 60 for dump-only items: the dump has no item level, and a dump-only
    # SM item is the server's new lvl-60 version (only pristine sub-60 SM gear is cut).
    meta = {i: (it.get("setName"), it.get("ilvl") or 0) for i, it in pristine.items()}
    for i, body in blocks.items():
        if i in meta:
            continue
        pit = pv.parse_item(_lines(body))
        meta[i] = (pit.get("setName"), 60)

    phases = {}
    for iid, (sn, ilvl) in meta.items():
        server_id = renum.get(iid, iid)
        if sn in TIER_SETS:
            phases[server_id] = TIER_PHASE
            continue
        ph = sd.phase_from_tables(
            id_tables.get(iid, set()) | id_tables.get(server_id, set()))
        if ph is None:
            phases[server_id] = 1
            continue
        if ph == 6 and ilvl < 60:
            continue  # old sub-60 SM item -> removed, not phased
        phases[server_id] = ph

    json.dump({str(k): v for k, v in sorted(phases.items())}, open(OUT, "w"), indent=0)
    print(f"wrote {OUT}: {len(phases)} items  "
          f"{dict(sorted(collections.Counter(phases.values()).items()))}")


def _lines(body):
    import re
    out = []
    for ln in body.split("\n"):
        m = re.match(r'^\t\t\t\t\[\d+\] = "((?:[^"\\]|\\.)*)",?$', ln)
        if m:
            s = m.group(1).replace('\\"', '"').replace("\\n", "\n").strip()
            if s:
                out.append(s)
    return out


if __name__ == "__main__":
    main()
