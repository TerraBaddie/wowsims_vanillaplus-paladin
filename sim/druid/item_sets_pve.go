package druid

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

var ItemSetFeralheartRaiment = core.NewItemSet(core.ItemSet{
	Name: "Feralheart Raiment",
	Bonuses: map[int32]core.ApplyEffect{
		// (2) Set : +10 Resistances/+200 Armor.
		2: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddResistances(10)
			c.AddStat(stats.Armor, 200)
		},
		// (4) Set : When struck in combat has a chance of returning 300 mana, 10 rage, or 40 energy to the wearer. (Proc chance: 2%)
		4: func(agent core.Agent) {
			c := agent.GetCharacter()
			actionID := core.ActionID{SpellID: 27781}
			manaMetrics := c.NewManaMetrics(actionID)
			energyMetrics := c.NewEnergyMetrics(actionID)
			rageMetrics := c.NewRageMetrics(actionID)

			core.MakeProcTriggerAura(&c.Unit, core.ProcTrigger{
				Name:       "Nature's Bounty (Mana)",
				Callback:   core.CallbackOnCastComplete,
				ProcMask:   core.ProcMaskSpellDamage | core.ProcMaskSpellHealing,
				ProcChance: 0.02,
				Handler: func(sim *core.Simulation, spell *core.Spell, _ *core.SpellResult) {
					c.AddMana(sim, 300, manaMetrics)
				},
			})
			core.MakeProcTriggerAura(&c.Unit, core.ProcTrigger{
				Name:       "Nature's Bounty (Energy)",
				Callback:   core.CallbackOnSpellHitDealt,
				Outcome:    core.OutcomeLanded,
				ProcMask:   core.ProcMaskMeleeWhiteHit,
				ProcChance: 0.02,
				Handler: func(sim *core.Simulation, spell *core.Spell, _ *core.SpellResult) {
					if c.HasEnergyBar() {
						c.AddEnergy(sim, 40, energyMetrics)
					}
				},
			})
			core.MakeProcTriggerAura(&c.Unit, core.ProcTrigger{
				Name:       "Nature's Bounty (Rage)",
				Callback:   core.CallbackOnSpellHitTaken,
				ProcMask:   core.ProcMaskMelee,
				ProcChance: 0.02,
				Handler: func(sim *core.Simulation, spell *core.Spell, _ *core.SpellResult) {
					if c.HasRageBar() {
						c.AddRage(sim, 10, rageMetrics)
					}
				},
			})
		},
		// (6) Set : Increases damage and healing done by magical spells and effects by up to 15.
		// (6) Set : +26 Attack Power.
		6: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.SpellPower, 15)
			c.AddStat(stats.AttackPower, 26)
		},
		// (8) Set : +10 Resistances/+200 Armor.
		8: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddResistances(10)
			c.AddStat(stats.Armor, 200)
		},
	},
})

var ItemSetCenarionArmor = core.NewItemSet(core.ItemSet{
	Name: "Cenarion Armor",
	Bonuses: map[int32]core.ApplyEffect{
		// (2) Set : Reduces the cast time of Rebirth, Revive, Hibernate and Soothe Animal
		// spells by 50% as well as the cooldown of Rebirth by 50%.
		2: func(agent core.Agent) {
			// Nothing to do: Rebirth/Revive/Hibernate/Soothe Animal are utility spells with
			// no cast-time/cooldown modeling relevant to this sim's DPS/HPS output.
		},
		// (4) Set : Increases the critical strike chance of your Claw, Rake, Ferocious Bite,
		// Maul and Swipe abilities by 4%.
		4: func(agent core.Agent) {
			druid := agent.(DruidAgent).GetDruid()
			bonusCrit := 4 * float64(core.CritRatingPerCritChance)
			druid.OnSpellRegistered(func(spell *core.Spell) {
				switch spell.SpellCode {
				case SpellCode_DruidClaw, SpellCode_DruidRake, SpellCode_DruidFerociousBite:
					spell.BonusCritRating += bonusCrit
				}
			})
			// Maul and Swipe (Bear Form) are not registered in this sim's Feral Tank
			// implementation, so there's nothing to hook the bonus onto for those two.
		},
		// (6) Set : Your finishing moves now refund 40 energy on a Miss, Dodge, Block, or Parry.
		6: func(agent core.Agent) {
			c := agent.GetCharacter()
			actionID := core.ActionID{SpellID: 22961}
			energyMetrics := c.NewEnergyMetrics(actionID)
			core.MakeProcTriggerAura(&c.Unit, core.ProcTrigger{
				Name:     "Cenarion Armor Finisher Bonus",
				Callback: core.CallbackOnSpellHitDealt,
				Outcome:  core.OutcomeMiss | core.OutcomeDodge | core.OutcomeBlock | core.OutcomeParry,
				Handler: func(sim *core.Simulation, spell *core.Spell, _ *core.SpellResult) {
					if spell.SpellCode == SpellCode_DruidFerociousBite || spell.SpellCode == SpellCode_DruidRip {
						c.AddEnergy(sim, 40, energyMetrics)
					}
				},
			})
		},
		// (8) Set : Reduces the energy cost of your abilities by 20%.
		8: func(agent core.Agent) {
			druid := agent.(DruidAgent).GetDruid()
			druid.OnSpellRegistered(func(spell *core.Spell) {
				if spell.Cost != nil && spell.Cost.CostType() == core.CostTypeEnergy {
					spell.Cost.Multiplier -= 20
				}
			})
		},
	},
})

var ItemSetWildheartRaiment = core.NewItemSet(core.ItemSet{
	Name: "Wildheart Raiment",
	Bonuses: map[int32]core.ApplyEffect{
		// (2) Set : +10 Resistances/+200 Armor.
		2: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddResistances(10)
			c.AddStat(stats.Armor, 200)
		},
		// (4) Set : +26 Attack Power.
		// (4) Set : Increases damage and healing done by magical spells and effects by up to 15.
		4: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.AttackPower, 26)
			c.AddStat(stats.SpellPower, 15)
		},
		// (6) Set : When struck in combat has a chance of returning 300 mana, 10 rage, or 40 energy to the wearer. (Proc chance: 2%)
		6: func(agent core.Agent) {
			c := agent.GetCharacter()
			actionID := core.ActionID{SpellID: 27781}
			manaMetrics := c.NewManaMetrics(actionID)
			energyMetrics := c.NewEnergyMetrics(actionID)
			rageMetrics := c.NewRageMetrics(actionID)

			core.MakeProcTriggerAura(&c.Unit, core.ProcTrigger{
				Name:       "Nature's Bounty (Mana)",
				Callback:   core.CallbackOnCastComplete,
				ProcMask:   core.ProcMaskSpellDamage | core.ProcMaskSpellHealing,
				ProcChance: 0.02,
				Handler: func(sim *core.Simulation, spell *core.Spell, _ *core.SpellResult) {
					c.AddMana(sim, 300, manaMetrics)
				},
			})
			core.MakeProcTriggerAura(&c.Unit, core.ProcTrigger{
				Name:       "Nature's Bounty (Energy)",
				Callback:   core.CallbackOnSpellHitDealt,
				Outcome:    core.OutcomeLanded,
				ProcMask:   core.ProcMaskMeleeWhiteHit,
				ProcChance: 0.02,
				Handler: func(sim *core.Simulation, spell *core.Spell, _ *core.SpellResult) {
					if c.HasEnergyBar() {
						c.AddEnergy(sim, 40, energyMetrics)
					}
				},
			})
			core.MakeProcTriggerAura(&c.Unit, core.ProcTrigger{
				Name:       "Nature's Bounty (Rage)",
				Callback:   core.CallbackOnSpellHitTaken,
				ProcMask:   core.ProcMaskMelee,
				ProcChance: 0.02,
				Handler: func(sim *core.Simulation, spell *core.Spell, _ *core.SpellResult) {
					if c.HasRageBar() {
						c.AddRage(sim, 10, rageMetrics)
					}
				},
			})
		},
		// (8) Set : +10 Resistances/+200 Armor.
		8: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddResistances(10)
			c.AddStat(stats.Armor, 200)
		},
	},
})

var ItemSetStormrageRaiment = core.NewItemSet(core.ItemSet{
	Name: "Stormrage Raiment",
	Bonuses: map[int32]core.ApplyEffect{
		// (2) Set : Allows 15% of your Mana regeneration to continue while casting.
		2: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.PseudoStats.SpiritRegenRateCasting += .15
		},
		// (4) Set : Increases the duration of your Rejuvenation spell by 3 sec.
		4: func(agent core.Agent) {
			// Nothing to do: Rejuvenation is part of the Restoration healing kit, which is entirely
			// unimplemented in this sim (sim/druid/_restoration is excluded from compilation via its
			// leading underscore, and RegisterRejuvenationSpell doesn't exist), so there's no spell
			// aura duration to extend.
		},
		// (6) Set : Your Healing Touch spell is 30% more effective on targets below 20% health.
		6: func(agent core.Agent) {
			// Nothing to do: Healing Touch is part of the Restoration healing kit, which is entirely
			// unimplemented in this sim (see the (4) comment above), so there's no heal to modify.
		},
		// (8) Set : Reduces the cooldown of your Swiftmend and Nature's Swiftness spells by 25%.
		8: func(agent core.Agent) {
			// Nothing to do: Swiftmend is part of the unimplemented Restoration kit (see above), and
			// Nature's Swiftness is a talent whose cooldown handler is commented out in talents.go
			// (registerNaturesSwiftnessCD), so there's no cooldown to reduce for either.
		},
	},
})

// https://www.wowhead.com/classic/item-set=talonclaw-regalia
var ItemSetTalonclawRegalia = core.NewItemSet(core.ItemSet{
	Name: "Talonclaw Regalia",
	Bonuses: map[int32]core.ApplyEffect{
		// (2) Set : Damage dealt by Thorns increased by 4 and duration increased by 50%.
		2: func(agent core.Agent) {
			// Nothing to do: Thorns is implemented as a permanent (non-expiring) raid buff aura shared by
			// whoever receives it (core.ThornsAura in sim/core/buffs.go), not as a per-caster spell with its
			// own duration or a damage value scoped to the item wearer, so there's nothing here to hook into.
		},
		// (4) Set : Improves your chance to get a critical strike with spells by 2%.
		4: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.SpellCrit, 2*core.SpellCritRatingPerCritChance)
		},
		// (6) Set : Your Hibernate spell is now usable against Humanoid targets.
		6: func(agent core.Agent) {
			// Nothing to do: Hibernate is a crowd-control spell not implemented anywhere in this sim
			// (no registerHibernateSpell exists for druid), so there's no target-type restriction to lift.
		},
		// (8) Set : Increases your Wrath damage against targets afflicted with Insect Swarm and increases
		// your Starfire damage against targets afflicted with Moonfire by 5%.
		8: func(agent core.Agent) {
			// Implemented directly in wrath.go/starfire.go ApplyEffects (conditional baseDamage *= 1.05),
			// since the bonus is conditional on the target's debuff state at the moment damage is
			// calculated and needs to affect DPS metrics correctly.
		},
	},
})

// https://www.wowhead.com/classic/item-set=ursoc-armor
var ItemSetUrsocArmor = core.NewItemSet(core.ItemSet{
	Name: "Ursoc Armor",
	Bonuses: map[int32]core.ApplyEffect{
		// (2) Set : Increases the duration of your Barkskin and Nature's Grasp by 50%.
		2: func(agent core.Agent) {
			// Nothing to do: Barkskin and Nature's Grasp are both part of the Feral Tank kit, which is
			// entirely unimplemented in this sim (registerBarkskinCD is commented out in
			// RegisterFeralTankSpells, and there is no Nature's Grasp spell at all).
		},
		// (4) Set : Your Enrage ability no longer decreases your armor and its cooldown is reduced by 10 sec.
		4: func(agent core.Agent) {
			// Nothing to do: Enrage is not wired into the build (sim/druid/_enrage.go is excluded from
			// compilation via its leading underscore, and registerEnrageSpell is commented out wherever
			// it would otherwise be called), so there's no Enrage spell or cooldown to modify.
		},
		// (6) Set : Reduces the chance for enemies to resist Growl and Challenging Roar by 6%.
		6: func(agent core.Agent) {
			// Nothing to do: Growl and Challenging Roar are tank threat abilities not implemented anywhere
			// in this sim (no registerGrowlSpell/registerChallengingRoarSpell exist for druid).
		},
		// (8) Set : Whenever you are below 20% health, you gain a blessing from Ursoc that will absorb 725
		// damage. This effect can only occur once per minute.
		8: func(agent core.Agent) {
			druid := agent.(DruidAgent).GetDruid()

			actionID := core.ActionID{SpellID: 27779}

			shieldSpell := druid.GetOrRegisterSpell(core.SpellConfig{
				ActionID:    actionID,
				SpellSchool: core.SpellSchoolNature,
				ProcMask:    core.ProcMaskEmpty,
				Flags:       core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell | core.SpellFlagHelpful,

				DamageMultiplier: 1,
				ThreatMultiplier: 1,

				Shield: core.ShieldConfig{
					SelfOnly: true,
					Aura: core.Aura{
						Label:    "Blessing of Ursoc",
						Duration: core.NeverExpires,
					},
				},
			})

			icd := core.Cooldown{
				Timer:    druid.NewTimer(),
				Duration: time.Minute,
			}

			checkAndTrigger := func(sim *core.Simulation) {
				if icd.IsReady(sim) && druid.CurrentHealthPercent() < 0.20 {
					icd.Use(sim)
					shieldSpell.SelfShield().Apply(sim, 725)
				}
			}

			core.MakePermanent(druid.RegisterAura(core.Aura{
				Label: "Ursoc Armor 8pc Trigger",
				OnSpellHitTaken: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
					checkAndTrigger(sim)
				},
				OnPeriodicDamageTaken: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
					checkAndTrigger(sim)
				},
			}))
		},
	},
})

var ItemSetSymbolsOfUnendingLife = core.NewItemSet(core.ItemSet{
	Name: "Symbols of Unending Life",
	Bonuses: map[int32]core.ApplyEffect{
		// (3) Set : Your finishing moves now refund 30 energy on a Miss, Dodge, Block, or Parry.
		3: func(agent core.Agent) {
			c := agent.GetCharacter()
			actionID := core.ActionID{SpellID: 26107}
			energyMetrics := c.NewEnergyMetrics(actionID)
			core.MakeProcTriggerAura(&c.Unit, core.ProcTrigger{
				Name:     "Symbols of Unending Life Finisher Bonus",
				Callback: core.CallbackOnSpellHitDealt,
				Outcome:  core.OutcomeMiss | core.OutcomeDodge | core.OutcomeBlock | core.OutcomeParry,
				Handler: func(sim *core.Simulation, spell *core.Spell, _ *core.SpellResult) {
					if spell.SpellCode == SpellCode_DruidFerociousBite || spell.SpellCode == SpellCode_DruidRip {
						c.AddEnergy(sim, 30, energyMetrics)
					}
				},
			})
		},
	},
})
