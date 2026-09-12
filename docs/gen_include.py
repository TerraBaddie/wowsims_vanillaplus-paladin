"""Rule 1 (inclusion) from docs/private-server-item-rules.md.

An item is on the server if EITHER
  (a) it appears in a non-AQ/Naxx AtlasLoot table, OR
  (b) it is in VPlusItemDB.lua as an equippable blue-or-better item that AtlasLoot
      does not tag AQ/Naxx and that has no raid-zone drop source in the base data.

Writes (server IDs -- i.e. after docs/parse_vplus.py's renumber map):
  assets/db_inputs/included_items.json  = [id, ...]      allowlist; gen_db drops the rest
  assets/db_inputs/add_items.json       = {id: phase}    new-to-DB items for parse_vplus
"""
import glob
import importlib.util
import json
import os
import re

import serverdata as sd

REPO = sd.REPO
INCLUDED = os.path.join(REPO, "assets/db_inputs/included_items.json")
ADD = os.path.join(REPO, "assets/db_inputs/add_items.json")
RENUMBER = os.path.join(REPO, "assets/db_inputs/renumber.json")

_spec = importlib.util.spec_from_file_location("pv", os.path.join(REPO, "docs/parse_vplus.py"))
pv = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(pv)

RAID_ZONES = {3428, 3429, 3456}

# Alterac Valley: cut entirely (per user 2026-09-11). AtlasLoot AVRep*/Stormpike/
# Frostwolf tables are in serverdata.IGNORED_TABLE; this also drops items the sim DB
# sources from the AV reputations, plus a few exalted-reward weapons that carry
# neither a rep source nor an AtlasLoot entry.
AV_REP_FACTIONS = {729, 730}  # Frostwolf Clan, Stormpike Guard
AV_EXTRA_IDS = {19105, 19106, 19107, 19108, 19109}

# Rule 3 hard exclusions that would otherwise pass Rule 1 (not craftable yet).
FROST_RESIST_SETS = {"Icebane", "Glacial", "Polar", "Icy Scale"}

# Recipe items (crafting plans/patterns/schematics/etc.) are not equippable gear --
# AtlasLoot Crafting tables list the recipe drop itself, which would otherwise pass
# Rule 1a alongside the actual crafted item. Exclude by name prefix.
RECIPE_PREFIXES = ("Plans:", "Pattern:", "Schematic:", "Formula:", "Recipe:", "Design:", "Manual:")


def _dump_lines(body):
    out = []
    for ln in body.split("\n"):
        m = re.match(r'^\t\t\t\t\[\d+\] = "((?:[^"\\]|\\.)*)",?$', ln)
        if m:
            s = m.group(1).replace('\\"', '"').replace("\\n", "\n").strip()
            if s:
                out.append(s)
    return out


def _raid_sourced(it):
    for s in it.get("sources", []):
        d = s.get("drop")
        if d and d.get("zoneId") in RAID_ZONES:
            return True
    return False


def _alterac_valley(iid, it):
    if iid in AV_EXTRA_IDS:
        return True
    for s in it.get("sources", []):
        if s.get("rep", {}).get("repFactionId") in AV_REP_FACTIONS:
            return True
    return False


def _frost_resist(name):
    return any(name.startswith(p + " ") or name == p for p in FROST_RESIST_SETS) \
        or name.startswith("Ramaladni")


def _is_recipe(name):
    return name.startswith(RECIPE_PREFIXES)


def main():
    pristine = sd.pristine_db()
    id_tables, atlas_names = sd.atlasloot()
    blocks = sd.dump_blocks()
    try:
        renum = {int(k): v for k, v in json.load(open(RENUMBER)).items()}
    except OSError:
        renum = {}

    def tables_of(iid):
        return id_tables.get(iid, set()) | id_tables.get(renum.get(iid, iid), set())

    def in_atlas(iid, name):
        live = [t for t in tables_of(iid) if not sd.IGNORED_TABLE.match(t)]
        if live:
            return True
        return bool(name) and name.lower() in atlas_names

    def atlas_aqnaxx_only(iid):
        tbls = tables_of(iid)
        return bool(tbls) and all(sd.IGNORED_TABLE.match(t) for t in tbls)

    def sm_old(iid, ilvl):
        live = [t for t in tables_of(iid) if not sd.IGNORED_TABLE.match(t)]
        return live and all(t.startswith(("SM", "Scarlet")) for t in live) and (ilvl or 0) < 60

    included = set()
    add = {}
    n_a = n_b = 0

    # pristine sim items
    for iid, it in pristine.items():
        name = it.get("name", "")
        server_id = renum.get(iid, iid)
        if _frost_resist(name) or _is_recipe(name) or sm_old(iid, it.get("ilvl")) or _alterac_valley(iid, it):
            continue
        if in_atlas(iid, name):
            included.add(server_id)
            n_a += 1
            continue
        # Rule 1(b): dump-listed, equippable, blue+, not AQ/Naxx, not raid-sourced
        if (iid in blocks
                and (it.get("quality") or 0) >= 3
                and it.get("type")
                and not atlas_aqnaxx_only(iid)
                and not _raid_sourced(it)):
            included.add(server_id)
            n_b += 1

    # dump items the sim DB doesn't have at all -> candidates to ADD
    for iid, body in blocks.items():
        server_id = renum.get(iid, iid)
        if server_id in included or iid in pristine or server_id in pristine:
            continue
        pit = pv.parse_item(_dump_lines(body))
        name = pit.get("name", "")
        # (sm_old only applies to pristine items -- the dump carries no item level,
        #  and a dump-only SM item is the server's new lvl-60 version.)
        if not pit.get("type") or _frost_resist(name) or _is_recipe(name) or iid in AV_EXTRA_IDS:
            continue
        q = pit.get("quality")
        if in_atlas(iid, name):
            included.add(server_id)
            add[server_id] = 1
            n_a += 1
        elif q is not None and q >= 3 and not atlas_aqnaxx_only(iid):
            included.add(server_id)
            add[server_id] = 1
            n_b += 1

    # Force-keep items hand-picked in a built-in gear preset -- covers random-suffix
    # world greens the pre-BiS sets use that AtlasLoot never lists. Still refuse
    # anything AtlasLoot tags AQ/Naxx or that is raid-zone-sourced (dead raid gear
    # left in old presets).
    n_preset = 0
    for gj in glob.glob(os.path.join(REPO, "ui/*/gear_sets/*.gear.json")):
        for it in json.load(open(gj)).get("items", []):
            iid = it.get("id")
            if not iid:
                continue
            sid = renum.get(iid, iid)
            if sid in included or atlas_aqnaxx_only(iid):
                continue
            pit_it = pristine.get(iid) or pristine.get(sid)
            if pit_it and (_raid_sourced(pit_it) or _alterac_valley(iid, pit_it) or _is_recipe(pit_it.get("name", ""))):
                continue
            if iid in AV_EXTRA_IDS:
                continue
            included.add(sid)
            n_preset += 1

    json.dump(sorted(included), open(INCLUDED, "w"), indent=0)
    json.dump({str(k): v for k, v in sorted(add.items())}, open(ADD, "w"), indent=0)
    print(f"wrote {INCLUDED}: {len(included)} items  "
          f"(Rule 1a: {n_a}, Rule 1b: {n_b}, preset-pinned: {n_preset})")
    print(f"wrote {ADD}: {len(add)} new-to-DB items")


if __name__ == "__main__":
    main()
