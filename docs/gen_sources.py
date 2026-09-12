"""Set each item's displayed loot source (drop location) from the server's own
AtlasLoot fork (CSV's/AtlasLoot), rather than whatever the generic upstream
wowsims retail-AtlasLoot/wowhead data attached.

Scope: dungeon/raid instance tables and world-boss tables (the vast majority of
"where does this drop" questions). Crafting/Factions/PvP sources are left as-is.

Boss identification never invents an NPC id: it matches the AtlasLoot table name
against the sim DB's existing (wowhead-derived) NPC roster by name, so the
Wowhead NPC link keeps working. When no confident match exists, it falls back to
a prettified name with no clickable link.

Writes assets/db_inputs/item_sources.json = {server_id: {zoneId, npcId?, otherName?}}.
gen_db applies this as a full replacement of the item's `sources` (last wins).
"""
import json
import os
import re

import serverdata as sd

REPO = sd.REPO
OUT = os.path.join(REPO, "assets/db_inputs/item_sources.json")
RENUMBER = os.path.join(REPO, "assets/db_inputs/renumber.json")
PHASES = os.path.join(REPO, "assets/db_inputs/item_phases.json")
DB_JSON = os.path.join(REPO, "assets/database/db.json")

WORLD_BOSS_ZONE_ID = 900001  # minted display-only zone; not a real WoW zone id
WORLD_BOSS_ZONE_NAME = "World Boss"

# AtlasLoot table-name prefix -> real vanilla zone id, longest/most-specific first
# so e.g. "STRAT" is matched before the generic "ST", and "DME"/"DMN"/"DMW" before "DM".
ZONE_PREFIXES = [
    ("SCHOLO", 2057), ("Scholomance", 2057),
    ("STRATBS", 2017), ("STRAT", 2017), ("Stratholme", 2017),
    ("ST", 1477), ("TheSunkenTemple", 1477),
    ("SMA", 796), ("SMC", 796), ("SMG", 796), ("SML", 796),
    ("SM", 796), ("Scarlet", 796), ("ScarletMonastery", 796),
    ("SFK", 209), ("Shadowfang", 209), ("ShadowfangKeep", 209),
    ("SW", 717), ("TheStockade", 717),
    ("RFK", 491),
    ("RFD", 722), ("RazorfenDowns", 722),
    ("RFC", 2437), ("Ragefire", 2437), ("RagefireChasm", 2437),
    ("BFD", 719), ("Blackfathom", 719), ("BlackfathomDeeps", 719),
    ("BRD", 1584), ("BlackrockDepths", 1584),
    ("LBRS", 1583), ("UBRS", 1583),
    ("BlackrockSpireLower", 1583), ("BlackrockSpireUpper", 1583),
    ("BWL", 2677), ("BlackwingLair", 2677),
    ("DME", 2557), ("DMN", 2557), ("DMW", 2557), ("DMNTRIBUTERUN", 2557),
    ("DireMaulEast", 2557), ("DireMaulNorth", 2557), ("DireMaulWest", 2557),
    ("DireMaulEnt", 2557),
    ("DM", 1581), ("TheDeadmines", 1581), ("TheDeadminesEnt", 1581),
    ("GnDI", 721), ("Gn", 721), ("Gnomeregan", 721), ("GnomereganEnt", 721),
    ("MCRANDOMBOSSDROPS", 2717), ("MC", 2717), ("MoltenCore", 2717), ("Molten", 2717),
    ("Mara", 2100), ("Maraudon", 2100), ("MaraudonEnt", 2100),
    ("Onyxias", 2159), ("Onyxia", 2159),
    ("Uld", 1337), ("Uldaman", 1337), ("UldamanEnt", 1337),
    ("WC", 718), ("Wailing", 718), ("WailingCaverns", 718), ("WailingCavernsEnt", 718),
    ("ZF", 1176), ("ZulFarrak", 1176),
    ("ZG", 1977), ("ZulGurub", 1977),
]

TRASH_RE = re.compile(r'(Trash\d*|RANDOMBOSSDROPS|Ent)$')

# WorldBosses/ table names -> clean display name. Several bosses have more than
# one table (alt reward tiers); alias them to one name.
WB_NAMES = {
    "Azuregos": "Azuregos", "AAzuregos": "Azuregos",
    "LordKazzak": "Lord Kazzak", "KKazzak": "Lord Kazzak",
    "LadyHederine": "Lady Hederine", "Hederine": "Lady Hederine",
    "SiliKurinnaxx": "Kurinnaxx", "WBKurinnaxx": "Kurinnaxx",
    "DTaerar": "Taerar", "DEmeriss": "Emeriss", "DLethon": "Lethon", "DYsondre": "Ysondre",
    "FourDragons": "Emerald Dragons",
}


def _norm(s):
    return re.sub(r'[^a-z0-9]', '', s.lower())


def _prettify(slug):
    # insert spaces before capitals: "DarkmasterGandling" -> "Darkmaster Gandling"
    return re.sub(r'(?<!^)(?=[A-Z])', ' ', slug).strip()


def zone_for_table(tname):
    for prefix, zid in ZONE_PREFIXES:
        if tname.startswith(prefix):
            return zid, tname[len(prefix):]
    return None, None


def main():
    id_tables, _ = sd.atlasloot()
    renum = {}
    try:
        renum = {int(k): v for k, v in json.load(open(RENUMBER)).items()}
    except OSError:
        pass
    try:
        phases = {int(k): v for k, v in json.load(open(PHASES)).items()}
    except OSError:
        phases = {}

    npcs = json.load(open(DB_JSON))["npcs"]
    npc_by_norm = {}
    for n in npcs:
        npc_by_norm.setdefault(_norm(n["name"]), n)

    def best_npc(slug_norm):
        if not slug_norm:
            return None
        if slug_norm in npc_by_norm:
            return npc_by_norm[slug_norm]
        if len(slug_norm) < 6:
            return None  # too short to fuzzy-match safely (e.g. "Tome", "Ent")
        best = None
        for key, n in npc_by_norm.items():
            # require the shorter string to cover most of the longer one, so
            # "geddon" ~ "barongeddon" hits but "ram" ~ "ramaladni" doesn't.
            shorter, longer = (slug_norm, key) if len(slug_norm) <= len(key) else (key, slug_norm)
            if shorter in longer and len(shorter) >= 0.5 * len(longer):
                if best is None or len(key) > len(_norm(best["name"])):
                    best = n
        return best

    out = {}
    for iid in sorted(id_tables):
        tables = id_tables[iid]
        server_id = renum.get(iid, iid)
        # sorted() first: `tables` is a set (hash-randomized string iteration
        # order, different every process), so without this the tie-break below
        # would pick a different table on every run for items with more than
        # one equally-scored candidate.
        live = sorted(t for t in tables if not sd.IGNORED_TABLE.match(t))
        if not live:
            continue

        # Prefer a table matching the item's own assigned phase bucket, and
        # prefer a specific boss table over a generic trash table. Table name
        # is the final tie-break, so equal-score candidates always resolve the
        # same way.
        item_phase = phases.get(server_id) or phases.get(iid)

        def score(t):
            bucket = 2 if t.startswith("WB") else sd.table_bucket(t)
            phase_match = 1 if (item_phase is not None and bucket == item_phase) else 0
            is_trash = 1 if TRASH_RE.search(t) else 0
            return (-phase_match, is_trash, t)

        live.sort(key=score)
        chosen = live[0]

        if chosen.startswith("WB"):
            boss_slug = chosen[2:]
            entry = {"zoneId": WORLD_BOSS_ZONE_ID}
            if boss_slug in WB_NAMES:
                entry["otherName"] = WB_NAMES[boss_slug]
            else:
                npc = best_npc(_norm(boss_slug))
                if npc:
                    entry["npcId"] = npc["id"]
                else:
                    entry["otherName"] = _prettify(boss_slug)
            out[server_id] = entry
            continue

        zid, boss_slug = zone_for_table(chosen)
        if zid is None:
            continue  # unrecognized prefix (quest markers, vendors) -> leave unchanged

        if TRASH_RE.search(chosen) or not boss_slug:
            out[server_id] = {"zoneId": zid, "otherName": "Trash"}
            continue

        # Only override when the boss name matches a known NPC (real id + zone +
        # working Wowhead link). A non-boss table label ("Tome", "Druid" class-set
        # reward, ...) with no NPC match is left alone rather than guessing.
        npc = best_npc(_norm(boss_slug))
        if npc:
            out[server_id] = {"zoneId": npc["zoneId"] or zid, "npcId": npc["id"]}

    json.dump({str(k): v for k, v in sorted(out.items())}, open(OUT, "w"), indent=0)
    matched_npc = sum(1 for v in out.values() if "npcId" in v)
    print(f"wrote {OUT}: {len(out)} item sources ({matched_npc} matched a known NPC, "
          f"{len(out) - matched_npc} fell back to a plain name)")


if __name__ == "__main__":
    main()
