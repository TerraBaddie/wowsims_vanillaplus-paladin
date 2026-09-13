#!/usr/bin/python
"""Correct tooltip VALUES (not names/icons) in wowhead_spell_tooltips.csv from Spell.csv.

This repo's item pipeline already pulls stats from the private server's DBC dump
instead of Wowhead (see docs/private-server-item-rules.md). This script does the
same thing for spell tooltips: for every spell id the sim actually references
(talents + SharedSpellsIcons + every `ActionID{SpellID: N}` literal in sim/**/*.go),
it resolves the description text from CSV's/Spell.csv and, when it differs from what
is currently cached from Wowhead, rewrites ONLY the descriptive `<div class="q">...`
block inside the cached tooltip HTML. The name and icon fields (and everything else
in the row - mana cost, cast time, range, requirements) are left exactly as Wowhead
scraped them.

Spell ids that don't exist on Wowhead at all are appended as minimal DBC-only rows,
same as tools/gen_custom_spell_tooltips.py already does for talent-only spells - this
script's job is everything BESIDES talents (that script still owns talents and the
per-rank `descriptions` arrays it patches into ui/core/talents/trees/*.json).

Usage:
  python tools/gen_spell_value_overrides.py
  python tools/gen_spell_value_overrides.py --dry-run
Then regenerate the DB:
  go run ./tools/database/gen_db -outDir=./assets -gen=db
"""
import argparse
import json
import os
import re
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gen_custom_spell_tooltips import (  # noqa: E402
    CLASSES, FALLBACK_ICON, ICON_ID_OVERRIDES, existing_ids, load_durations,
    load_spell_csv, load_spell_icon_csv, talent_spell_ranks, resolve_desc,
)

DESC_DIV_RE = re.compile(r'(<div class=\\"q\\">)(.*?)(</div>)')
TAG_RE = re.compile(r"<[^>]+>")


def find_talent_ids(repo):
    ids = set()
    for cn in CLASSES:
        jp = os.path.join(repo, "ui/core/talents/trees", cn + ".json")
        if not os.path.exists(jp):
            continue
        for sid, _rank in talent_spell_ranks(jp):
            ids.add(sid)
    return ids


def find_shared_spell_icon_ids(repo):
    """Parse the SharedSpellsIcons []int32{...} literal out of overrides.go."""
    path = os.path.join(repo, "tools/database/overrides.go")
    ids = set()
    in_block = False
    for line in open(path, encoding="utf-8"):
        if line.strip().startswith("var SharedSpellsIcons"):
            in_block = True
            continue
        if in_block:
            if line.strip().startswith("}"):
                break
            m = re.match(r"\s*(\d+)\s*,", line)
            if m:
                ids.add(int(m.group(1)))
    return ids


def find_go_action_spell_ids(repo):
    """Every literal `SpellID: N` in sim/**/*.go. ActionID{SpellID, ItemID, OtherID,
    Tag} only ever uses this field name for spells, so no ItemID/OtherID confusion."""
    ids = set()
    sim_dir = os.path.join(repo, "sim")
    pat = re.compile(r"SpellID:\s*(\d+)")
    for root, _dirs, files in os.walk(sim_dir):
        for fn in files:
            if not fn.endswith(".go"):
                continue
            text = open(os.path.join(root, fn), encoding="utf-8", errors="ignore").read()
            for m in pat.finditer(text):
                ids.add(int(m.group(1)))
    return ids


def resolved_ok(desc):
    """desc_for() emits a bare 'X' where a $ token couldn't be resolved (missing/zero
    Spell.csv data for that context), and leaves the literal '$...' token in place when
    e.g. a duration index has no SpellDuration.csv row. Either case means our column
    reverse-engineering doesn't cover this spell - never apply a description with one,
    since applying it would corrupt an otherwise-good Wowhead tooltip."""
    if not desc or "$" in desc:
        return False
    return "X" not in re.split(r"[\s.,;:()]+", desc)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--csv-dir", default="CSV's")
    ap.add_argument("--repo", default=".")
    ap.add_argument("--dry-run", action="store_true")
    args = ap.parse_args()

    repo = args.repo
    csv_dir = os.path.join(repo, args.csv_dir)
    names, icon_ids, raw_descs, spell_data = load_spell_csv(os.path.join(csv_dir, "Spell.csv"))
    durations = load_durations(os.path.join(csv_dir, "SpellDuration.csv"))
    icon_slugs = load_spell_icon_csv(os.path.join(csv_dir, "SpellIcon.csv"))

    def desc_for(sid):
        raw = raw_descs.get(sid, "")
        if not raw or raw == "0" or re.fullmatch(r"0x[0-9A-Fa-f]+", raw):
            return ""
        # The $o (amount-over-duration) formula was only validated against
        # periodic-damage/-heal spells (SW:Pain, Renew, ...). A spot check found it
        # badly wrong for mana-regen-over-time spells (e.g. Drink: DBC-implied 42
        # mana vs. Wowhead's known-correct 151.2) - EffectBasePoints likely means
        # something different for that periodic-energize aura type. Until that's
        # reverse-engineered separately, don't apply $o corrections to mana-over-time
        # text so we don't silently corrupt those tooltips.
        if "$o" in raw and "mana" in raw.lower():
            return ""
        return resolve_desc(raw, sid, spell_data, durations)

    tooltip_path = os.path.join(repo, "assets/db_inputs/wowhead_spell_tooltips.csv")
    record_path = os.path.join(repo, "tools/custom_all_spell_ids.txt")
    talent_record_path = os.path.join(repo, "tools/custom_talent_spell_ids.txt")

    talent_ids = find_talent_ids(repo)
    already_custom_talent = set()
    if os.path.exists(talent_record_path):
        for line in open(talent_record_path, encoding="utf-8"):
            line = line.split("#")[0].strip()
            if line.isdigit():
                already_custom_talent.add(int(line))

    all_ids = find_go_action_spell_ids(repo) | find_shared_spell_icon_ids(repo) | talent_ids
    # Talents (including their already-missing-from-Wowhead ids) stay owned by
    # gen_custom_spell_tooltips.py, which also patches per-rank descriptions into the
    # talent tree JSON. Don't double-handle them here.
    target_ids = sorted(all_ids - talent_ids)

    have = existing_ids(tooltip_path)
    lines = open(tooltip_path, encoding="utf-8").read().splitlines()
    line_idx = {}
    for n, line in enumerate(lines):
        j = line.find(",")
        if j > 0 and line[:j].isdigit():
            line_idx[int(line[:j])] = n

    patched, added, skipped_unresolved, no_desc = [], [], [], []
    recorded = set(already_custom_talent)  # keep talent ids untouched, just don't re-add
    new_custom = set()

    for sid in target_ids:
        desc = desc_for(sid)
        if sid in have:
            n = line_idx[sid]
            line = lines[n]
            j = line.find(",")
            payload = line[j + 1:]
            m = DESC_DIV_RE.search(payload)
            if not m:
                continue  # no description block to correct (e.g. a passive with no `q` div)
            current_text = TAG_RE.sub("", m.group(2)).strip()
            if not desc:
                no_desc.append(sid)
                continue
            if not resolved_ok(desc):
                skipped_unresolved.append(sid)
                continue
            if current_text == desc:
                continue
            new_payload = payload[:m.start()] + m.group(1) + desc + m.group(3) + payload[m.end():]
            # Sanity: must still be valid JSON with name/icon unchanged.
            try:
                old_obj = json.loads(payload)
                new_obj = json.loads(new_payload)
            except json.JSONDecodeError:
                continue
            if new_obj.get("name") != old_obj.get("name") or new_obj.get("icon") != old_obj.get("icon"):
                continue
            lines[n] = f"{sid},{new_payload}"
            patched.append(sid)
        else:
            # Not on Wowhead at all - same minimal-row fallback as the talent script.
            name = names.get(sid, "")
            if not name or name == "0":
                no_desc.append(sid)
                continue
            icon = ICON_ID_OVERRIDES.get(icon_ids.get(sid, -1)) or icon_slugs.get(icon_ids.get(sid, -1)) or FALLBACK_ICON
            tooltip = f'<b class="whtt-name">{name}</b>' + (f"<br />{desc}" if resolved_ok(desc) else "")
            obj = {
                "name": name, "quality": None, "icon": icon,
                "tooltip": tooltip, "tooltip2": "", "buff": "",
                "spells": {}, "buffspells": {}, "completion_category": 0,
            }
            lines.append(f"{sid},{json.dumps(obj, separators=(',', ':'))}")
            added.append(sid)
            new_custom.add(sid)

    recorded |= new_custom
    # Keep previously-recorded non-talent custom ids sticky across runs too.
    if os.path.exists(record_path):
        for line_ in open(record_path, encoding="utf-8"):
            line_ = line_.split("#")[0].strip()
            if line_.isdigit():
                recorded.add(int(line_))

    print(f"{len(patched)} tooltip(s) value-corrected, {len(added)} new DBC-only row(s), "
          f"{len(skipped_unresolved)} skipped (unresolved $ token), {len(no_desc)} with no usable desc.")
    if skipped_unresolved:
        print("  unresolved:", skipped_unresolved[:30], "..." if len(skipped_unresolved) > 30 else "")

    if args.dry_run:
        return

    with open(tooltip_path, "w", encoding="utf-8", newline="") as f:
        f.write("\n".join(lines) + "\n")

    with open(record_path, "w", encoding="utf-8") as f:
        f.write("# Non-talent spell ids with no Classic Wowhead data - DBC-only rows.\n")
        f.write("# Maintained by tools/gen_spell_value_overrides.py\n")
        for sid in sorted(recorded - talent_ids):
            f.write(f"{sid}\n")

    print(f"Wrote {tooltip_path} ({len(patched)} patched, {len(added)} appended)")


if __name__ == "__main__":
    main()
