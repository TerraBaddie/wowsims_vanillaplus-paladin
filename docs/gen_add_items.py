"""Cross-reference AtlasLoot's loot tables against the sim DB + VPlus dump and write
assets/db_inputs/add_items.json = {item_id: phase} for equippable level-55+ items
the server has (present in the dump) that the sim DB is missing.

parse_vplus.py adds every id listed here as a brand-new item.
"""
import collections
import glob
import importlib.util
import json
import re
import subprocess

OUT = "assets/db_inputs/add_items.json"

_spec = importlib.util.spec_from_file_location("pv", "docs/parse_vplus.py")
pv = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(pv)

# AtlasLoot table-name prefix -> content phase.
PREFIX_PHASE = [
    ("MC", 1), ("BWL", 3), ("Ony", 3), ("Onyxia", 3), ("ZG", 4),
    ("SCHOLO", 1), ("SM", 1), ("SCARLET", 1), ("STRAT", 1), ("DM", 2),
    ("LBRS", 1), ("UBRS", 1), ("BRD", 1), ("SUNKEN", 1), ("ST", 1),
    ("KKazzak", 3), ("AAzuregos", 3), ("WB", 5), ("Hederine", 5),
    ("AZC", 1), ("Winterspring", 1), ("EggHunt", 1), ("Juju", 1),
    ("AVRep", 1), ("ABRep", 1), ("WSGRep", 1), ("AZCRep", 1),
]
EXCLUDE_TABLES = ("Legendaries", "Artifacts")
BAD_NAME_PREFIX = ("Plans:", "Pattern:", "Schematic:", "Formula:", "Recipe:",
                   "Design:", "Head of ", "Blueprint:", "Alex's", "Test ")
BAD_NAME_SUBSTR = (" Test", " DEP", "DEPRECATED", "(Right)", "(Left)", "Twilight Cultist")
EXCLUDE_IDS = {13262, 18582, 18583, 18584}  # Ashbringer, Azzinoth glaives


def table_phase(name):
    for pfx, ph in sorted(PREFIX_PHASE, key=lambda x: -len(x[0])):
        if name.startswith(pfx):
            return ph
    return None


def main():
    lua = pv.parse_lua(pv.LUA)

    # id -> set of AtlasLoot tables it appears in
    atlas = collections.defaultdict(set)
    for f in glob.glob("CSV's/AtlasLoot/**/*.lua", recursive=True):
        try:
            txt = open(f, encoding="utf8", errors="ignore").read()
        except OSError:
            continue
        for m in re.finditer(r'\n\t([A-Za-z][A-Za-z0-9_]+) = \{\n(.*?)\n\t\};', txt, re.S):
            tbl, body = m.group(1), m.group(2)
            if tbl in EXCLUDE_TABLES:
                continue
            for sid in re.findall(r'\{\s*(\d+),\s*"[^"]*",\s*"=q\d+=', body):
                atlas[int(sid)].add(tbl)

    # Use the PRISTINE db only, so the output is stable regardless of what a prior
    # gen_db run already merged in.
    pristine = {it["id"] for it in json.loads(subprocess.check_output(
        ["git", "show", "HEAD:assets/database/db.json"]))["items"]}
    db = pristine

    add = {}
    seen_key = {}  # (name, type) -> id  (dedupe identical items)
    per_phase = collections.Counter()
    for iid, lines in lua.items():
        if iid in db or iid in pristine or iid in EXCLUDE_IDS:
            continue
        pit = pv.parse_item(lines)
        if "type" not in pit:
            continue
        nm = pit["name"]
        if nm.startswith(BAD_NAME_PREFIX) or any(s in nm for s in BAD_NAME_SUBSTR):
            continue
        blob = "\n".join(lines)
        if "Quest Item" in blob or "Begins a Quest" in blob or "This Item Begins" in blob:
            continue
        lvl = next((int(re.search(r"\d+", l).group())
                    for l in lines if l.startswith("Requires Level")), 0)
        if lvl < 55:
            continue

        tables = atlas.get(iid, set())
        phase = next((p for p in (table_phase(t) for t in tables) if p), None)
        if phase is None:
            if not tables:
                phase = 1          # server has it (in the dump) but AtlasLoot has no page
            else:
                continue           # only appears in excluded tables (legendary/artifact)

        key = (nm, pit["type"])
        if key in seen_key:
            continue
        seen_key[key] = iid
        add[iid] = phase
        per_phase[phase] += 1

    json.dump({str(k): v for k, v in sorted(add.items())}, open(OUT, "w"), indent=0)
    print(f"wrote {OUT}: {len(add)} items")
    for ph, n in sorted(per_phase.items()):
        print(f"  phase {ph}: {n}")


if __name__ == "__main__":
    main()
