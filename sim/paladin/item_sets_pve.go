package paladin

import (
	"slices"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

///////////////////////////////////////////////////////////////////////////
//                            Classic Phase 1 Item Sets - Molten Core
///////////////////////////////////////////////////////////////////////////

var ItemSetVestmentsOfProphecy = core.NewItemSet(core.ItemSet{
	Name: "Lawbringer Armor",
	Bonuses: map[int32]core.ApplyEffect{
		// Improves your critical strike chance for all attacks and spells by 1%.
		2: func(agent core.Agent) {
			paladin := agent.(PaladinAgent).GetPaladin()
			paladin.AddStat(stats.MeleeCrit, 1)
			paladin.AddStat(stats.SpellCrit, 1)
		},
		// Increases the chance of triggering a Judgement of Light heal by 20%.
		4: func(agent core.Agent) {
			// Nothing to do: Judgement of Light's heal proc isn't modeled
			// anywhere in this sim (there is no Holy Light or Judgement of
			// Light healing implementation in sim/paladin at all, as this
			// Paladin sim only simulates damage output), so there's no proc
			// chance to buff.
		},
		// Reduces the cost of your Holy Light by 5%.
		6: func(agent core.Agent) {
			// Nothing to do: Holy Light is not implemented anywhere in
			// sim/paladin (this Paladin sim only simulates damage output),
			// so there is no spell cost to reduce.
		},
		// Increases spell damage and healing by up to 20% of your total Intellect.
		8: func(agent core.Agent) {
			paladin := agent.(PaladinAgent).GetPaladin()
			paladin.AddStatDependency(stats.Intellect, stats.SpellPower, 0.20)
		},
	},
})

var ItemSetRighteousArmor = core.NewItemSet(core.ItemSet{
	Name: "Righteous Armor",
	Bonuses: map[int32]core.ApplyEffect{
		// Improves your chance to hit with your Judgements by 5%.
		2: func(agent core.Agent) {
			paladin := agent.(PaladinAgent).GetPaladin()
			bonusHit := 5 * float64(core.SpellHitRatingPerHitChance)

			core.MakePermanent(paladin.RegisterAura(core.Aura{
				Label: "Improved Judgement Hit - Righteous Armor 2P Bonus",
				OnGain: func(aura *core.Aura, sim *core.Simulation) {
					for _, spellsJoX := range paladin.allJudgeSpells {
						for _, judgeSpell := range spellsJoX {
							if judgeSpell != nil {
								judgeSpell.BonusHitRating += bonusHit
							}
						}
					}
				},
				OnExpire: func(aura *core.Aura, sim *core.Simulation) {
					for _, spellsJoX := range paladin.allJudgeSpells {
						for _, judgeSpell := range spellsJoX {
							if judgeSpell != nil {
								judgeSpell.BonusHitRating -= bonusHit
							}
						}
					}
				},
			}))
		},
		// Increases the damage done by your Retribution Aura by 6.
		4: func(agent core.Agent) {
			// Nothing to do: Retribution Aura is only modeled in this sim as an
			// externally-applied raid buff (core.RetributionAura in sim/core/buffs.go)
			// granted to party/raid members. The Paladin's own personal aura selection
			// (proto.PaladinAura_RetributionAura) is never wired up to a self-inflicted
			// "damage attackers" mechanic anywhere in sim/paladin, so there is no
			// simulated instance of this Paladin's own Retribution Aura to buff.
		},
		// Gives Paladin a chance on every melee hit to heal your party for 189 to 211.
		6: func(agent core.Agent) {
			// Nothing to do: matches the identical tooltip text on Lawbringer Armor's
			// 8-piece bonus above, which is likewise a no-op in this codebase. A party
			// heal proc has no effect on this Paladin's own simulated combat metrics,
			// and no proc rate is specified anywhere in the tooltip data to model it.
		},
		// Reduces the mana cost of all your spells by 20% when your Mana drops below 20%.
		8: func(agent core.Agent) {
			// Nothing to do: spell costs in this sim (SpellCost.GetCurrentCost) are
			// computed once at finalize() and cached as DefaultCast.Cost; there is no
			// dynamic, resource-threshold-based cost recalculation mechanism anywhere
			// in sim/core for a "while below X% mana" style conditional cost discount.
		},
	},
})

///////////////////////////////////////////////////////////////////////////
//                            Classic Phase 3 Item Sets - BWL
///////////////////////////////////////////////////////////////////////////

var ItemSetSoulforgeArmor = core.NewItemSet(core.ItemSet{
	Name: "Judgement Armor",
	Bonuses: map[int32]core.ApplyEffect{
		// Increases the duration of your Judgements by 20%.
		2: func(agent core.Agent) {
			// Nothing to do: Judgement is modeled as an instant dummy spell
			// with no debuff/duration of its own (sim/paladin/judgement.go) -
			// individual Seals carry their own effects instead. There is no
			// simulated "Judgement duration" for this bonus to extend.
		},
		// Increases damage and healing done by magical spells and effects by up to 25.
		4: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.SpellPower, 25)
		},
		// 50% chance to regain 220 mana when you cast a Judgement.
		6: func(agent core.Agent) {
			paladin := agent.(PaladinAgent).GetPaladin()
			actionID := core.ActionID{SpellID: 27164}
			manaMetrics := paladin.NewManaMetrics(actionID)

			paladin.RegisterAura(core.Aura{
				Label: "Judgement - T3 - Paladin - 6P Bonus",
				OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
					if spell == paladin.judgement && sim.Proc(0.5, "Judgement Armor 6pc") {
						paladin.AddMana(sim, 220, manaMetrics)
					}
				},
			})
		},
		// Inflicts 80 to 137 additional Holy damage on the target of a Paladin's Judgement.
		8: func(agent core.Agent) {
			paladin := agent.(PaladinAgent).GetPaladin()

			spellCodes := []int32{SpellCode_PaladinJudgementOfCommand, SpellCode_PaladinJudgementOfRighteousness}
			paladin.RegisterAura(core.Aura{
				Label: "Judgement - T3 - Paladin - 8P Bonus",
				OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
					if slices.Contains(spellCodes, spell.SpellCode) && result.Landed() {
						spell.CalcAndDealDamage(sim, result.Target, sim.Roll(80, 137), spell.OutcomeMagicCrit)
					}
				},
			})
		},
	},
})

///////////////////////////////////////////////////////////////////////////
//                            Classic Phase 4 Item Sets - ZG and AB
///////////////////////////////////////////////////////////////////////////

var ItemSetConfessorsRaiment = core.NewItemSet(core.ItemSet{
	Name: "Freethinker's Armor",
	Bonuses: map[int32]core.ApplyEffect{
		// Increases the effect of all Blessings by 10%.
		2: func(agent core.Agent) {
			// Nothing to do: only Blessing of Sanctuary is simulated as a
			// self-applied effect on this Paladin (blessing_of_sanctuary.go),
			// and it isn't a generic "all Blessings" magnitude scalar this sim
			// can hook into. The other Blessings (Might/Wisdom/Kings/etc.) are
			// modeled purely as external raid buffs applied to party members,
			// with no simulated instance of this Paladin casting/receiving a
			// scalable "Blessing effect" value.
		},
		// Reduces the casting time of your Holy Light spell by 0.1 sec.
		3: func(agent core.Agent) {
			// Nothing to do
		},
		// Increases healing done by spells and effects by up to 70.
		5: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.HealingPower, 70)
		},
	},
})

///////////////////////////////////////////////////////////////////////////
//                            Classic Phase 5 Item Sets - AQ
///////////////////////////////////////////////////////////////////////////

var ItemSetGarmentsOfTheOracle = core.NewItemSet(core.ItemSet{
	Name: "Avenger's Battlegear",
	Bonuses: map[int32]core.ApplyEffect{
		// Increases the duration of your Judgements by 20%.
		3: func(agent core.Agent) {
			// Nothing to do
		},
		// Increases damage and healing done by magical spells and effects by up to 71.
		5: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.SpellPower, 71)
		},
	},
})

///////////////////////////////////////////////////////////////////////////
//                            Classic Phase 6 Item Sets - Naxx
///////////////////////////////////////////////////////////////////////////

var ItemSetVestmentsOfFaith = core.NewItemSet(core.ItemSet{
	Name: "Redemption Armor",
	Bonuses: map[int32]core.ApplyEffect{
		// Increases the amount healed by your Judgement of Light by 20.
		2: func(agent core.Agent) {
			// Nothing to do
		},
		// Reduces cooldown on your Lay on Hands by 12 min.
		4: func(agent core.Agent) {
			// Nothing to do
		},
		// Your Flash of Light and Holy Light spells have a chance to imbue your target with Holy Power.
		6: func(agent core.Agent) {
			// Nothing to do
		},
		// Your Cleanse spell also heals the target for 200.
		8: func(agent core.Agent) {
			// Nothing to do
		},
	},
})
