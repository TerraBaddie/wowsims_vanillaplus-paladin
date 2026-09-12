#!/usr/bin/python
"""Generate wowsims talent config (proto message + tree JSON) from custom DBC CSV exports.

Replaces the Wowhead scrapers (scrape_talents_proto.py / scrape_talents_config.py).

Inputs (in --csv-dir, default "CSV's"):
  Talent.csv     cols: 0 id, 1 tabId, 2 row, 3 col, 4-8 spellRank[1..5],
                       13 prereqTalentId, 16 prereqRank, 19 flags
  TalentTab.csv  cols: 0 id, 1 name, 12 classMask, 13 orderIndex
  Spell.csv      cols: 0 id, 120 name

Usage:
  python tools/dbc_to_talents.py priest
  python tools/dbc_to_talents.py --all
"""
import argparse
import csv
import json
import os
import re
import sys

CLASS_MASK = {
    "warrior": "1", "paladin": "2", "hunter": "4", "rogue": "8",
    "priest": "16", "shaman": "64", "mage": "128", "warlock": "256", "druid": "1024",
}

# Spell ids missing a usable SpellName in the CSV dump -> explicit override.
NAME_OVERRIDES = {
    15237: "Holy Nova",
}

TALENT_COL_ID, TALENT_COL_TAB, TALENT_COL_ROW, TALENT_COL_COL = 0, 1, 2, 3
TALENT_COL_RANKS = slice(4, 9)
TALENT_COL_PREREQ = 13
TAB_COL_ID, TAB_COL_NAME, TAB_COL_CLASSMASK, TAB_COL_ORDER = 0, 1, 12, 13
SPELL_COL_ID, SPELL_COL_NAME = 0, 120

BACKGROUND_URL = "https://wow.zamimg.com/images/wow/talents/backgrounds/classic/{tab}.jpg"


def load_rows(path):
    with open(path, encoding="utf-8") as f:
        for row in csv.reader(f):
            if not row or row[0] == "long":
                continue
            yield row


def snake(name):
    # Treat hyphens/slashes as word boundaries ("Two-Handed" -> two_handed).
    name = re.sub(r"[-/]", " ", name)
    parts = re.sub(r"[^0-9a-zA-Z ]", "", name).split()
    return "_".join(p.lower() for p in parts)


def camel(sn):
    head, *tail = sn.split("_")
    return head + "".join(w.title() for w in tail)


def build(class_name, csv_dir):
    mask = CLASS_MASK[class_name]

    spell_names = {}
    for r in load_rows(os.path.join(csv_dir, "Spell.csv")):
        try:
            name = r[SPELL_COL_NAME]
            nxt = r[SPELL_COL_NAME + 1] if len(r) > SPELL_COL_NAME + 1 else ""
            # A few rows in this dump are shifted right: the name slot holds junk
            # (empty / "0" / a stray number) and the real name is one column over.
            if (not name or name == "0" or (name.isdigit() and any(c.isalpha() for c in nxt))):
                name = nxt
            spell_names[int(r[SPELL_COL_ID])] = name
        except (ValueError, IndexError):
            pass
    for sid, nm in NAME_OVERRIDES.items():
        spell_names[sid] = nm

    tabs = []  # (orderIndex, tabId, treeName)
    for r in load_rows(os.path.join(csv_dir, "TalentTab.csv")):
        if r[TAB_COL_CLASSMASK] == mask:
            tabs.append((int(r[TAB_COL_ORDER]), int(r[TAB_COL_ID]), r[TAB_COL_NAME]))
    tabs.sort()
    tab_order = {tab_id: i for i, (_, tab_id, _) in enumerate(tabs)}

    talents = []  # dicts
    for r in load_rows(os.path.join(csv_dir, "Talent.csv")):
        tab_id = int(r[TALENT_COL_TAB])
        if tab_id not in tab_order:
            continue
        ranks = [int(x) for x in r[TALENT_COL_RANKS] if x and x != "0"]
        prereq = int(r[TALENT_COL_PREREQ]) if r[TALENT_COL_PREREQ] not in ("", "0") else 0
        talents.append({
            "id": int(r[TALENT_COL_ID]),
            "tab": tab_id,
            "row": int(r[TALENT_COL_ROW]),
            "col": int(r[TALENT_COL_COL]),
            "ranks": ranks,
            "prereq": prereq,
        })
    by_id = {t["id"]: t for t in talents}

    warnings = []
    for t in talents:
        first = t["ranks"][0] if t["ranks"] else 0
        nm = spell_names.get(first, "")
        if not nm or nm == "0":
            warnings.append("  tab {} r{} c{}: spell {} has no name".format(t["tab"], t["row"], t["col"], first))
            nm = "talent_{}".format(t["id"])
        t["snake"] = snake(nm)

    # ---- tree JSON ----
    trees_json = []
    for _, tab_id, tree_name in tabs:
        tree_talents = sorted(
            (t for t in talents if t["tab"] == tab_id),
            key=lambda t: t["row"] * 4 + t["col"],
        )
        entries = []
        for t in tree_talents:
            entry = {
                "fieldName": camel(t["snake"]),
                "location": {"rowIdx": t["row"], "colIdx": t["col"]},
                "spellIds": t["ranks"],
                "maxPoints": len(t["ranks"]),
            }
            if t["prereq"] and t["prereq"] in by_id:
                p = by_id[t["prereq"]]
                entry["prereqLocation"] = {"rowIdx": p["row"], "colIdx": p["col"]}
            entries.append(entry)
        trees_json.append({
            "name": tree_name,
            "backgroundUrl": BACKGROUND_URL.format(tab=tab_id),
            "talents": entries,
        })

    # ---- proto message ----
    pretty = class_name.title()
    lines = ["message {}Talents {{".format(pretty)]
    field_idx = 1
    for tree_i, (_, tab_id, tree_name) in enumerate(tabs):
        lines.append("\t// {}".format(tree_name))
        tree_talents = sorted(
            (t for t in talents if t["tab"] == tab_id),
            key=lambda t: t["row"] * 4 + t["col"],
        )
        for t in tree_talents:
            ftype = "bool" if len(t["ranks"]) == 1 else "int32"
            lines.append("\t{} {} = {};".format(ftype, t["snake"], field_idx))
            field_idx += 1
        if tree_i != len(tabs) - 1:
            lines.append("")
    lines.append("}")

    return "\n".join(lines) + "\n", trees_json, warnings


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("classes", nargs="*")
    ap.add_argument("--all", action="store_true")
    ap.add_argument("--csv-dir", default="CSV's")
    ap.add_argument("--repo", default=".")
    ap.add_argument("--write", action="store_true", help="write proto + json into the repo")
    args = ap.parse_args()

    targets = list(CLASS_MASK) if args.all else args.classes
    if not targets:
        ap.error("pass class names or --all")

    for cn in targets:
        proto_msg, trees_json, warnings = build(cn, os.path.join(args.repo, args.csv_dir))
        json_path = os.path.join(args.repo, "ui/core/talents/trees", cn + ".json")
        if args.write:
            with open(json_path, "w", encoding="utf-8") as f:
                json.dump(trees_json, f, indent=2)
                f.write("\n")
            print("wrote", json_path)
            print("--- paste into proto/{}.proto ---".format(cn))
        print(proto_msg)
        if warnings:
            sys.stderr.write("WARNINGS for {}:\n".format(cn) + "\n".join(warnings) + "\n")


if __name__ == "__main__":
    main()
