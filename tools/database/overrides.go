package database

import (
	"regexp"

	"github.com/wowsims/classic/sim/core/proto"
)

var OtherItemIdsToFetch = []string{}

var ItemOverrides = []*proto.UIItem{
	// Valentine's day event rewards
	// {Id: 51804, Phase: 2},

	// Some items might slip past the phase filters defined in main.go
	// Dragonspur Wraps
	{Id: 20615, Phase: 4},

	// Argent Dawn Armaments of Battle
	{Id: 22657, Phase: 6},
	{Id: 22667, Phase: 6},
	{Id: 22668, Phase: 6},
	{Id: 22659, Phase: 6},
	{Id: 22678, Phase: 6},
	{Id: 22656, Phase: 6},

	{Id: 22681, Phase: 6},
	{Id: 22680, Phase: 6},
	{Id: 22688, Phase: 6},
	{Id: 22679, Phase: 6},
	{Id: 22690, Phase: 6},
	{Id: 22689, Phase: 6},

	// Nef Head
	{Id: 19383, Phase: 3},
	{Id: 19366, Phase: 3},
	{Id: 19384, Phase: 3},

	// Dire Maul Crafts
	{Id: 18504, Phase: 2},
	{Id: 18506, Phase: 2},
	{Id: 18508, Phase: 2},
	{Id: 18509, Phase: 2},
	{Id: 18510, Phase: 2},
	{Id: 18511, Phase: 2},

	{Id: 18405, Phase: 2},
	{Id: 18407, Phase: 2},
	{Id: 18408, Phase: 2},
	{Id: 18409, Phase: 2},
	{Id: 18413, Phase: 2},

	// Sunken Temple Class Quests
	{Id: 22274, Phase: 3},
	{Id: 22272, Phase: 3},
	{Id: 22458, Phase: 3},

	{Id: 20083, Phase: 3},
	{Id: 19991, Phase: 3},
	{Id: 19992, Phase: 3},

	{Id: 20035, Phase: 3},
	{Id: 20037, Phase: 3},
	{Id: 20036, Phase: 3},

	{Id: 20504, Phase: 3},
	{Id: 20512, Phase: 3},
	{Id: 20505, Phase: 3},

	{Id: 19990, Phase: 3},
	{Id: 20082, Phase: 3},
	{Id: 20006, Phase: 3},

	{Id: 19984, Phase: 3},
	{Id: 20255, Phase: 3},
	{Id: 19982, Phase: 3},

	{Id: 20369, Phase: 3},
	{Id: 20556, Phase: 3},
	{Id: 20503, Phase: 3},

	{Id: 20536, Phase: 3},
	{Id: 20534, Phase: 3},
	{Id: 20530, Phase: 3},

	{Id: 20130, Phase: 3},
	{Id: 20521, Phase: 3},
	{Id: 20517, Phase: 3},

	// AQ Patch Items with incorrect Phases
	// Shaman Totems
	{Id: 23199, Phase: 5},
	{Id: 23200, Phase: 5},
	{Id: 22345, Phase: 5},
	{Id: 22395, Phase: 5},

	// Crafted
	{Id: 22191, Phase: 5},
	{Id: 22195, Phase: 5},
}

// Keep these sorted by item ID.
var ItemAllowList = map[int32]struct{}{}

// Items to remove from the UI
var ItemDenyList = map[int32]struct{}{
	9653:  {}, // Speedy Racer Goggles
	12104: {}, // Brindlethorn Tunic
	12805: {}, // Orb of Fire
	17782: {}, // talisman of the binding shard
	17783: {}, // talisman of the binding fragment
	17802: {}, // Deprecated version of Thunderfury
	20522: {}, // Feral Staff
	22736: {}, // Andonisus, Reaper of Souls

	// Unimplemented PvP Belts/Bracers (Marshal's/General's)
	16482: {},
	16447: {},
	16458: {},
	16470: {},
	17585: {},
	17609: {},
	16439: {},
	16464: {},

	16481: {},
	17606: {},
	16438: {},
	16445: {},
	16460: {},
	16469: {},
	17582: {},
	16461: {},

	16572: {},
	16557: {},
	16547: {},
	16556: {},
	16575: {},
	17589: {},
	17621: {},
	16537: {},

	16553: {},
	16546: {},
	16576: {},
	16538: {},
	16559: {},
	16570: {},
	17587: {},
	17619: {},

	// Bladebane Armguards
	14550: {},
}

// Item icons to include in the DB, so they don't need to be separately loaded in the UI.
var ExtraItemIcons = []int32{
	// Demonic Rune
	12662,

	// Explosives
	13180,
	11566,
	8956,
	10646,
	18641,
	15993,
	16040,

	// Food IDs
	13928,
	20452,
	13931,
	18254,
	21023,
	13813,
	13810,

	// Flask IDs
	13510,
	13511,
	13512,
	13513,

	// Zanza
	20079,

	// Blasted Lands
	8412,
	8423,
	8424,
	8411,

	// Agility Elixer IDs
	13452,
	9187,

	// Single Elixirs
	20007, // Mana Regen Elixir
	20004, // Major Troll's Blood Potion
	9088,  // Gift of Arthas

	// Armor Elixirs
	3389,  // Defense
	8951,  // Greater
	13445, // Superior Defense

	// Health Elixirs
	2458, // Minor Fortitude
	3825, // Fortitude

	// Strength
	12451,
	9206,

	// AP
	12460,
	12820,

	// Random
	5206, // Bogling Root

	// SP
	13454,
	9264,
	21546,
	17708,

	// Crystal
	11564, // Armor

	// Alcohol Buff
	18284,
	18269,
	20709,
	21114,
	21151,

	// Potions / In Battle Consumes
	13444,

	// Thistle Tea
	7676,

	// Weapon Oils
	20748,
	20749,
	12404,
	18262,
}

// Local corrections for enchants whose SpellId is either fully custom
// (private-server-only, no Wowhead entry) or reuses a real Blizzard spell id
// for an unrelated effect (Wowhead's live tooltip for that id is then wrong).
// Per docs/private-server-item-rules.md "Tooltip/icon text": our data wins
// over a live Wowhead lookup whenever the two disagree. Icons here are
// reused from the enchant's own real item icon (already correctly resolved
// via AddItemIcon since ItemId is set) for consistency.
// (per user 2026-09-12)
var SpellIconoverrides = []*proto.IconData{
	{Id: 15389, Name: "Lesser Arcanum of Constitution", Icon: "inv_misc_gem_03", Tooltip: `<table><tr><td><a class="whtt-name" href="/classic/spell=15389/lesser-arcanum-of-constitution"><b class="whtt-name">Lesser Arcanum of Constitution</b></a></td></tr></table><table><tr><td><span class="wowhead-tooltip-requirements">Requires Helms, Pants</span><div class="q">Permanently adds 100 hit points to a leg or head slot item. Does not stack with other enchantments for the selected equipment slot.</div></td></tr></table>`},
	{Id: 15394, Name: "Lesser Arcanum of Resilience", Icon: "inv_misc_gem_03", Tooltip: `<table><tr><td><a class="whtt-name" href="/classic/spell=15394/lesser-arcanum-of-resilience"><b class="whtt-name">Lesser Arcanum of Resilience</b></a></td></tr></table><table><tr><td><span class="wowhead-tooltip-requirements">Requires Helms, Pants</span><div class="q">Permanently adds 20 fire resistance to a leg or head slot item. Does not stack with other enchantments for the selected equipment slot.</div></td></tr></table>`},
	{Id: 15340, Name: "Lesser Arcanum of Rumination", Icon: "inv_misc_gem_03", Tooltip: `<table><tr><td><a class="whtt-name" href="/classic/spell=15340/lesser-arcanum-of-rumination"><b class="whtt-name">Lesser Arcanum of Rumination</b></a></td></tr></table><table><tr><td><span class="wowhead-tooltip-requirements">Requires Helms, Pants</span><div class="q">Permanently adds 200 mana to a leg or head slot item. Does not stack with other enchantments for the selected equipment slot.</div></td></tr></table>`},
	{Id: 15391, Name: "Lesser Arcanum of Tenacity", Icon: "inv_misc_gem_03", Tooltip: `<table><tr><td><a class="whtt-name" href="/classic/spell=15391/lesser-arcanum-of-tenacity"><b class="whtt-name">Lesser Arcanum of Tenacity</b></a></td></tr></table><table><tr><td><span class="wowhead-tooltip-requirements">Requires Helms, Pants</span><div class="q">Permanently adds 1% Crit suppression to a leg or head slot item. Does not stack with other enchantments for the selected equipment slot.</div></td></tr></table>`},
	{Id: 15397, Name: "Lesser Arcanum of Voracity", Icon: "inv_misc_gem_03", Tooltip: `<table><tr><td><a class="whtt-name" href="/classic/spell=15397/lesser-arcanum-of-voracity"><b class="whtt-name">Lesser Arcanum of Voracity</b></a></td></tr></table><table><tr><td><span class="wowhead-tooltip-requirements">Requires Helms, Pants</span><div class="q">Permanently adds 8 strength to a leg or head slot item. Does not stack with other enchantments for the selected equipment slot.</div></td></tr></table>`},
	{Id: 15400, Name: "Lesser Arcanum of Voracity", Icon: "inv_misc_gem_03", Tooltip: `<table><tr><td><a class="whtt-name" href="/classic/spell=15400/lesser-arcanum-of-voracity"><b class="whtt-name">Lesser Arcanum of Voracity</b></a></td></tr></table><table><tr><td><span class="wowhead-tooltip-requirements">Requires Helms, Pants</span><div class="q">Permanently adds 8 stamina to a leg or head slot item. Does not stack with other enchantments for the selected equipment slot.</div></td></tr></table>`},
	{Id: 15402, Name: "Lesser Arcanum of Voracity", Icon: "inv_misc_gem_03", Tooltip: `<table><tr><td><a class="whtt-name" href="/classic/spell=15402/lesser-arcanum-of-voracity"><b class="whtt-name">Lesser Arcanum of Voracity</b></a></td></tr></table><table><tr><td><span class="wowhead-tooltip-requirements">Requires Helms, Pants</span><div class="q">Permanently adds 8 agility to a leg or head slot item. Does not stack with other enchantments for the selected equipment slot.</div></td></tr></table>`},
	{Id: 15404, Name: "Lesser Arcanum of Voracity", Icon: "inv_misc_gem_03", Tooltip: `<table><tr><td><a class="whtt-name" href="/classic/spell=15404/lesser-arcanum-of-voracity"><b class="whtt-name">Lesser Arcanum of Voracity</b></a></td></tr></table><table><tr><td><span class="wowhead-tooltip-requirements">Requires Helms, Pants</span><div class="q">Permanently adds 8 intellect to a leg or head slot item. Does not stack with other enchantments for the selected equipment slot.</div></td></tr></table>`},
	{Id: 15406, Name: "Lesser Arcanum of Voracity", Icon: "inv_misc_gem_03", Tooltip: `<table><tr><td><a class="whtt-name" href="/classic/spell=15406/lesser-arcanum-of-voracity"><b class="whtt-name">Lesser Arcanum of Voracity</b></a></td></tr></table><table><tr><td><span class="wowhead-tooltip-requirements">Requires Helms, Pants</span><div class="q">Permanently adds 8 spirit to a leg or head slot item. Does not stack with other enchantments for the selected equipment slot.</div></td></tr></table>`},
	{Id: 22844, Name: "Arcanum of Focus", Icon: "inv_misc_gem_02", Tooltip: `<table><tr><td><a class="whtt-name" href="/classic/spell=22844/arcanum-of-focus"><b class="whtt-name">Arcanum of Focus</b></a></td></tr></table><table><tr><td><span class="wowhead-tooltip-requirements">Requires Helms, Pants</span><div class="q">Permanently adds +10 to your Healing and Damage from spells to a leg or head slot item. Does not stack with other enchantments for the selected equipment slot.</div></td></tr></table>`},
	{Id: 22840, Name: "Arcanum of Rapidity", Icon: "inv_misc_gem_02", Tooltip: `<table><tr><td><a class="whtt-name" href="/classic/spell=22840/arcanum-of-rapidity"><b class="whtt-name">Arcanum of Rapidity</b></a></td></tr></table><table><tr><td><span class="wowhead-tooltip-requirements">Requires Helms, Pants</span><div class="q">Permanently adds 2% haste to a leg or head slot item. Does not stack with other enchantments for the selected equipment slot.</div></td></tr></table>`},
}

// Same corrections as SpellIconoverrides above, but keyed by the enchant's real
// ItemId instead of SpellId. The gear-picker UI prefers ActionId.fromItemId when
// an enchant has one set (ui/core/components/gear_picker/selector_modal.tsx), so
// for these reagent-item enchants THIS is the table that actually reaches the
// on-hover tooltip client-side -- SpellIconoverrides above is only consulted for
// enchants with no ItemId. (per user 2026-09-12; see docs/private-server-item-rules.md
// "Known limitation: reused real spell/item ids still show Wowhead's live tooltip")
var ItemIconoverrides = []*proto.IconData{
	{Id: 11642, Name: "Lesser Arcanum of Constitution", Icon: "inv_misc_gem_03", Tooltip: `<table><tr><td><a class="whtt-name" href="/classic/item=11642/lesser-arcanum-of-constitution"><b class="whtt-name">Lesser Arcanum of Constitution</b></a></td></tr></table><table><tr><td><span class="wowhead-tooltip-requirements">Quest Item</span><div class="q">Permanently adds 100 hit points to a leg or head slot item. Does not stack with other enchantments for the selected equipment slot.</div></td></tr></table>`},
	{Id: 11644, Name: "Lesser Arcanum of Resilience", Icon: "inv_misc_gem_03", Tooltip: `<table><tr><td><a class="whtt-name" href="/classic/item=11644/lesser-arcanum-of-resilience"><b class="whtt-name">Lesser Arcanum of Resilience</b></a></td></tr></table><table><tr><td><span class="wowhead-tooltip-requirements">Quest Item</span><div class="q">Permanently adds 20 fire resistance to a leg or head slot item. Does not stack with other enchantments for the selected equipment slot.</div></td></tr></table>`},
	{Id: 11622, Name: "Lesser Arcanum of Rumination", Icon: "inv_misc_gem_03", Tooltip: `<table><tr><td><a class="whtt-name" href="/classic/item=11622/lesser-arcanum-of-rumination"><b class="whtt-name">Lesser Arcanum of Rumination</b></a></td></tr></table><table><tr><td><span class="wowhead-tooltip-requirements">Quest Item</span><div class="q">Permanently adds 200 mana to a leg or head slot item. Does not stack with other enchantments for the selected equipment slot.</div></td></tr></table>`},
	{Id: 11643, Name: "Lesser Arcanum of Tenacity", Icon: "inv_misc_gem_03", Tooltip: `<table><tr><td><a class="whtt-name" href="/classic/item=11643/lesser-arcanum-of-tenacity"><b class="whtt-name">Lesser Arcanum of Tenacity</b></a></td></tr></table><table><tr><td><span class="wowhead-tooltip-requirements">Quest Item</span><div class="q">Permanently adds 1% Crit suppression to a leg or head slot item. Does not stack with other enchantments for the selected equipment slot.</div></td></tr></table>`},
	{Id: 11645, Name: "Lesser Arcanum of Voracity", Icon: "inv_misc_gem_03", Tooltip: `<table><tr><td><a class="whtt-name" href="/classic/item=11645/lesser-arcanum-of-voracity"><b class="whtt-name">Lesser Arcanum of Voracity</b></a></td></tr></table><table><tr><td><span class="wowhead-tooltip-requirements">Quest Item</span><div class="q">Permanently adds 8 strength to a leg or head slot item. Does not stack with other enchantments for the selected equipment slot.</div></td></tr></table>`},
	{Id: 11646, Name: "Lesser Arcanum of Voracity", Icon: "inv_misc_gem_03", Tooltip: `<table><tr><td><a class="whtt-name" href="/classic/item=11646/lesser-arcanum-of-voracity"><b class="whtt-name">Lesser Arcanum of Voracity</b></a></td></tr></table><table><tr><td><span class="wowhead-tooltip-requirements">Quest Item</span><div class="q">Permanently adds 8 stamina to a leg or head slot item. Does not stack with other enchantments for the selected equipment slot.</div></td></tr></table>`},
	{Id: 11647, Name: "Lesser Arcanum of Voracity", Icon: "inv_misc_gem_03", Tooltip: `<table><tr><td><a class="whtt-name" href="/classic/item=11647/lesser-arcanum-of-voracity"><b class="whtt-name">Lesser Arcanum of Voracity</b></a></td></tr></table><table><tr><td><span class="wowhead-tooltip-requirements">Quest Item</span><div class="q">Permanently adds 8 agility to a leg or head slot item. Does not stack with other enchantments for the selected equipment slot.</div></td></tr></table>`},
	{Id: 11648, Name: "Lesser Arcanum of Voracity", Icon: "inv_misc_gem_03", Tooltip: `<table><tr><td><a class="whtt-name" href="/classic/item=11648/lesser-arcanum-of-voracity"><b class="whtt-name">Lesser Arcanum of Voracity</b></a></td></tr></table><table><tr><td><span class="wowhead-tooltip-requirements">Quest Item</span><div class="q">Permanently adds 8 intellect to a leg or head slot item. Does not stack with other enchantments for the selected equipment slot.</div></td></tr></table>`},
	{Id: 11649, Name: "Lesser Arcanum of Voracity", Icon: "inv_misc_gem_03", Tooltip: `<table><tr><td><a class="whtt-name" href="/classic/item=11649/lesser-arcanum-of-voracity"><b class="whtt-name">Lesser Arcanum of Voracity</b></a></td></tr></table><table><tr><td><span class="wowhead-tooltip-requirements">Quest Item</span><div class="q">Permanently adds 8 spirit to a leg or head slot item. Does not stack with other enchantments for the selected equipment slot.</div></td></tr></table>`},
	{Id: 18330, Name: "Arcanum of Focus", Icon: "inv_misc_gem_02", Tooltip: `<table><tr><td><a class="whtt-name" href="/classic/item=18330/arcanum-of-focus"><b class="whtt-name">Arcanum of Focus</b></a></td></tr></table><table><tr><td><span class="wowhead-tooltip-requirements">Binds when picked up</span><div class="q">Permanently adds +10 to your Healing and Damage from spells to a leg or head slot item. Does not stack with other enchantments for the selected equipment slot.</div></td></tr></table>`},
	{Id: 18329, Name: "Arcanum of Rapidity", Icon: "inv_misc_gem_02", Tooltip: `<table><tr><td><a class="whtt-name" href="/classic/item=18329/arcanum-of-rapidity"><b class="whtt-name">Arcanum of Rapidity</b></a></td></tr></table><table><tr><td><span class="wowhead-tooltip-requirements">Binds when picked up</span><div class="q">Permanently adds 2% haste to a leg or head slot item. Does not stack with other enchantments for the selected equipment slot.</div></td></tr></table>`},
}

// Raid buffs / debuffs
var SharedSpellsIcons = []int32{
	// World Buffs
	22888, // Ony / Nef
	24425, // Spirit
	16609, // Warchief
	23768, // DMF Damage
	23736, // DMF Agi
	23766, // DMF Int
	23738, // DMF Spirit
	23737, // DMF Stam

	22818, // DM Stam
	22820, // DM Spell Crit
	22817, // DM AP

	15366, // Songflower

	29534, // Silithus

	18264, // Headmasters

	// Registered CD's
	10060, // Power Infusion
	29166, // Innervate

	// Mark
	1126,
	5232,
	6756,
	5234,
	8907,
	9884,
	9885,
	17055,

	20217, // Kings (Talent)
	25898, // Greater Kings
	25899, // Sanctuary

	10293, // Devo Aura
	20142, // Imp. Devo

	// Stoneskin Totem
	10408,
	16293,

	// Fort
	1243,
	1244,
	1245,
	2791,
	10937,
	10938,
	14767,

	// Spirit
	14752,
	14818,
	14819,
	27841,

	// Might
	19740,
	19834,
	19835,
	19836,
	19837,
	19838,
	25291,
	20048,

	// Commanding Shout
	6673,
	5242,
	6192,
	11549,
	11550,
	11551,
	25289,
	12861,

	// AP
	30811, // Unleashed Rage
	19506, // Trueshot

	// Battle Shout
	6673,
	5242,
	6192,
	11549,
	11550,
	11551,
	25289,
	12861, // Imp

	// Wisdom
	19742,
	19850,
	19852,
	19853,
	19854,
	25290,
	20245,

	// Mana Spring
	5675,
	10495,
	10496,
	10497,

	17007, // Leader of the Pack
	24858, // Moonkin

	// Windfury
	8512,
	10613,
	10614,
	29193, // Imp WF

	// Raid Debuffs
	8647,
	7386,
	7405,
	8380,
	11596,
	11597,

	770,
	778,
	9749,
	9907,
	11708,
	18181,

	26016,
	12879,
	9452,
	26021,
	16862,
	9747,
	9898,

	3043,
	14275,
	14276,
	14277,

	17800,
	17803,
	12873,
	28593,

	11374,
	15235,

	24977,
}

// If any of these match the item name, don't include it.
var DenyListNameRegexes = []*regexp.Regexp{
	regexp.MustCompile(`30 Epic`),
	regexp.MustCompile(`63 Blue`),
	regexp.MustCompile(`63 Green`),
	regexp.MustCompile(`66 Epic`),
	regexp.MustCompile(`90 Epic`),
	regexp.MustCompile(`90 Green`),
	regexp.MustCompile(`Boots 1`),
	regexp.MustCompile(`Boots 2`),
	regexp.MustCompile(`Boots 3`),
	regexp.MustCompile(`Bracer 1`),
	regexp.MustCompile(`Bracer 2`),
	regexp.MustCompile(`Bracer 3`),
	regexp.MustCompile(`DB\d`),
	regexp.MustCompile(`DEPRECATED`),
	regexp.MustCompile(`Deprecated: Keanna`),
	regexp.MustCompile(`Indalamar`),
	regexp.MustCompile(`Monster -`),
	regexp.MustCompile(`NEW`),
	regexp.MustCompile(`PH`),
	regexp.MustCompile(`QR XXXX`),
	regexp.MustCompile(`TEST`),
	regexp.MustCompile(`Test`),
	regexp.MustCompile(`zOLD`),
}
