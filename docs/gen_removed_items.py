"""Generate assets/db_inputs/removed_items.json -- item IDs to drop from the sim DB
because the private server doesn't have that content.

Rules:
  * AQ20/AQ40/Naxxramas loot (AtlasLoot NAX*/AQ20*/AQ40* boss table, or sim-DB
    source zone 3428/3429/3456): removed unless the id is in Items.xlsx. The VPlus
    dump lists the full raid item range so it is NOT a keep-signal for raid loot.
  * member of REMOVE_SETS (AQ40 tier 2.5 sets + obsolete PvP groupings).
  * old pre-rework Scarlet Monastery gear (SM-only AtlasLoot source, ilvl < 60).
  * otherwise, removed if absent from BOTH VPlusItemDB.lua and Items.xlsx and
    sourced from a raid zone.

The server KEEPS both tier 1 and tier 2 armour sets (Might + Wrath, Prophecy +
Transcendence, ...), so tier 2 is NOT removed.
"""
import json
import os
import re
import subprocess
import sys

import openpyxl

DB = "assets/database/db.json"
LUA = "CSV's/VPlusItemDB.lua"
XLSX = "CSV's/Items.xlsx"
OUT = "assets/db_inputs/removed_items.json"

RAID_ZONES = {3428: "AQ40", 3429: "AQ20", 3456: "Naxxramas"}
SM_INSTANCES = "CSV's/AtlasLoot/Instances/instances.en.lua"

# AQ40 tier 2.5 armour sets ("5/5" sets) the server does not have.
_AQ_SETS = {
    "Conqueror's Battlegear", "Avenger's Battlegear", "Striker's Garb",
    "Deathdealer's Embrace", "Garments of the Oracle", "Genesis Raiment",
    "Enigma Vestments", "Doomcaller's Attire", "Stormcaller's Garb",
    "Gift of the Gathering Storm", "The Gift of the Gathering Storm",
}

# Obsolete PvP set groupings: the server's canonical rank 7-10 sets are
# "<rank>'s Arcanum/Battlearmor/Dreadgear/Guard/Investiture/Pursuance/Redoubt/
# Refuge/Stormcaller" (per AtlasLoot). These older duplicate groupings are dead.
_OLD_PVP_SETS = {
    "Champion's Battlegear", "Champion's Earthshaker", "Champion's Pursuit",
    "Champion's Raiment", "Champion's Regalia", "Champion's Sanctuary",
    "Champion's Threads", "Champion's Vestments",
    "Lieutenant Commander's Aegis", "Lieutenant Commander's Battlegear",
    "Lieutenant Commander's Pursuit", "Lieutenant Commander's Raiment",
    "Lieutenant Commander's Regalia", "Lieutenant Commander's Sanctuary",
    "Lieutenant Commander's Threads", "Lieutenant Commander's Vestments",
    "The Highlander's Fortitude",
}

REMOVE_SETS = _AQ_SETS | _OLD_PVP_SETS


def xlsx_ids():
    wb = openpyxl.load_workbook(XLSX, read_only=True, data_only=True)
    return {r[1] for r in wb.active.iter_rows(values_only=True) if isinstance(r[1], int)}


def sheet_ids():
    lua = open(LUA, encoding="utf8").read()
    ids = set(int(x) for x in re.findall(r'\n\t\t\t\[(\d+)\] = \{', lua))
    return ids | xlsx_ids()


# AtlasLoot boss-table prefixes for content the server does not have (AQ20/AQ40/Naxx).
# The VPlus dump is a broad extract and still lists these items, so "present in the
# dump" is NOT a keep-signal for raid loot -- only an explicit Items.xlsx entry is.
AQ_NAXX_TABLE = re.compile(r'^(NAX|AQ20|AQ40)')


def raid_zone(item):
    for s in item.get("sources", []):
        d = s.get("drop")
        if d and d.get("zoneId") in RAID_ZONES:
            return RAID_ZONES[d["zoneId"]]
    return None


def main():
    sheet = sheet_ids()
    xlsx = xlsx_ids()
    # Compute against the pristine (pre-removal) DB so the script is idempotent.
    try:
        db = json.loads(subprocess.check_output(
            ["git", "show", "HEAD:assets/database/db.json"]))["items"]
    except Exception:
        db = json.load(open(DB))["items"]

    # Items whose ONLY drop source is Scarlet Monastery, at ilvl < 60 -- the old
    # pre-rework SM gear the server replaced with level-60 versions.
    body = open(SM_INSTANCES, encoding="utf8", errors="ignore").read()
    body = body[body.index('AtlasLoot_Data["AtlasLootItems"]'):]
    id_tables = {}
    for m in re.finditer(r'\n\t([A-Za-z][A-Za-z0-9_]+) = \{\n(.*?)\n\t\};', body, re.S):
        for sid in re.findall(r'\{\s*(\d+),\s*"[^"]*",\s*"=q\d+=', m.group(2)):
            id_tables.setdefault(int(sid), set()).add(m.group(1))

    def sm_only_old(it):
        tbls = id_tables.get(it["id"])
        if not tbls or not all(t.startswith(("SM", "SCARLET")) for t in tbls):
            return False
        return (it.get("ilvl") or 0) < 60

    def aq_naxx(it):
        # AtlasLoot lists it under an AQ20/AQ40/Naxx boss, or the sim DB sources it
        # from one of those raid zones. Either way the server doesn't have it.
        if any(AQ_NAXX_TABLE.match(t) for t in id_tables.get(it["id"], ())):
            return True
        return raid_zone(it) is not None

    remove = {}
    for it in db:
        # AQ40 tier 2.5 sets: remove even though they're in the dump.
        if it.get("setName") in REMOVE_SETS:
            remove[it["id"]] = "AQ-set:" + it["setName"]
            continue
        if sm_only_old(it):
            remove[it["id"]] = "SM-old"
            continue
        # AQ/Naxx loot: kept only by an explicit Items.xlsx entry, never by the dump.
        if aq_naxx(it) and it["id"] not in xlsx:
            remove[it["id"]] = raid_zone(it) or "AQ/Naxx"
            continue
        if it["id"] in sheet:
            continue
        z = raid_zone(it)
        if z:
            remove[it["id"]] = z

    json.dump(sorted(remove), open(OUT, "w"), indent=0)
    by_reason = {}
    for r in remove.values():
        by_reason[r.split(":")[0]] = by_reason.get(r.split(":")[0], 0) + 1
    print(f"wrote {OUT}: {len(remove)} items removed  {by_reason}")


if __name__ == "__main__":
    main()
