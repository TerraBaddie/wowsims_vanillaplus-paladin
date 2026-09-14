"""Parse CSV's/VPlusItemDB.lua (in-game tooltip dump for the private server) into
item stat overrides and merge-ready item entries for the sim database.

Output: assets/db_inputs/custom_items.json  -- an array of UIItem-shaped dicts
(camelCase keys, matching assets/database/db.json). Consumed by tools/database/gen_db.

The private server's item values are authoritative: for every item present in the
dump we emit a COMPLETE 44-length stats array (so the merge fully replaces whatever
the sim had) plus weapon damage/speed. Items not already in the sim DB are emitted
with enough shape (name/type/quality/etc.) to be usable.
"""
import collections
import json
import re
import sys

LUA = "CSV's/VPlusItemDB.lua"
DB = "assets/database/db.json"
OUT = "assets/db_inputs/custom_items.json"
ADD_ITEMS_FILE = "assets/db_inputs/add_items.json"
ATLASLOOT_SETS = "CSV's/AtlasLoot/Sets/sets.lua"


def atlasloot_icons():
    """{item_id: icon_string} from the AtlasLoot data tables, for new items."""
    icons = {}
    for path in (ATLASLOOT_SETS, "CSV's/AtlasLoot/Instances/instances.en.lua"):
        try:
            txt = open(path, encoding="utf8", errors="ignore").read()
        except OSError:
            continue
        for iid, icon in re.findall(r'\{\s*(\d+),\s*"([A-Za-z][^"]*)"', txt):
            if icon and int(iid) not in icons:
                icons[int(iid)] = icon.lower()
    return icons

# sim/core/proto Stat indices
S = {
    "Strength": 0, "Agility": 1, "Stamina": 2, "Intellect": 3, "Spirit": 4,
    "SpellPower": 5, "Arcane": 6, "Fire": 7, "Frost": 8, "Holy": 9, "Nature": 10, "Shadow": 11,
    "MP5": 12, "SpellHit": 13, "SpellCrit": 14, "AttackPower": 17, "MeleeHit": 18, "MeleeCrit": 19,
    "Defense": 28, "Block": 29, "BlockValue": 30, "Dodge": 31, "Parry": 32, "Health": 34,
    "ArcaneRes": 35, "FireRes": 36, "FrostRes": 37, "NatureRes": 38, "ShadowRes": 39,
    "RangedAttackPower": 27, "Armor": 26, "HealingPower": 41,
}
NSTATS = 44

ITEM_TYPE = {
    "Head": 1, "Neck": 2, "Shoulder": 3, "Back": 4, "Chest": 5, "Wrist": 6, "Hands": 7,
    "Waist": 8, "Legs": 9, "Feet": 10, "Finger": 11, "Trinket": 12,
}
ARMOR_TYPE = {"Cloth": 1, "Leather": 2, "Mail": 3, "Plate": 4}
WEAPON_TYPE = {
    "Axe": 1, "Dagger": 2, "Fist Weapon": 3, "Mace": 4, "Polearm": 6, "Shield": 7,
    "Staff": 8, "Sword": 9,
}
HAND_TYPE = {"Main Hand": 1, "One-Hand": 2, "Off Hand": 3, "Two-Hand": 4}
RANGED_TYPE = {"Bow": 1, "Crossbow": 2, "Gun": 3, "Wand": 8, "Thrown": 6}
QUALITY = {"Poor": 0, "Common": 1, "Uncommon": 2, "Rare": 3, "Epic": 4, "Legendary": 5}
SCHOOL = {"Arcane": "Arcane", "Fire": "Fire", "Frost": "Frost",
          "Holy": "Holy", "Nature": "Nature", "Shadow": "Shadow"}
CLASS = {"Druid": 1, "Hunter": 2, "Mage": 3, "Paladin": 4, "Priest": 5,
         "Rogue": 6, "Shaman": 7, "Warlock": 8, "Warrior": 9}

# Custom private-server armour sets that don't exist in the sim DB at all and must
# be added as new items. {set name (as it appears in the dump): {phase, ilvl}}.
NEW_SETS = {
    "Talonclaw Regalia": {"phase": 1, "ilvl": 66},   # druid
    "Ursoc Armor": {"phase": 1, "ilvl": 66},         # druid
    "Cataclysm Armor": {"phase": 1, "ilvl": 66},     # shaman
    "The Stonefury": {"phase": 1, "ilvl": 66},       # shaman
    "Righteous Armor": {"phase": 1, "ilvl": 66},     # paladin
}

# lua set name -> sim DB set name, for the set+slot matching pass.
SET_ALIASES = {
    "Cenarion Armor": "Cenarion Raiment",
}

# Every class tier set on the server is Tier 1 (MC/BWL) - there is no Tier 2.
# Force phase 1 on all pieces so they show together in the Phase 1 picker.
TIER1_SETS = {
    "Battlegear of Might", "Battlegear of Wrath",
    "Lawbringer Armor", "Righteous Armor", "Judgement Armor",
    "Giantstalker Armor", "Dragonstalker Armor",
    "Nightslayer Armor", "Bloodfang Armor",
    "Vestments of Prophecy", "Vestments of Transcendence",
    "The Earthfury", "Cataclysm Armor", "The Stonefury", "The Ten Storms",
    "Arcanist Regalia", "Netherwind Regalia",
    "Felheart Raiment", "Nemesis Raiment",
    "Cenarion Raiment", "Talonclaw Regalia", "Ursoc Armor", "Stormrage Raiment",
}


def parse_lua(path):
    """Return {item_id: [tooltip line strings]}.

    The dump repeats items across phase sections and (for reworked items) keeps
    both the old retail id and a new 25xxx/26xxx id under the same name. We keep,
    per id, the block with the most tooltip lines.
    """
    txt = open(path, encoding="utf8").read()
    parts = re.split(r'\n\t\t\t\[(\d+)\] = \{\n', txt)
    items = {}
    for i in range(1, len(parts), 2):
        sid = int(parts[i])
        lines = []
        for ln in parts[i + 1].split("\n"):
            if re.match(r'^\t\t\t\},?$', ln):
                break
            m = re.match(r'^\t\t\t\t\[\d+\] = "((?:[^"\\]|\\.)*)",?$', ln)
            if m:
                s = m.group(1).replace('\\"', '"').replace("\\n", "\n").strip()
                if s:
                    lines.append(s)
        if lines and len(lines) > len(items.get(sid, [])):
            items[sid] = lines
    return items


def parse_item(lines):
    stats = [0] * NSTATS
    out = {"stats": stats}
    name = lines[0]
    out["name"] = name
    for ln in lines[1:]:
        # ---- weapon damage ----
        m = re.match(r'^(\d+) - (\d+)(?: (\w+))? Damage, Speed ([\d.]+)$', ln)
        if m:
            lo, hi, school, spd = int(m.group(1)), int(m.group(2)), m.group(3), float(m.group(4))
            out["weaponDamageMin"] = float(lo)
            out["weaponDamageMax"] = float(hi)
            out["weaponSpeed"] = spd
            continue
        m = re.match(r'^\+? ?(\d+) - (\d+) (\w+) Damage$', ln)
        if m:  # bonus elemental "+ N - M Shadow Damage"
            continue
        # ---- primary stats ----
        m = re.match(r'^\+(\d+) (Strength|Agility|Stamina|Intellect|Spirit)$', ln)
        if m:
            stats[S[m.group(2)]] += int(m.group(1))
            continue
        m = re.match(r'^\+(\d+) (Arcane|Fire|Frost|Nature|Shadow) Resistance$', ln)
        if m:
            stats[S[m.group(2) + "Res"]] += int(m.group(1))
            continue
        m = re.match(r'^\+(\d+) All Resistances\.?$', ln)
        if m:
            for k in ("ArcaneRes", "FireRes", "FrostRes", "NatureRes", "ShadowRes"):
                stats[S[k]] += int(m.group(1))
            continue
        # ---- armor / block value ----
        m = re.match(r'^(\d+) Armor$', ln)
        if m:
            stats[S["Armor"]] += int(m.group(1))
            continue
        m = re.match(r'^(\d+) Block$', ln)
        if m:
            stats[S["BlockValue"]] += int(m.group(1))
            continue
        # ---- equip lines ----
        m = re.match(r'^Equip: Increases damage and healing done by magical spells and effects by up to (\d+)\.$', ln)
        if m:
            stats[S["SpellPower"]] += int(m.group(1))
            continue
        m = re.match(r'^Equip: Increases healing done by spells and effects by up to (\d+)\.$', ln)
        if m:
            stats[S["HealingPower"]] += int(m.group(1))
            continue
        m = re.match(r'^Equip: Increases damage done by (\w+) spells and effects by up to (\d+)\.$', ln)
        if m and m.group(1) in SCHOOL:
            stats[S[m.group(1)]] += int(m.group(2))
            continue
        m = re.match(r'^Equip: Restores (\d+) mana per \d+ sec\.$', ln)
        if m:
            stats[S["MP5"]] += int(m.group(1))
            continue
        m = re.match(r'^Equip: \+(\d+) Attack Power\.$', ln) or \
            re.match(r'^Equip: Increases attack power by (\d+)\.$', ln)
        if m:
            # In vanilla, generic "+Attack Power" raises both melee and ranged AP.
            stats[S["AttackPower"]] += int(m.group(1))
            stats[S["RangedAttackPower"]] += int(m.group(1))
            continue
        m = re.match(r'^Equip: \+(\d+) ranged Attack Power\.$', ln) or \
            re.match(r'^Equip: Increases ranged attack power by (\d+)\.$', ln)
        if m:
            stats[S["RangedAttackPower"]] += int(m.group(1))
            continue
        m = re.match(r'^Equip: Improves your chance to get a critical strike with spells by ([\d.]+)%\.$', ln)
        if m:
            stats[S["SpellCrit"]] += float(m.group(1))
            continue
        m = re.match(r'^Equip: Improves your chance to get a critical strike by ([\d.]+)%\.$', ln)
        if m:
            stats[S["MeleeCrit"]] += float(m.group(1))
            continue
        m = re.match(r'^Equip: Improves your chance to hit with spells by ([\d.]+)%\.$', ln)
        if m:
            stats[S["SpellHit"]] += float(m.group(1))
            continue
        m = re.match(r'^Equip: Improves your chance to hit by ([\d.]+)%\.$', ln)
        if m:
            stats[S["MeleeHit"]] += float(m.group(1))
            continue
        m = re.match(r'^Equip: Improves your chance to (?:get a critical strike|hit)(?: with)? (?:with )?attacks and spells by ([\d.]+)%\.$', ln)
        m2 = re.match(r'^Equip: Improves your chance to hit with attacks and spells by ([\d.]+)%\.$', ln)
        if m2:
            stats[S["MeleeHit"]] += float(m2.group(1))
            stats[S["SpellHit"]] += float(m2.group(1))
            continue
        m2 = re.match(r'^Equip: Improves your chance to get a critical strike with attacks and spells by ([\d.]+)%\.$', ln)
        if m2:
            stats[S["MeleeCrit"]] += float(m2.group(1))
            stats[S["SpellCrit"]] += float(m2.group(1))
            continue
        m = re.match(r'^Equip: Increased Defense \+(\d+)\.$', ln)
        if m:
            stats[S["Defense"]] += int(m.group(1))
            continue
        m = re.match(r'^Equip: Increases your chance to dodge an attack by ([\d.]+)%\.$', ln)
        if m:
            stats[S["Dodge"]] += float(m.group(1))
            continue
        m = re.match(r'^Equip: Increases your chance to parry an attack by ([\d.]+)%\.$', ln)
        if m:
            stats[S["Parry"]] += float(m.group(1))
            continue
        m = re.match(r'^Equip: Increases your chance to block attacks with a shield by ([\d.]+)%\.$', ln)
        if m:
            stats[S["Block"]] += float(m.group(1))
            continue
        m = re.match(r'^Equip: Increases the block value of your shield by (\d+)\.$', ln)
        if m:
            stats[S["BlockValue"]] += int(m.group(1))
            continue
        m = re.match(r'^Equip: Decreases the magical resistances of your spell targets by (\d+)\.$', ln)
        if m:
            stats[16] += int(m.group(1))  # SpellPenetration
            continue
        # ---- meta ----
        m = re.match(r'^Rarity: (\w+)$', ln)
        if m and m.group(1) in QUALITY:
            out["quality"] = QUALITY[m.group(1)]
            continue
        if ln == "Unique" or re.match(r'^Unique \(\d+\)$', ln):
            out["unique"] = True
            continue
        m = re.match(r'^Requires Level (\d+)$', ln)
        if m:
            out["_reqLevel"] = int(m.group(1))
            continue
        m = re.match(r'^Classes: (.+)$', ln)
        if m:
            out["classAllowlist"] = sorted(
                CLASS[c.strip()] for c in m.group(1).split(",") if c.strip() in CLASS)
            continue
        # ---- set membership: "<Set Name> (0/8)" (not indented) ----
        m = re.match(r'^([A-Z][^\n]+?) \(\d+/\d+\)$', ln)
        if m and "setName" not in out:
            out["setName"] = m.group(1).strip()
            continue
        # ---- slot / type ----
        parsed_slot(ln, out)
    return out


def parsed_slot(ln, out):
    if ln in ITEM_TYPE:
        out["type"] = ITEM_TYPE[ln]
        return
    if ln in ("Held In Off-hand",):
        out["type"] = 13
        out["weaponType"] = 5
        out["handType"] = 3
        return
    m = re.match(r'^(Head|Neck|Shoulder|Back|Chest|Wrist|Hands|Waist|Legs|Feet|Finger|Trinket), (Cloth|Leather|Mail|Plate)$', ln)
    if m:
        out["type"] = ITEM_TYPE[m.group(1)]
        out["armorType"] = ARMOR_TYPE[m.group(2)]
        return
    m = re.match(r'^(Main Hand|One-Hand|Off Hand|Two-Hand), (Axe|Dagger|Fist Weapon|Mace|Polearm|Shield|Staff|Sword)$', ln)
    if m:
        out["type"] = 13
        out["handType"] = HAND_TYPE[m.group(1)]
        out["weaponType"] = WEAPON_TYPE[m.group(2)]
        return
    m = re.match(r'^(Ranged|Projectile|Thrown), (Bow|Crossbow|Gun|Wand|Thrown|Arrow|Bullet)$', ln)
    if m and m.group(2) in RANGED_TYPE:
        out["type"] = 14
        out["rangedWeaponType"] = RANGED_TYPE[m.group(2)]
        return
    if ln in ("Bow", "Gun", "Crossbow", "Wand"):
        out["type"] = 14
        out["rangedWeaponType"] = RANGED_TYPE[ln]
        return


Q_COLOR = {0: "#9d9d9d", 1: "#ffffff", 2: "#1eff00", 3: "#0070dd", 4: "#a335ee", 5: "#ff8000"}
_GREEN_PREFIX = ("Equip:", "Use:", "Chance on hit:", "Chance on being hit:",
                 "Chance on being struck:")


def html_escape(s):
    return s.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")


def build_tooltip(lines, quality):
    name = html_escape(lines[0])
    color = Q_COLOR.get(quality if quality is not None else 1, "#ffffff")
    parts = [f'<div class="whtt-name" style="color:{color}">{name}</div>']
    for ln in lines[1:]:
        if ln.strip() in ("", "\n"):
            continue
        if re.match(r'^\(\d+\) Set:', ln) or ln.startswith("  ") or re.match(r'^.+\(\d+/\d+\)$', ln):
            style = "color:#9d9d9d"  # set list / set bonuses
        elif ln.startswith(_GREEN_PREFIX):
            style = "color:#1eff00"
        elif ln.startswith(('"', 'Requires ', 'Classes:', 'Races:')):
            style = "color:#9d9d9d"
        else:
            style = "color:#ffffff"
        parts.append(f'<div style="{style}">{html_escape(ln)}</div>')
    return "".join(parts)


def _entry_for(iid, lines):
    it = parse_item(lines)
    entry = {"id": iid, "stats": it["stats"],
             "tooltip": build_tooltip(lines, it.get("quality"))}
    for k in ("weaponDamageMin", "weaponDamageMax", "weaponSpeed"):
        if k in it:
            entry[k] = it[k]
    return entry, it


def main():
    lua = parse_lua(LUA)
    icons = atlasloot_icons()
    # Match against the pristine (pre-removal) DB so overrides are generated for
    # items that a previous gen_db run stripped out (e.g. tier-2 sets).
    try:
        import subprocess
        db = json.loads(subprocess.check_output(
            ["git", "show", "HEAD:assets/database/db.json"]))
    except Exception:
        db = json.load(open(DB))
    db_items = db["items"]
    have = {it["id"] for it in db_items}

    # The private server renumbered many items (esp. tier/PvP sets) to 25xxx/26xxx
    # ids that the sim DB doesn't use. Match those onto the sim item by NAME.
    # Prefer the highest lua id for a given name (the renumbered / current version).
    lua_by_name = {}
    lua_parsed = {}
    lua_by_setslot = {}
    setslot_dupes = set()
    for iid, lines in lua.items():
        nm = lines[0]
        if nm not in lua_by_name or iid > lua_by_name[nm]:
            lua_by_name[nm] = iid
        pit = parse_item(lines)
        lua_parsed[iid] = pit
        if pit.get("setName") and pit.get("type"):
            sn = SET_ALIASES.get(pit["setName"], pit["setName"])
            key = (sn, pit["type"])
            if key in lua_by_setslot and lua_by_setslot[key] != iid:
                setslot_dupes.add(key)
            lua_by_setslot.setdefault(key, iid)
    db_name_counts = collections.Counter(it["name"] for it in db_items)
    db_setslot_counts = collections.Counter(
        (it["setName"], it["type"]) for it in db_items if it.get("setName"))

    # (type, armorType, classes) -> lua id, for items the dump marks as set pieces.
    # Handles PvP sets whose set NAME the server also changed.
    def shape_key(pit_or_it):
        return (pit_or_it.get("type"), pit_or_it.get("armorType"),
                tuple(pit_or_it.get("classAllowlist") or ()))
    lua_by_shape = {}
    shape_dupes = set()
    for iid, pit in lua_parsed.items():
        if not pit.get("setName") or not pit.get("type") or not pit.get("classAllowlist"):
            continue
        k = shape_key(pit)
        if k in lua_by_shape and lua_by_shape[k] != iid:
            shape_dupes.add(k)
        lua_by_shape.setdefault(k, iid)
    db_shape_counts = collections.Counter(
        shape_key(it) for it in db_items if it.get("setName") and it.get("classAllowlist"))

    out = []
    used_db_ids = set()
    renumber = {}   # {old sim id: new server id} for pieces the server renumbered
    n_id = n_name = n_setslot = n_shape = n_new = 0
    ambiguous = []

    def emit_renumbered(old_id, lua_id, lines):
        """Emit the override under the server's (dump) id and record old->new."""
        entry, _ = _entry_for(lua_id, lines)
        out.append(entry)
        used_db_ids.add(old_id)
        if lua_id != old_id:
            renumber[old_id] = lua_id

    # 1. Direct id matches (sim id present verbatim in the dump). Skip when the dump
    #    also carries a higher "server" id (>=24000) for the same name -- that means
    #    the server renumbered the piece and pass 2 should map old -> new instead.
    for iid, lines in sorted(lua.items()):
        if iid not in have:
            continue
        newer = lua_by_name.get(lines[0])
        if newer is not None and newer != iid and newer >= 24000 and newer not in have:
            continue
        entry, _ = _entry_for(iid, lines)
        out.append(entry)
        used_db_ids.add(iid)
        n_id += 1

    # 2. Name matches for sim items the dump doesn't cover by id.
    for it in db_items:
        if it["id"] in used_db_ids:
            continue
        lua_id = lua_by_name.get(it["name"])
        if lua_id is None or lua_id in have:
            continue
        if db_name_counts[it["name"]] > 1:
            ambiguous.append((it["id"], it["name"]))
            continue
        emit_renumbered(it["id"], lua_id, lua[lua_id])
        n_name += 1

    # 2b. Set-name + slot matches (the server renamed many set pieces, so
    #     "Legplates of Might" in the sim is "Legguards of Might" in the dump, but
    #     both are the Legs piece of "Battlegear of Might").
    for it in db_items:
        if it["id"] in used_db_ids or not it.get("setName"):
            continue
        key = (it["setName"], it["type"])
        lua_id = lua_by_setslot.get(key)
        if lua_id is None or lua_id in have or lua_id in used_db_ids:
            continue
        if db_setslot_counts[key] > 1 or key in setslot_dupes:
            ambiguous.append((it["id"], it["name"] + f"  [{it['setName']} slot {it['type']}]"))
            continue
        emit_renumbered(it["id"], lua_id, lua[lua_id])
        n_setslot += 1

    # 2c. Set piece match by shape (type + armor + classes) for sets whose NAME
    #     the server also changed (PvP Champion's / Lieutenant Commander's sets).
    for it in db_items:
        if it["id"] in used_db_ids or not it.get("setName") or not it.get("classAllowlist"):
            continue
        k = shape_key(it)
        lua_id = lua_by_shape.get(k)
        if lua_id is None or lua_id in have or lua_id in used_db_ids:
            continue
        if db_shape_counts[k] > 1 or k in shape_dupes:
            ambiguous.append((it["id"], it["name"] + f"  [{it['setName']} shape {k}]"))
            continue
        emit_renumbered(it["id"], lua_id, lua[lua_id])
        n_shape += 1

    # 2d. PvP sets: the server renamed both the set AND the pieces. Match by
    #     rank prefix + armor type + slot (unique within a rank/armor group).
    PVP_PREFIXES = ("Lieutenant Commander's ", "Champion's ", "Knight-Lieutenant's ",
                    "Blood Guard's ", "Legionnaire's ", "First Sergeant's ")

    def pvp_prefix(name):
        for p in PVP_PREFIXES:
            if name.startswith(p):
                return p.strip()
        return None

    def pvp_key(pfx, it):
        return (pfx, it.get("armorType"), it.get("type"),
                tuple(it.get("classAllowlist") or ()))
    lua_pvp = {}
    lua_pvp_dupes = set()
    for iid, pit in lua_parsed.items():
        pfx = pvp_prefix(pit["name"])
        if not pfx or not pit.get("type") or not pit.get("armorType"):
            continue
        k = pvp_key(pfx, pit)
        if k in lua_pvp and lua_pvp[k] != iid:
            lua_pvp_dupes.add(k)
        lua_pvp.setdefault(k, iid)
    db_pvp_counts = collections.Counter(
        pvp_key(pvp_prefix(it["name"]), it)
        for it in db_items if pvp_prefix(it["name"]))
    for it in db_items:
        if it["id"] in used_db_ids:
            continue
        pfx = pvp_prefix(it["name"])
        if not pfx:
            continue
        k = pvp_key(pfx, it)
        lua_id = lua_pvp.get(k)
        if lua_id is None or lua_id in have or lua_id in used_db_ids:
            continue
        if db_pvp_counts[k] > 1 or k in lua_pvp_dupes:
            ambiguous.append((it["id"], it["name"] + f"  [pvp {k}]"))
            continue
        emit_renumbered(it["id"], lua_id, lua[lua_id])
        n_shape += 1

    # 3. Brand-new items the sim DB is missing entirely:
    #    (a) custom private-server armour sets (NEW_SETS), and
    #    (b) individual raid/dungeon drops from add_items.json (docs/gen_add_items.py).
    try:
        add_items = {int(k): v for k, v in json.load(open(ADD_ITEMS_FILE)).items()}
    except OSError:
        add_items = {}
    renumber_targets = set(renumber.values())
    for iid, lines in sorted(lua.items()):
        if iid in have or iid in used_db_ids or iid in renumber_targets:
            continue
        entry, it = _entry_for(iid, lines)
        if "type" not in it:
            continue
        cfg = NEW_SETS.get(it.get("setName"))
        if cfg is not None:
            entry["phase"], entry["ilvl"] = cfg["phase"], cfg["ilvl"]
            if it.get("setName"):
                entry["setName"] = it["setName"]
        elif iid in add_items:
            entry["phase"] = add_items[iid]
            entry["ilvl"] = {1: 66, 2: 66, 3: 76, 4: 80}.get(add_items[iid], 66)
        else:
            continue
        entry["name"] = it["name"]
        if iid in icons:
            entry["icon"] = icons[iid]
        for k in ("quality", "unique", "type", "armorType", "weaponType",
                  "handType", "rangedWeaponType", "classAllowlist"):
            if k in it:
                entry[k] = it[k]
        entry.setdefault("quality", 4)
        out.append(entry)
        n_new += 1

    # Item phases are assigned by docs/gen_phases.py (location-based), applied in gen_db.

    json.dump({"items": out}, open(OUT, "w"), indent=1)
    renum_path = "assets/db_inputs/renumber.json"
    json.dump({str(k): v for k, v in sorted(renumber.items())},
              open(renum_path, "w"), indent=0)
    print(f"wrote {OUT}: {n_id} by id, {n_name} by name, {n_setslot} by set+slot, "
          f"{n_shape} by shape, {n_new} new items")
    print(f"wrote {renum_path}: {len(renumber)} renumbered to server ids")
    if ambiguous:
        print(f"  {len(ambiguous)} skipped (name maps to multiple sim items):")
        for iid, nm in ambiguous:
            print(f"    {iid} {nm}")


if __name__ == "__main__":
    main()
