"""Shared helpers for the private-server item pipeline.

See docs/private-server-item-rules.md for the ruleset these implement.

  parse_vplus.py     -> custom_items.json + renumber.json  (stats + server IDs)
  gen_include.py     -> included_items.json + add_items.json  (Rule 1 allowlist)
  gen_phases.py      -> item_phases.json  (Rule 3, 6-phase by drop location)
  gen_removed_items.py -> removed_items.json  (Rule 2/3 hard exclusions)
"""
import glob
import json
import os
import re
import subprocess

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
LUA = os.path.join(REPO, "CSV's/VPlusItemDB.lua")
ATLAS = os.path.join(REPO, "CSV's/AtlasLoot")
DB_JSON = os.path.join(REPO, "assets/database/db.json")

# ---------------------------------------------------------------------------
# pristine sim DB (pre-pipeline) -- read from git so every tool is idempotent
# ---------------------------------------------------------------------------

def pristine_db():
    try:
        raw = subprocess.check_output(
            ["git", "show", "HEAD:assets/database/db.json"], cwd=REPO)
        return {i["id"]: i for i in json.loads(raw)["items"]}
    except Exception:
        return {i["id"]: i for i in json.load(open(DB_JSON))["items"]}


# ---------------------------------------------------------------------------
# VPlusItemDB.lua  (the in-game tooltip dump)
# ---------------------------------------------------------------------------

def dump_blocks():
    """{item_id: raw tooltip text block} from the dump, largest block per id."""
    txt = open(LUA, encoding="utf8").read()
    parts = re.split(r'\n\t\t\t\[(\d+)\] = \{\n', txt)
    out = {}
    for i in range(1, len(parts), 2):
        sid = int(parts[i])
        body = parts[i + 1].split("\n\t\t\t}")[0]
        if sid not in out or len(body) > len(out[sid]):
            out[sid] = body
    return out


def dump_ids():
    return set(dump_blocks())


# ---------------------------------------------------------------------------
# AtlasLoot addon data
# ---------------------------------------------------------------------------

# Tables we never read: AQ20 / AQ40 / Naxxramas, the Atiesh (Naxx questline) frame
# drop in Stratholme, the Scourge Invasion pre-Naxx world event, and all Alterac
# Valley content (drops, rep rewards, and the Stormpike/Frostwolf insignia trinkets).
IGNORED_TABLE = re.compile(
    r'^(AQ|NAX|Naxxramas|TheRuinsofAhnQiraj|TheTempleofAhnQiraj'
    r'|STRATAtiesh|WBScourgeInvasion|ScourgeInvasion'
    r'|AlteracValley|AVRep|Stormpike|Frostwolf)')
AQ_NAXX = IGNORED_TABLE  # backwards-compatible alias

# Phase buckets by AtlasLoot table-name prefix. None => ignore.
def table_bucket(t):
    if IGNORED_TABLE.match(t):
        return None
    if t.startswith(("ZG", "ZulGurub")) or t.startswith("Zandalar"):
        return 3
    if t.startswith(("MC", "MoltenCore", "Onyxia")):
        return 4
    if t.startswith(("BWL", "BlackwingLair")):
        return 5
    if t.startswith(("SM", "ScarletMonastery", "Scarlet")):
        return 6
    return 1  # every 5-man dungeon, Crafting, PvP, WorldEvents, non-ZG Factions


def _craft_item_map():
    """AtlasLoot Crafting rows use spell ids; resolve to the crafted item id."""
    txt = open(os.path.join(ATLAS, "Core/Spells.lua"), encoding="utf8", errors="ignore").read()
    out = {}
    for m in re.finditer(r'\[(\d+)\]\s*=\s*\{(.*?)\n\t*\}', txt, re.S):
        cm = re.search(r'\["craftItem"\]\s*=\s*(\d+)', m.group(2))
        if cm:
            out[m.group(1)] = int(cm.group(1))
    return out


def atlasloot():
    """Returns (id_tables, names).

      id_tables : {item_id: set(table_name)}   -- every table an item appears in
      names     : set(lower-case item names AtlasLoot lists)  -- for name matching

    WorldBosses tables get a synthetic 'WB' prefix so they bucket to phase 2.
    """
    craft = _craft_item_map()
    id_tables = {}
    names = set()
    for path in glob.glob(ATLAS + "/**/*.lua", recursive=True):
        if any(x in path for x in ("/Core/", "/Libs/", "/Tooltips/", "TableRegister")):
            continue
        is_wb = "WorldBosses" in path
        txt = open(path, encoding="utf8", errors="ignore").read()
        # table name -> its body, so we can attribute ids to tables
        for tm in re.finditer(r'\n\t([A-Za-z][A-Za-z0-9_]+) = \{\n(.*?)\n\t\};', txt, re.S):
            tname = tm.group(1)
            if is_wb:
                tname = "WB" + tname
            body = tm.group(2)
            row_ids = set()
            for rm in re.finditer(r'\{\s*"?(s?\d{3,6})"?\s*,\s*"[A-Za-z]', body):
                tok = rm.group(1)
                if tok.startswith("s"):
                    it = craft.get(tok[1:])
                    if it:
                        row_ids.add(it)
                else:
                    v = int(tok)
                    if 1000 <= v <= 99999:
                        row_ids.add(v)
            for i in row_ids:
                id_tables.setdefault(i, set()).add(tname)
            if table_bucket(tname) is not None:   # skip AQ/Naxx/Scourge names
                for nm in re.findall(r'=q\d+=([^"]+?)"', body):
                    names.add(nm.strip().lower())
    return id_tables, names


def wb_prefix_ok(t):
    """WorldBosses table -> phase 2 (all of them: dragons, Kazzak, Azuregos,
    Hederine, Kurinnaxx)."""
    return t.startswith("WB")


# convenience: bucket a set of table names (None entries dropped) -> min phase
def phase_from_tables(tables):
    buckets = set()
    for t in tables:
        if t.startswith("WB"):
            buckets.add(2)
            continue
        b = table_bucket(t)
        if b is not None:
            buckets.add(b)
    return min(buckets) if buckets else None
