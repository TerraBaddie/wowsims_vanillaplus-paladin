package paladin

import (
	"math"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

func (paladin *Paladin) ApplyTalents() {
	// Precision: +1% melee & spell hit per rank.
	paladin.AddStat(stats.MeleeHit, float64(paladin.Talents.Precision)*core.MeleeHitRatingPerHitChance)
	paladin.AddStat(stats.SpellHit, float64(paladin.Talents.Precision)*core.SpellHitRatingPerHitChance)

	// Conviction: +1% crit with attacks & offensive spells per rank.
	paladin.AddStat(stats.MeleeCrit, float64(paladin.Talents.Conviction)*core.CritRatingPerCritChance)
	paladin.AddStat(stats.SpellCrit, float64(paladin.Talents.Conviction)*core.SpellCritRatingPerCritChance)

	if paladin.Talents.Toughness > 0 {
		paladin.ApplyEquipScaling(stats.Armor, 1.0+0.02*float64(paladin.Talents.Toughness))
	}

	// These are no-op if untalented.
	paladin.MultiplyStat(stats.Strength, 1.0+0.03*float64(paladin.Talents.DivineStrength))
	paladin.MultiplyStat(stats.Intellect, 1.0+0.03*float64(paladin.Talents.DivineIntellect))
	paladin.AddStat(stats.Defense, 2*float64(int32(0))) // TODO: custom tree has no Anticipation

	// Shield Specialization bonus is additive. NOTE: Total SBV will be inflated until
	// https://github.com/wowsims/sod/issues/1025 gets resolved.
	paladin.PseudoStats.BlockValueMultiplier += 0.15 * float64(paladin.Talents.ShieldSpecialization)

	paladin.AddStat(stats.Parry, 2*float64(paladin.Talents.Deflection))

	paladin.applyWeaponSpecialization()
	if paladin.Talents.Vengeance > 0 {
		paladin.applyVengeance()
	}
	if paladin.Talents.Vindication > 0 {
		paladin.applyVindication()
	}
	// Holy Power: +2% Holy spell crit per rank.
	paladin.PseudoStats.SchoolBonusCritChance[stats.SchoolIndexHoly] += 2 * core.SpellCritRatingPerCritChance * float64(paladin.Talents.HolyPower)

	// Searing Light: +4% Holy damage per rank (-4% healing, not modeled).
	if paladin.Talents.SearingLight > 0 {
		paladin.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexHoly] *= 1 + 0.04*float64(paladin.Talents.SearingLight)
	}

	// Crusade: spell damage & healing increased by up to 6% of total Strength per rank.
	if paladin.Talents.Crusade > 0 {
		paladin.AddStatDependency(stats.Strength, stats.SpellPower, 0.06*float64(paladin.Talents.Crusade))
	}

	// Blessed Strikes: attacks ignore up to 180 armor per rank.
	if paladin.Talents.BlessedStrikes > 0 {
		paladin.AddStat(stats.ArmorPenetration, 180*float64(paladin.Talents.BlessedStrikes))
	}

	// Inevitable Justice: +50% crit chance to all Judgements.
	if paladin.Talents.InevitableJustice {
		paladin.OnSpellRegistered(func(spell *core.Spell) {
			switch spell.SpellCode {
			case SpellCode_PaladinJudgementOfCommand, SpellCode_PaladinJudgementOfRighteousness:
				spell.BonusCritRating += 50 * core.SpellCritRatingPerCritChance
			}
		})
	}

	paladin.applyRedoubt()
	paladin.applyReckoning()
	paladin.applyImprovedLayOnHands()
}

func (paladin *Paladin) improvedSoR() float64 {
	return []float64{1, 1.05, 1.10, 1.15}[paladin.Talents.ImprovedSealOfRighteousness]
}

func (paladin *Paladin) benediction() int32 {
	return []int32{100, 90, 80, 70}[paladin.Talents.Benediction]
}

func (paladin *Paladin) applyRedoubt() {
	if paladin.Talents.Redoubt == 0 {
		return
	}

	// Redoubt grants 6% block chance per point.
	blockBonus := 6.0 * float64(paladin.Talents.Redoubt) * core.BlockRatingPerBlockChance

	paladin.redoubtAura = paladin.RegisterAura(core.Aura{
		Label:     "Redoubt",
		ActionID:  core.ActionID{SpellID: 20134},
		Duration:  time.Second * 10,
		MaxStacks: 5,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			paladin.AddStatDynamic(sim, stats.Block, blockBonus)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			paladin.AddStatDynamic(sim, stats.Block, -blockBonus)
		},
		OnSpellHitTaken: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if result.DidBlock() {
				aura.RemoveStack(sim)
			}
		},
	})

	paladin.RegisterAura(core.Aura{
		Label:    "Redoubt Crit Trigger",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitTaken: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if result.DidCrit() && spell.ProcMask.Matches(core.ProcMaskMeleeOrRanged) {
				paladin.redoubtAura.Activate(sim)
				paladin.redoubtAura.SetStacks(sim, 5)
			}
		},
	})
}

func (paladin *Paladin) applyReckoning() {

	if paladin.Talents.Reckoning == 0 {
		return
	}

	procID := core.ActionID{SpellID: 20178} // Reckoning Proc ID
	procChance := 0.2 * float64(paladin.Talents.Reckoning)

	core.MakeProcTriggerAura(&paladin.Unit, core.ProcTrigger{
		Name:       "Reckoning Crit Trigger",
		Callback:   core.CallbackOnSpellHitTaken,
		Outcome:    core.OutcomeCrit,
		ProcMask:   core.ProcMaskMeleeOrRanged,
		ProcChance: procChance,
		Handler: func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			paladin.AutoAttacks.ExtraMHAttack(sim, 1, procID, spell.ActionID)
		},
	})
}

func (paladin *Paladin) getWeaponSpecializationModifier() float64 {
	handType := paladin.MainHand().HandType
	if handType == proto.HandType_HandTypeMainHand || handType == proto.HandType_HandTypeOneHand {
		return 1. + 0.02*float64(paladin.Talents.OneHandedWeaponSpecialization)
	} else if handType == proto.HandType_HandTypeTwoHand {
		// Current VanillaPlus value after the nerf: +2% damage per rank.
		return 1. + 0.02*float64(paladin.Talents.TwoHandedWeaponSpecialization)
	} else {
		return 1.
	}
}

// Affects all physical damage or spells that can be rolled as physical.
func (paladin *Paladin) applyWeaponSpecialization() {
	paladin.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical] *= paladin.getWeaponSpecializationModifier()
}

func (paladin *Paladin) applyVengeance() {
	if paladin.Talents.Vengeance == 0 {
		return
	}

	// Vengeance: crit gives 20% chance per rank to gain a stack of +2% damage dealt,
	// stacking up to 10 times, lasting 15 sec.
	procChance := 0.2 * float64(paladin.Talents.Vengeance)
	const perStack = 1.02

	procAura := paladin.RegisterAura(core.Aura{
		Label:     "Vengeance Proc",
		ActionID:  core.ActionID{SpellID: 20059},
		Duration:  time.Second * 15,
		MaxStacks: 10,
		OnStacksChange: func(aura *core.Aura, sim *core.Simulation, oldStacks int32, newStacks int32) {
			mult := math.Pow(perStack, float64(newStacks-oldStacks))
			aura.Unit.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexHoly] *= mult
			aura.Unit.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical] *= mult
		},
	})

	paladin.RegisterAura(core.Aura{
		Label:    "Vengeance",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if result.DidCrit() && (procChance >= 1 || sim.RandomFloat("Vengeance") < procChance) {
				procAura.Activate(sim)
				procAura.AddStack(sim)
			}
		},
	})
}

func (paladin *Paladin) applyVindication() {
	if paladin.Talents.Vindication == 0 {
		return
	}
	//vindicationMultiplier := []float64{1, 1.05, 1.10, 1.15}[paladin.Talents.Vengeance]
	vindicationMultiplier := []*stats.StatDependency{
		paladin.NewDynamicMultiplyStat(stats.AttackPower, 1.00),
		paladin.NewDynamicMultiplyStat(stats.AttackPower, 1.05),
		paladin.NewDynamicMultiplyStat(stats.AttackPower, 1.10),
		paladin.NewDynamicMultiplyStat(stats.AttackPower, 1.15),
	}

	vindicationAura := paladin.RegisterAura(core.Aura{
		Label:    "Vindication Proc",
		ActionID: core.ActionID{SpellID: 26021},
		Duration: time.Second * 30,
		OnInit: func(aura *core.Aura, sim *core.Simulation) {
			paladin.EnableDynamicStatDep(sim, vindicationMultiplier[0])
		},
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			paladin.EnableDynamicStatDep(sim, vindicationMultiplier[paladin.Talents.Vindication])
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			paladin.DisableDynamicStatDep(sim, vindicationMultiplier[paladin.Talents.Vindication])
		},
	})
	// \tvindicationAuras := paladin.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
	// \t\treturn core.VindicationAura(target, paladin.Talents.Vindication)
	// \t})
	paladin.RegisterAura(core.Aura{
		Label:    "Vindication Talent",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			// TODO: Replace with actual proc mask / proc chance
			if result.Landed() && spell.ProcMask.Matches(core.ProcMaskMelee) {
				vindicationAura.Activate(sim)
			}
		},
	})
}

func (paladin *Paladin) applyImprovedLayOnHands() {

	if paladin.Talents.ImprovedLayOnHands > 0 {

		armorMultiplier := []float64{1, 1.15, 1.3}[paladin.Talents.ImprovedLayOnHands]
		auraID := []int32{0, 20233, 20236}[paladin.Talents.ImprovedLayOnHands]

		paladin.RegisterAura(core.Aura{
			Label:    "Lay on Hands",
			ActionID: core.ActionID{SpellID: auraID},
			Duration: time.Minute * 2,
			OnGain: func(aura *core.Aura, sim *core.Simulation) {
				paladin.ApplyDynamicEquipScaling(sim, stats.Armor, armorMultiplier)
			},
			OnExpire: func(aura *core.Aura, sim *core.Simulation) {
				paladin.RemoveDynamicEquipScaling(sim, stats.Armor, armorMultiplier)
			},
			OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
				if spell.SpellCode == SpellCode_PaladinLayOnHands {
					aura.Activate(sim)
				}
			},
		})
	}
}
