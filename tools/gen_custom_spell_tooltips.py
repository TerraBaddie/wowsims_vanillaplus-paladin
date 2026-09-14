#!/usr/bin/python
"""Append tooltip rows for custom talent spells to assets/db_inputs/wowhead_spell_tooltips.csv.

Custom talent spell IDs (from Talent.dbc) don't exist on Classic Wowhead, so gen_db
can't find an icon/name for them and the talent renders blank. This reads the name +
SpellIconID from Spell.csv and resolves the icon slug via SpellIcon.csv, then writes
minimal tooltip rows so gen_db picks them up.

Usage:
  python tools/gen_custom_spell_tooltips.py priest
  python tools/gen_custom_spell_tooltips.py --all
Then regenerate the DB:
  go run ./tools/database/gen_db -outDir=./assets -gen=db
"""
import argparse
import csv
import json
import os
import re
import sys

CLASSES = ["warrior", "paladin", "hunter", "rogue", "priest", "shaman", "mage", "warlock", "druid"]

# Spell.csv column indices (0-based). This dump has no header row; positions were
# reverse-engineered from spells with known values (SW:Pain, Renew, Blur, ...).
COL_NAME = 120
COL_DESC = 138
COL_ICON = 117
COL_PROC = 25          # ProcChance            -> $h
COL_STACKS = 39        # StackAmount           -> $u
COL_DURIDX = 30        # DurationIndex          -> $d (via SpellDuration.csv)
COL_BP = (76, 77, 78)  # EffectBasePoints[1..3] -> $s1/$s2/$s3 (abs value + 1)
COL_AMP = (94, 95, 96)  # EffectAmplitude[1..3] -> $t1.. and feeds $o1..
COL_TRIG = (109, 110, 111)  # EffectTriggerSpell[1..3] (context for $<spellId>.. refs)

FALLBACK_ICON = "inv_misc_questionmark"

# $g male:female; / $l singular:plural; grammar switches -> keep the first option.
GENDER_RE = re.compile(r"\$[glGL]\s*([^:;]*):[^;]*;")
# $<spellId>?  $/<divisor>;?  <letter><index>?   e.g. $s1  $d  $/2;s1  $33807s1  $t1
TOKEN_RE = re.compile(r"\$(\d+)?(?:/(\d+);)?([a-zA-Z])(\d)?")


def fmt_duration(ms):
    if ms is None or ms < 0:
        return None
    if ms >= 60000 and ms % 60000 == 0:
        return f"{ms // 60000} min"
    sec = ms / 1000
    return f"{sec:g} sec"


def resolve_desc(text, sid, spells, durations):
    """Substitute Wowhead $ tokens using the reverse-engineered Spell.csv data."""
    if not text:
        return ""
    text = GENDER_RE.sub(lambda m: m.group(1).strip(), text)
    text = text.replace("$b", " ").replace("$B", " ")

    def repl(m):
        ref, divisor, letter, idx = m.group(1), m.group(2), m.group(3).lower(), m.group(4)
        ctx = int(ref) if ref else sid
        sp = spells.get(ctx)
        i = (int(idx) - 1) if idx else 0
        # Blizzard's $s = EffectBasePoints + 1 (min roll, DieSides assumed 1 here);
        # display magnitude only, since the surrounding text carries the sign
        # ("increases by" / "reduced by").
        val = None
        if sp:
            if letter in "sm" and 0 <= i < 3:
                val = abs(sp["bp"][i] + 1)
            elif letter == "t" and 0 <= i < 3:
                val = sp["amp"][i] / 1000
            elif letter == "o" and 0 <= i < 3:
                base = abs(sp["bp"][i] + 1)
                dur = durations.get(sp["dur_idx"])
                amp = sp["amp"][i]
                val = base * (dur[0] / amp) if (dur and amp) else base
            elif letter == "h":
                val = sp["proc"]
            elif letter == "u":
                val = sp["stacks"]
            elif letter == "d":
                dur = durations.get(sp["dur_idx"])
                return fmt_duration(dur[0]) if dur else m.group(0)
        if val is None:
            return "X"
        if divisor:
            d = int(divisor)
            val = val / d
            if d != 1000:  # $/1000; is a ms->sec conversion, keep the decimals
                val = int(val)
        val = round(val, 2)
        return f"{val:g}"

    text = TOKEN_RE.sub(repl, text)
    return re.sub(r"\s+", " ", text).strip()

# Icon IDs referenced by custom spells but absent from the Classic-era SpellIcon.csv
# (they point at art added in later expansions). Map them to a zamimg slug by hand.
ICON_ID_OVERRIDES = {
    4433: "spell_shadow_unholyfrenzy",       # Insanity
    4404: "spell_nature_healingwavegreater", # HealingWaveOld - not hosted on zamimg
}


def _int(s):
    try:
        s = s.strip()
        return int(s, 16) if s[:2].lower() == "0x" else int(s)
    except (TypeError, ValueError, AttributeError):
        return 0


def load_durations(path):
    out = {}
    if os.path.exists(path):
        for r in csv.reader(open(path, encoding="utf-8")):
            if r and r[0].isdigit():
                out[int(r[0])] = (_int(r[1]), _int(r[2]) if len(r) > 2 else 0,
                                  _int(r[3]) if len(r) > 3 else 0)
    return out


def load_spell_csv(path):
    """Return names, icon-ids, raw descriptions, and a compact per-spell data record."""
    names, icons, raw_descs, data = {}, {}, {}, {}
    for r in csv.reader(open(path, encoding="utf-8")):
        if not r or not r[0].isdigit():
            continue
        sid = int(r[0])
        # A few rows in this dump are shifted right by one; detect via the name
        # column holding junk (empty / "0" / a stray number) with a real name next.
        shift = 0
        nm = r[COL_NAME] if len(r) > COL_NAME else ""
        nxt = r[COL_NAME + 1] if len(r) > COL_NAME + 1 else ""
        if (not nm or nm == "0" or (nm.isdigit() and any(c.isalpha() for c in nxt))) and nxt and nxt != "0":
            shift = 1

        def col(idx, _r=r, _s=shift):
            j = idx + _s
            return _r[j] if 0 <= j < len(_r) else ""

        names[sid] = col(COL_NAME)
        raw_descs[sid] = col(COL_DESC)
        if col(COL_ICON).isdigit():
            icons[sid] = int(col(COL_ICON))
        data[sid] = {
            "bp": [_int(col(c)) for c in COL_BP],
            "amp": [_int(col(c)) for c in COL_AMP],
            "proc": _int(col(COL_PROC)),
            "stacks": _int(col(COL_STACKS)),
            "dur_idx": _int(col(COL_DURIDX)),
            "trig": [_int(col(c)) for c in COL_TRIG],
        }
    return names, icons, raw_descs, data


def load_spell_icon_csv(path):
    out = {}
    for r in csv.reader(open(path, encoding="utf-8")):
        if not r or not r[0].isdigit():
            continue
        slug = r[1].replace("\\", "/").split("/")[-1].strip().lower()
        out[int(r[0])] = slug
    return out


def existing_ids(path):
    ids = set()
    if os.path.exists(path):
        for line in open(path, encoding="utf-8"):
            i = line.find(",")
            if i > 0 and line[:i].isdigit():
                ids.add(int(line[:i]))
    return ids


def talent_spell_ranks(json_path):
    """Yield (spellId, rank) for every talent rank, expanding short rank lists like gen_db."""
    for tree in json.load(open(json_path, encoding="utf-8")):
        for t in tree["talents"]:
            sids = list(t["spellIds"])
            while len(sids) < t.get("maxPoints", len(sids)):
                sids.append(sids[-1])
            for rank, sid in enumerate(sids, start=1):
                yield sid, rank


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("classes", nargs="*")
    ap.add_argument("--all", action="store_true")
    ap.add_argument("--csv-dir", default="CSV's")
    ap.add_argument("--repo", default=".")
    ap.add_argument("--dry-run", action="store_true")
    args = ap.parse_args()

    targets = CLASSES if args.all else args.classes
    if not targets:
        ap.error("pass class names or --all")

    csv_dir = os.path.join(args.repo, args.csv_dir)
    names, icon_ids, raw_descs, spell_data = load_spell_csv(os.path.join(csv_dir, "Spell.csv"))
    durations = load_durations(os.path.join(csv_dir, "SpellDuration.csv"))
    icon_slugs = load_spell_icon_csv(os.path.join(csv_dir, "SpellIcon.csv"))

    def desc_for(sid):
        raw = raw_descs.get(sid, "")
        # Skip empty / truncated-row garbage (bare flag values like "0x3F007E").
        if not raw or raw == "0" or re.fullmatch(r"0x[0-9A-Fa-f]+", raw):
            return ""
        return resolve_desc(raw, sid, spell_data, durations)
    tooltip_path = os.path.join(args.repo, "assets/db_inputs/wowhead_spell_tooltips.csv")
    record_path = os.path.join(args.repo, "tools/custom_talent_spell_ids.txt")
    have = existing_ids(tooltip_path)

    # Spell ids we've previously classified as custom (stay custom even once their
    # tooltip row exists), so the run is idempotent.
    recorded = set()
    if os.path.exists(record_path):
        for line in open(record_path, encoding="utf-8"):
            line = line.split("#")[0].strip()
            if line.isdigit():
                recorded.add(int(line))

    # Every talent spell id (all ranks) for the target classes. The DBC is
    # authoritative, so we (re)write a clean tooltip row for ALL of them - not just
    # the ones missing from the Wowhead scrape - so buff/debuff/talent icons that
    # point at these spells show the DBC values everywhere.
    want = {}
    for cn in targets:
        jp = os.path.join(args.repo, "ui/core/talents/trees", cn + ".json")
        for sid, rank in talent_spell_ranks(jp):
            if sid not in want or rank < want[sid][1]:
                want[sid] = (cn, rank)

    # Track which ones Wowhead has no data for (icons still need a db.bin entry).
    recorded |= {sid for sid in want if sid not in have}

    warns = []
    rows_by_id = {}  # sid -> tooltip JSON string (regenerated for every talent spell)
    for sid, (cn, rank) in sorted(want.items()):
        name = names.get(sid, "")
        if not name or name == "0":
            name = f"Spell {sid}"
            warns.append(f"  {sid}: no name in Spell.csv")
        icon = ICON_ID_OVERRIDES.get(icon_ids.get(sid, -1)) or icon_slugs.get(icon_ids.get(sid, -1))
        if not icon:
            icon = FALLBACK_ICON
            warns.append(f"  {sid} ({name}): iconId {icon_ids.get(sid)} not in SpellIcon.csv -> fallback")
        desc = desc_for(sid)
        tooltip = f'<b class="whtt-name">{name}</b><br />Rank {rank}' + (f"<br />{desc}" if desc else "")
        payload = {
            "name": name, "quality": None, "icon": icon,
            "tooltip": tooltip, "tooltip2": "", "buff": "",
            "spells": {}, "buffspells": {}, "completion_category": 0,
        }
        rows_by_id[sid] = json.dumps(payload, separators=(",", ":"))

    added = sorted(sid for sid in rows_by_id if sid not in have)
    refreshed = sorted(sid for sid in rows_by_id if sid in have)
    print(f"{len(added)} new + {len(refreshed)} refreshed tooltip rows for: {', '.join(targets)}")
    for sid in added:
        print("  +" + f"{sid},{rows_by_id[sid]}"[:150])
    if warns:
        sys.stderr.write("WARNINGS:\n" + "\n".join(warns) + "\n")

    if args.dry_run:
        return

    with open(record_path, "w", encoding="utf-8") as f:
        f.write("# Talent spell ids with no Classic Wowhead data - rendered with local tooltips.\n")
        f.write("# Maintained by tools/gen_custom_spell_tooltips.py\n")
        for sid in sorted(recorded):
            f.write(f"{sid}\n")

    # Rewrite existing custom rows in place, append the new ones at the end.
    lines = open(tooltip_path, encoding="utf-8").read().splitlines()
    for n, line in enumerate(lines):
        j = line.find(",")
        if j > 0 and line[:j].isdigit() and int(line[:j]) in rows_by_id:
            lines[n] = f"{line[:j]},{rows_by_id[int(line[:j])]}"
    for sid in added:
        lines.append(f"{sid},{rows_by_id[sid]}")
    with open(tooltip_path, "w", encoding="utf-8", newline="") as f:
        f.write("\n".join(lines) + "\n")
    print(f"Wrote {tooltip_path} ({len(added)} appended, {len(refreshed)} updated)")

    # Patch each tree JSON: EVERY talent gets a per-rank `descriptions` array built
    # from the DBC (one resolved string per rank). The DBC is authoritative - the
    # picker renders these locally instead of the (stale) Wowhead tooltip.
    for cn in targets:
        jp = os.path.join(args.repo, "ui/core/talents/trees", cn + ".json")
        trees = json.load(open(jp, encoding="utf-8"))
        changed = 0
        for tree in trees:
            for t in tree["talents"]:
                sids = t.get("spellIds") or []
                if not sids:
                    continue
                ranks = [sids[min(i, len(sids) - 1)] for i in range(t.get("maxPoints", len(sids)))]
                ds = [desc_for(s) for s in ranks]
                if not any(ds):
                    # No usable DBC text - leave the talent to its Wowhead tooltip.
                    if t.pop("descriptions", None) is not None or t.pop("description", None) is not None:
                        changed += 1
                    continue
                if t.get("descriptions") != ds:
                    t["descriptions"] = ds
                    changed += 1
                t.pop("description", None)
        if changed:
            with open(jp, "w", encoding="utf-8") as f:
                json.dump(trees, f, indent=2)
                f.write("\n")
            print(f"Patched {changed} descriptions into {jp}")


if __name__ == "__main__":
    main()
