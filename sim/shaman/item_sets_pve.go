package shaman

import (
	"slices"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

var ItemSetTheFiveThunders = core.NewItemSet(core.ItemSet{
	Name: "The Five Thunders",
	Bonuses: map[int32]core.ApplyEffect{
		// +10 Resistances/+200 Armor.
		2: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddResistances(10)
			c.AddStat(stats.Armor, 200)
		},
		// Chance on offensive action to increase your melee attack power, damage and healing by up to 100 for 10 sec.
		4: func(agent core.Agent) {
			c := agent.GetCharacter()

			procAura := c.NewTemporaryStatsAura("The Furious Storm", core.ActionID{SpellID: 27775}, stats.Stats{stats.AttackPower: 100, stats.SpellPower: 100}, time.Second*10)
			core.MakeProcTriggerAura(&c.Unit, core.ProcTrigger{
				Name:       "Item - The Furious Storm Proc (Offensive Action)",
				Callback:   core.CallbackOnSpellHitDealt,
				ProcMask:   core.ProcMaskMelee | core.ProcMaskSpellDamage | core.ProcMaskSpellHealing,
				Outcome:    core.OutcomeLanded,
				ProcChance: 0.04,
				Handler: func(sim *core.Simulation, spell *core.Spell, _ *core.SpellResult) {
					procAura.Activate(sim)
				},
			})
		},
		// Increases damage and healing done by magical spells and effects by up to 23.
		6: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.SpellPower, 23)
		},
		// +10 Resistances/+200 Armor.
		8: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddResistances(10)
			c.AddStat(stats.Armor, 200)
		},
	},
})

var ItemSetTheEarthfury = core.NewItemSet(core.ItemSet{
	Name: "The Earthfury",
	Bonuses: map[int32]core.ApplyEffect{
		// The radius of your totems that affect friendly targets is increased by 10 yd.
		2: func(agent core.Agent) {
			// Nothing to do: totem effect radius isn't modeled by this sim (no
			// positional/AoE-falloff simulation for friendly totem auras).
		},
		// Reduces the cost of your Healing Wave by 5%.
		4: func(agent core.Agent) {
			// Nothing to do: Healing Wave (restoration healing spec) isn't implemented
			// in this sim (see sim/shaman/_restoration, excluded from the build).
		},
		// Your Healing Wave will now jump to additional nearby target. Jump reduces the effectiveness of the heal by 70%.
		6: func(agent core.Agent) {
			// Nothing to do: Healing Wave isn't implemented in this sim.
		},
		// After casting your Healing Wave or Lesser Healing Wave spell, gives you a 40% chance to gain Mana equal to 35% of the base cost of the spell.
		8: func(agent core.Agent) {
			// Nothing to do: Healing Wave/Lesser Healing Wave aren't implemented in this sim.
		},
	},
})

var ItemSetTheTenStorms = core.NewItemSet(core.ItemSet{
	Name: "The Ten Storms",
	Bonuses: map[int32]core.ApplyEffect{
		// Increases the effect of your Mana Tide, Mana Spring and Healing Stream totems by 30%.
		2: func(agent core.Agent) {
			// Nothing to do: Mana Spring/Mana Tide's mana restoration isn't simulated per-rank
			// (buffs.go uses hard-coded values instead, see the TODO in water_totems.go's
			// newManaSpringTotemSpellConfig), and Healing Stream's heal amount has no bonus
			// multiplier hook to attach to.
		},
		// Increases the effect of Chain Lightning and Chain Heal spells to targets beyond the first by 30%.
		4: func(agent core.Agent) {
			shaman := agent.(ShamanAgent).GetShaman()
			// Removes the 30% per-bounce falloff on Chain Lightning (base coefficient is .70).
			shaman.ChainLightningBounceCoefficient += 0.30
			// Chain Heal isn't implemented in this sim (restoration spec is excluded from
			// the build), so there is nothing to do for that half of the bonus.
		},
		// Your Lightning Bolt, Chain Lightning and Chain Heal critical hits generate 30% less threat.
		6: func(agent core.Agent) {
			// Nothing to do: threat generation uses a static per-spell ThreatMultiplier with
			// no per-hit/crit-only override hook, so a crit-only threat reduction can't be
			// modeled without a broader threat-system change.
		},
		// Improves your chance to get a critical strike with Nature spells by 3%.
		8: func(agent core.Agent) {
			shaman := agent.(ShamanAgent).GetShaman()
			shaman.PseudoStats.SchoolBonusCritChance[stats.SchoolIndexNature] += 3 * core.SpellCritRatingPerCritChance
		},
	},
})

var ItemSetStormcallersGarb = core.NewItemSet(core.ItemSet{
	Name: "Stormcaller's Garb",
	Bonuses: map[int32]core.ApplyEffect{
		// Your Lightning Bolt, Chain Lightning, and Shock spells have a 20% chance to grant up to 50 Nature damage to spells for 8 sec. (Proc chance: 20%)
		3: func(agent core.Agent) {
			shaman := agent.(ShamanAgent).GetShaman()

			buffAura := shaman.RegisterAura(core.Aura{
				ActionID: core.ActionID{SpellID: 26121},
				Label:    "Stormcaller's Wrath",
				Duration: time.Second * 8,
				OnGain: func(aura *core.Aura, sim *core.Simulation) {
					shaman.AddStatDynamic(sim, stats.NaturePower, 50)
				},
				OnExpire: func(aura *core.Aura, sim *core.Simulation) {
					shaman.AddStatDynamic(sim, stats.NaturePower, -50)
				},
			})

			affectedSpellCodes := []int32{SpellCode_ShamanLightningBolt, SpellCode_ShamanChainLightning, SpellCode_ShamanEarthShock, SpellCode_ShamanFlameShock, SpellCode_ShamanFrostShock}

			core.MakeProcTriggerAura(&shaman.Unit, core.ProcTrigger{
				Name:     "Stormcaller Spelldamage Bonus",
				Callback: core.CallbackOnSpellHitDealt,
				Outcome:  core.OutcomeLanded,
				Handler: func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
					if slices.Contains(affectedSpellCodes, spell.SpellCode) && sim.Proc(0.20, "Stormcaller's Wrath") {
						buffAura.Activate(sim)
					}
				},
			})
		},
		// -0.4 seconds on the casting time of your Chain Heal spell.
		5: func(agent core.Agent) {
		},
	},
})

var ItemSetGiftOfTheGatheringStorm = core.NewItemSet(core.ItemSet{
	Name: "Gift of the Gathering Storm",
	Bonuses: map[int32]core.ApplyEffect{
		// Increases the chain target damage multiplier of your Chain Lightning spell by 5%.
		3: func(agent core.Agent) {
			shaman := agent.(ShamanAgent).GetShaman()
			shaman.RegisterAura(core.Aura{
				Label: "Gift of the Gathering Storm Chain Lightning Bonus",
				OnInit: func(aura *core.Aura, sim *core.Simulation) {
					shaman.ChainLightningBounceCoefficient += 0.05
				},
			})
		},
	},
})

var ItemSetCataclysmArmor = core.NewItemSet(core.ItemSet{
	Name: "Cataclysm Armor",
	Bonuses: map[int32]core.ApplyEffect{
		// Increases the effect of your Strength of Earth, Stoneskin and Windwall totems by 20%.
		2: func(agent core.Agent) {
			shaman := agent.(ShamanAgent).GetShaman()
			shaman.TotemEffectivenessBonusMultiplier += 0.20
			// Windwall Totem currently has no simulated effect (it only affects ranged
			// avoidance, which is not modeled), so there is nothing to boost for it.
		},
		// Increases the effectiveness of your elemental weapon enchants by 10%.
		4: func(agent core.Agent) {
			shaman := agent.(ShamanAgent).GetShaman()
			shaman.ElementalWeaponEnchantEffectivenessBonus += 0.10
		},
		// Your Shock spells criticals will refund 150% of their base mana cost.
		6: func(agent core.Agent) {
			shaman := agent.(ShamanAgent).GetShaman()
			manaMetrics := shaman.NewManaMetrics(core.ActionID{SpellID: 25046})

			shaman.RegisterAura(core.Aura{
				Label:    "Cataclysm Armor 6pc",
				Duration: core.NeverExpires,
				OnReset: func(aura *core.Aura, sim *core.Simulation) {
					aura.Activate(sim)
				},
				OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
					if !result.DidCrit() || spell.CurCast.Cost == 0 {
						return
					}
					if spell.SpellCode == SpellCode_ShamanEarthShock || spell.SpellCode == SpellCode_ShamanFlameShock || spell.SpellCode == SpellCode_ShamanFrostShock {
						shaman.AddMana(sim, spell.Cost.BaseCost*1.5, manaMetrics)
					}
				},
			})
		},
		// Increases your spell damage and healing by 10% of your Attack Power.
		8: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStatDependency(stats.AttackPower, stats.SpellPower, 0.10)
		},
	},
})

var ItemSetTheStonefury = core.NewItemSet(core.ItemSet{
	Name: "The Stonefury",
	Bonuses: map[int32]core.ApplyEffect{
		// +20 Stamina.
		2: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.Stamina, 20)
		},
		// Reduces the chance that the opponent can resist your Shock spells by 5%.
		4: func(agent core.Agent) {
			shaman := agent.(ShamanAgent).GetShaman()
			bonusHit := 5 * float64(core.SpellHitRatingPerHitChance)
			shaman.OnSpellRegistered(func(spell *core.Spell) {
				if spell.SpellCode == SpellCode_ShamanEarthShock || spell.SpellCode == SpellCode_ShamanFlameShock || spell.SpellCode == SpellCode_ShamanFrostShock {
					spell.BonusHitRating += bonusHit
				}
			})
		},
		// Reduces the cooldown of Aftershock, Stormstrike and Upheaval by 25%.
		6: func(agent core.Agent) {
			// Nothing to do: Stormstrike is disabled in this sim's custom talent tree
			// (see registerStormstrikeSpell), Aftershock is a passive talent with no
			// cooldown to reduce, and Upheaval is not a spell implemented in this sim.
		},
		// Reduces the cost of your spells by 15%.
		8: func(agent core.Agent) {
			shaman := agent.(ShamanAgent).GetShaman()
			shaman.OnSpellRegistered(func(spell *core.Spell) {
				if spell.Flags.Matches(SpellFlagShaman) && spell.Cost != nil {
					spell.Cost.Multiplier -= 15
				}
			})
		},
	},
})

var ItemSetTheEarthshatterer = core.NewItemSet(core.ItemSet{
	Name: "The Earthshatterer",
	Bonuses: map[int32]core.ApplyEffect{
		// Reduces the mana cost of your totem spells by 10%.
		2: func(agent core.Agent) {
			shaman := agent.(ShamanAgent).GetShaman()
			shaman.OnSpellRegistered(func(spell *core.Spell) {
				if spell.Flags.Matches(SpellFlagTotem) {
					spell.Cost.Multiplier -= 10
				}
			})
		},
		// Increases the mana gained from your Mana Tide and Mana Spring totems by 25%.
		4: func(agent core.Agent) {
			// Nothing to do: Mana Tide/Mana Spring's mana restoration isn't simulated
			// per-rank (buffs.go uses hard-coded values instead, see the TODO in
			// water_totems.go's newManaSpringTotemSpellConfig).
		},
		// Your Healing Wave and Lesser Healing Wave spells have a chance to imbue your target with Totemic Power.
		6: func(agent core.Agent) {
			// Nothing to do: Healing Wave/Lesser Healing Wave (restoration healing spec)
			// aren't implemented in this sim (see sim/shaman/_restoration, excluded from the build).
		},
		// Your Lightning Shield spell also grants you 15 mana per 5 sec. while active.
		8: func(agent core.Agent) {
			shaman := agent.(ShamanAgent).GetShaman()
			shaman.RegisterAura(core.Aura{
				Label: "Lightning Shield",
				OnInit: func(_ *core.Aura, _ *core.Simulation) {
					for _, lsAura := range shaman.LightningShieldAuras {
						if lsAura == nil {
							return
						}

						oldOnGain := lsAura.OnGain
						lsAura.OnGain = func(aura *core.Aura, sim *core.Simulation) {
							oldOnGain(aura, sim)
							shaman.AddStatDynamic(sim, stats.MP5, 15)
						}

						oldOnExpire := lsAura.OnExpire
						lsAura.OnExpire = func(aura *core.Aura, sim *core.Simulation) {
							oldOnExpire(aura, sim)
							shaman.AddStatDynamic(sim, stats.MP5, -15)
						}
					}
				},
			})
		},
	},
})
