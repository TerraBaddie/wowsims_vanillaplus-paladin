package hunter

import (
	"time"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)


///////////////////////////////////////////////////////////////////////////
//                            Phase 1 Item Sets - Molten Core
///////////////////////////////////////////////////////////////////////////

var ItemSetGiantStalkers = core.NewItemSet(core.ItemSet{
	Name: "Giantstalker Armor",
	Bonuses: map[int32]core.ApplyEffect{
		// (2) Set: Increases the range of your Mend Pet spell by 50% and the effect by 10%. Also reduces the cost by 30%.
		2: func(agent core.Agent) {
			// Mend Pet is not implemented in sim
		},
		// (4) Set: Increases your pet's stamina by 40 and all spell resistances by 60.
		4: func(agent core.Agent) {
			hunter := agent.(HunterAgent).GetHunter()
			if hunter.pet == nil {
				return
			}
			core.MakePermanent(hunter.RegisterAura(core.Aura{
				Label: "Nature's Ally",
				OnInit: func(aura *core.Aura, sim *core.Simulation) {
					hunter.pet.AddStatDynamic(sim, stats.Stamina, 40)
					hunter.pet.AddResistancesDynamic(sim, 60)
				},
			}))
		},
		// (6) Set: Increases the duration of your Rapid Strikes and Rapid Fire by 5 secs.
		6: func(agent core.Agent) {
			hunter := agent.(HunterAgent).GetHunter()
			// Rapid Strikes is not implemented in sim; only extend Rapid Fire's duration.
			core.MakePermanent(hunter.RegisterAura(core.Aura{
				Label: "Giantstalker Rapid Fire Duration",
				OnInit: func(aura *core.Aura, sim *core.Simulation) {
					hunter.RapidFireAura.Duration += time.Second * 5
				},
			}))
		},
		// (7) Set: Increases your damage against Giants and Elementals by 5%.
		7: func(agent core.Agent) {
			hunter := agent.(HunterAgent).GetHunter()
			hunter.Env.RegisterPostFinalizeEffect(func() {
				for _, target := range hunter.Env.Encounter.Targets {
					if target.MobType == proto.MobType_MobTypeGiant || target.MobType == proto.MobType_MobTypeElemental {
						for _, at := range hunter.AttackTables[target.UnitIndex] {
							at.DamageDealtMultiplier *= 1.05
						}
					}
				}
			})
		},
		// (8) Set: Reduces the cost of your spells by 20%.
		8: func(agent core.Agent) {
			hunter := agent.(HunterAgent).GetHunter()
			hunter.OnSpellRegistered(func(spell *core.Spell) {
				// Matches the same set of abilities ("spells") as the Efficiency talent: Shots, Stings, and Volley.
				if spell.Cost != nil && (spell.Flags.Matches(SpellFlagSting|SpellFlagShot) || spell.SpellCode == SpellCode_HunterVolley) {
					spell.Cost.Multiplier -= 20.0
				}
			})
		},
	},
})

///////////////////////////////////////////////////////////////////////////
//                            Phase 2 Item Sets - Dire Maul
///////////////////////////////////////////////////////////////////////////

///////////////////////////////////////////////////////////////////////////
//                            Phase 3 Item Sets - BWL
///////////////////////////////////////////////////////////////////////////

var ItemSetDragonstalkersArmor = core.NewItemSet(core.ItemSet{
	Name: "Dragonstalker Armor",
	Bonuses: map[int32]core.ApplyEffect{
		// (2) Set: Increases the effect of your Aspect of the Hawk and Aspect of the Monkey by 20%.
		2: func(agent core.Agent) {
			hunter := agent.(HunterAgent).GetHunter()
			// Aspect of the Monkey is not implemented in sim; only Aspect of the Hawk's bonus is applied.
			core.MakePermanent(hunter.RegisterAura(core.Aura{
				Label: "Improved Aspect of the Hawk",
				OnInit: func(aura *core.Aura, sim *core.Simulation) {
					hunter.AspectOfTheHawkAPMultiplier += 0.20
				},
			}))
		},
		// (4) Set: Increases Attack Power by 50 for both you and your pet.
		4: func(agent core.Agent) {
			hunter := agent.(HunterAgent).GetHunter()
			hunter.AddStats(stats.Stats{
				stats.AttackPower:       50,
				stats.RangedAttackPower: 50,
			})
			if hunter.pet == nil {
				return
			}

			core.MakePermanent(hunter.RegisterAura(core.Aura{
				Label: "Dragonstalker's Ally",
				OnInit: func(aura *core.Aura, sim *core.Simulation) {
					hunter.pet.AddStatsDynamic(sim, stats.Stats{
						stats.AttackPower:       50,
						stats.RangedAttackPower: 50,
					})
				},
			}))
		},
		// (6) Set: Increases the damage of Multi-shot and Volley by 15%.
		6: func(agent core.Agent) {
			hunter := agent.(HunterAgent).GetHunter()
			hunter.RegisterAura(core.Aura{
				Label: "Improved Volley and Multishot",
				OnInit: func(aura *core.Aura, sim *core.Simulation) {
					hunter.Volley.BaseDamageMultiplierAdditive += 0.15
					hunter.MultiShot.BaseDamageMultiplierAdditive += 0.15
				},
			})
		},
		// (7) Set: Increases your damage against Dragonkins by 5%.
		7: func(agent core.Agent) {
			hunter := agent.(HunterAgent).GetHunter()
			hunter.Env.RegisterPostFinalizeEffect(func() {
				for _, target := range hunter.Env.Encounter.Targets {
					if target.MobType == proto.MobType_MobTypeDragonkin {
						for _, at := range hunter.AttackTables[target.UnitIndex] {
							at.DamageDealtMultiplier *= 1.05
						}
					}
				}
			})
		},
		// (8) Set: You have a chance whenever you deal ranged damage to apply an Expose Weakness effect to the target. Expose Weakness increases the Ranged Attack Power of all attackers against that target by 450 for 7 sec.
		8: func(agent core.Agent) {
			hunter := agent.(HunterAgent).GetHunter()
			
			debuffAuras := hunter.NewEnemyAuraArray(core.ExposeWeaknessAura)

			core.MakeProcTriggerAura(&hunter.Unit, core.ProcTrigger{
				Name:     "T2 - Hunter - Ranged 8P Bonus Trigger",
				Callback: core.CallbackOnSpellHitDealt,
				Outcome:  core.OutcomeLanded,
				ProcMask: core.ProcMaskRanged,
				PPM:      0.5,
				Handler: func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
					debuffAuras.Get(result.Target).Activate(sim)
				},
			})
		},
	},
})

var ItemSetPredatorsArmor = core.NewItemSet(core.ItemSet{
	Name: "Predator's Armor",
	Bonuses: map[int32]core.ApplyEffect{
		// (2) Set: +20 Attack Power.
		2: func(agent core.Agent) {
			hunter := agent.(HunterAgent).GetHunter()
			hunter.AddStats(stats.Stats{
				stats.AttackPower:       20,
				stats.RangedAttackPower: 20,
			})
		},
		// (3) Set: Decreases the cooldown of Concussive Shot by 1 sec.
		3: func(agent core.Agent) {
			// Concussive Shot is not implemented in sim
		},
		// (5) Set: Increases the duration of Viper and Serpent Stings by 3 sec.
		5: func(agent core.Agent) {
			hunter := agent.(HunterAgent).GetHunter()

			// Viper Sting is not implemented in sim; only Serpent Sting's duration is extended.
			core.MakePermanent(hunter.RegisterAura(core.Aura{
				Label: "Improved Serpent Sting",
				OnInit: func(aura *core.Aura, sim *core.Simulation) {
					for _, dot := range hunter.SerpentSting.Dots() {
						if dot != nil {
							dot.NumberOfTicks += 1
							dot.RecomputeAuraDuration()
						}
					}
				},
			}))
		},
	},
})

///////////////////////////////////////////////////////////////////////////
//                            Phase 4 Item Sets - AQ
///////////////////////////////////////////////////////////////////////////

// hhttps://www.wowhead.com/classic/item-set=515/beastmaster-armor
var ItemSetBeastmasterArmor = core.NewItemSet(core.ItemSet{
	Name: "Beastmaster Armor",
	Bonuses: map[int32]core.ApplyEffect{
		// (2) Set: +10 Resistances/+200 Armor.
		2: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddResistances(10)
			c.AddStat(stats.Armor, 200)
		},
		// (4) Set: Your attacks have a 5% chance of restoring 200 mana.
		4: func(agent core.Agent) {
			c := agent.GetCharacter()
			actionID := core.ActionID{SpellID: 27785}
			manaMetrics := c.NewManaMetrics(actionID)

			core.MakeProcTriggerAura(&c.Unit, core.ProcTrigger{
				ActionID:   actionID,
				Name:       "Hunter Armor Energize",
				Callback:   core.CallbackOnSpellHitDealt,
				Outcome:    core.OutcomeLanded,
				ProcMask:   core.ProcMaskWhiteHit,
				ProcChance: 0.05,
				Handler: func(sim *core.Simulation, spell *core.Spell, _ *core.SpellResult) {
					if c.HasManaBar() {
						c.AddMana(sim, 200, manaMetrics)
					}
				},
			})
		},
		// (6) Set: +40 Attack Power.
		6: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStats(stats.Stats{
				stats.AttackPower:       40,
				stats.RangedAttackPower: 40,
			})
		},
		// (8) Set: +10 Resistances/+200 Armor.
		8: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddResistances(10)
			c.AddStat(stats.Armor, 200)
		},
	},
})

// https://www.wowhead.com/classic/item-set=509/strikers-garb
var ItemSetStrikersGarb = core.NewItemSet(core.ItemSet{
	Name: "Striker's Garb",
	Bonuses: map[int32]core.ApplyEffect{
		// (3) Set : Reduces the cost of your Arcane Shots by 10%. 
		3: func(agent core.Agent) {
			hunter := agent.(HunterAgent).GetHunter()
			core.MakePermanent(hunter.RegisterAura(core.Aura{
				Label: "Striker's Arcane Shot Bonus",
				OnInit: func(aura *core.Aura, sim *core.Simulation) {
					if hunter.ArcaneShot != nil {
						hunter.ArcaneShot.Cost.Multiplier -= 10.0
					}
				},
			}))
		},
		// (5) Set : Reduces the cooldown of your Rapid Fire ability by 2 minutes. 
		5: func(agent core.Agent) {
			hunter := agent.(HunterAgent).GetHunter()
			core.MakePermanent(hunter.RegisterAura(core.Aura{
				Label: "Striker's Rapid Bonus",
				OnInit: func(aura *core.Aura, sim *core.Simulation) {
					hunter.RapidFire.CD.Duration -= time.Minute * 2
				},
			}))
		},
	},
})

///////////////////////////////////////////////////////////////////////////
//                            Phase 5 Item Sets - Naxx
///////////////////////////////////////////////////////////////////////////

// https://www.wowhead.com/classic/item-set=530/cryptstalker-armor
var ItemSetCryptstalkerArmor = core.NewItemSet(core.ItemSet{
	Name: "Cryptstalker Armor",
	Bonuses: map[int32]core.ApplyEffect{
		// (2) Set : Increases the duration of your Rapid Fire by 4 secs.
		2: func(agent core.Agent) {
			hunter := agent.(HunterAgent).GetHunter()
			core.MakePermanent(hunter.RegisterAura(core.Aura{
				Label: "Rapid Fire Duration",
				OnInit: func(aura *core.Aura, sim *core.Simulation) {
					hunter.RapidFireAura.Duration += time.Second * 4
				},
			}))
		},
		// (4) Set : Increases Attack Power by 50 for both you and your pet.
		4: func(agent core.Agent) {
			hunter := agent.(HunterAgent).GetHunter()
			hunter.AddStats(stats.Stats{
				stats.AttackPower:       50,
				stats.RangedAttackPower: 50,
			})
			if hunter.pet == nil {
				return
			}

			core.MakePermanent(hunter.RegisterAura(core.Aura{
				Label: "Stalker's Ally",
				OnInit: func(aura *core.Aura, sim *core.Simulation) {
					hunter.pet.AddStatsDynamic(sim, stats.Stats{
						stats.AttackPower:       50,
						stats.RangedAttackPower: 50,
					})
				},
			}))
		},
		// (6) Set : Your ranged critical hits cause an Adrenaline Rush, granting you 50 mana.
		6: func(agent core.Agent) {
			hunter := agent.(HunterAgent).GetHunter()
			actionID := core.ActionID{SpellID: 28753}
			manaMetrics := hunter.NewManaMetrics(actionID)

			hunter.RegisterAura(core.Aura{
				Label:    "Adrenaline Rush",
				Duration: core.NeverExpires,
				OnReset: func(aura *core.Aura, sim *core.Simulation) {
					aura.Activate(sim)
				},
				OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
					if spell.ProcMask.Matches(core.ProcMaskRanged) && result.DidCrit() {
						hunter.AddMana(sim, 50, manaMetrics)
					}
				},
			})
			
		},
		// (8) Set : Reduces the mana cost of your Multi-Shot and Aimed Shot by 20.
		8: func(agent core.Agent) {
			hunter := agent.(HunterAgent).GetHunter()
			core.MakePermanent(hunter.RegisterAura(core.Aura{
				Label: "Cryptstalker Aimed and Multishot Bonus",
				OnInit: func(aura *core.Aura, sim *core.Simulation) {
					if hunter.AimedShot != nil {
						hunter.AimedShot.Cost.FlatModifier -= 20.0
					}
					hunter.MultiShot.Cost.FlatModifier -= 20.0
				},
			}))
		},
	},
})
