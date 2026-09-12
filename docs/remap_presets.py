"""Rewrite built-in gear presets after an item-ID renumber (docs/private-server-item-rules.md Rule 4).

For every ui/*/gear_sets/*.gear.json:
  * item ids in assets/db_inputs/renumber.json  -> mapped to the server id
  * item ids no longer in the database           -> slot blanked ({})

Run after gen_db. Presets that end up mostly empty are dead content -- delete the
file and its presets.ts references by hand.
"""
import glob
import json
import os

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


def main():
    renum = {int(k): v for k, v in json.load(
        open(os.path.join(REPO, "assets/db_inputs/renumber.json"))).items()}
    db = {i["id"] for i in json.load(
        open(os.path.join(REPO, "assets/database/db.json")))["items"]}

    files = mapped = blanked = 0
    for gj in glob.glob(os.path.join(REPO, "ui/*/gear_sets/*.gear.json")):
        d = json.load(open(gj))
        dirty = False
        for it in d.get("items", []):
            iid = it.get("id")
            if iid is None:
                continue
            if iid in renum:
                it["id"] = iid = renum[iid]
                mapped += 1
                dirty = True
            if iid not in db:
                it.clear()
                blanked += 1
                dirty = True
        if dirty:
            json.dump(d, open(gj, "w"), separators=(",", ":"))
            files += 1
    print(f"{files} gear files updated: {mapped} ids remapped, {blanked} dead slots blanked")


if __name__ == "__main__":
    main()
