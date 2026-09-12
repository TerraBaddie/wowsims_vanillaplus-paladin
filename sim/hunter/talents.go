package hunter

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

func (hunter *Hunter) ApplyTalents() {
	if hunter.pet != nil {
		hunter.applyFrenzy()
		hunter.registerBestialWrathCD()

		hunter.pet.AddStat(stats.MeleeCrit, core.CritRatingPerCritChance*3*float64(hunter.Talents.Ferocity))
		hunter.pet.AddStat(stats.SpellCrit, core.SpellCritRatingPerCritChance*3*float64(hunter.Talents.Ferocity))

		hunter.pet.PseudoStats.DamageDealtMultiplier *= 1 + 0.02*float64(hunter.Talents.UnleashedFury)

		if hunter.Talents.EnduranceTraining > 0 {
			hunter.pet.MultiplyStat(stats.Health, 1+(0.10*float64(hunter.Talents.EnduranceTraining)))
		}
	}

	if int32(0) /*MonsterSlaying removed*/+int32(0) /*HumanoidSlaying removed*/ > 0 {
		hunter.Env.RegisterPostFinalizeEffect(func() {
			for _, t := range hunter.Env.Encounter.Targets {
				switch t.MobType {
				case proto.MobType_MobTypeHumanoid:
					multiplier := []float64{1, 1.01, 1.02, 1.03}[int32(0) /*HumanoidSlaying removed*/]
					for _, at := range hunter.AttackTables[t.UnitIndex] {
						at.DamageDealtMultiplier *= multiplier
						at.CritMultiplier *= multiplier
					}
				case proto.MobType_MobTypeBeast, proto.MobType_MobTypeGiant, proto.MobType_MobTypeDragonkin:
					multiplier := []float64{1, 1.01, 1.02, 1.03}[int32(0) /*MonsterSlaying removed*/]
					for _, at := range hunter.AttackTables[t.UnitIndex] {
						at.DamageDealtMultiplier *= multiplier
						at.CritMultiplier *= multiplier
					}
				}
			}
		})
	}

	if hunter.Talents.BestialDiscipline > 0 {
		core.MakePermanent(hunter.RegisterAura(core.Aura{
			Label: "Bestial Discipline",
			OnInit: func(aura *core.Aura, sim *core.Simulation) {
				if hunter.pet != nil {
					hunter.pet.AddFocusRegenMultiplier(0.5 * float64(hunter.Talents.BestialDiscipline))
				}
			},
		}))
	}

	hunter.AddStat(stats.MeleeHit, float64(int32(0) /*Surefooted removed*/)*1*core.MeleeHitRatingPerHitChance)
	hunter.AddStat(stats.SpellHit, float64(int32(0) /*Surefooted removed*/)*1*core.SpellHitRatingPerHitChance)

	if hunter.Talents.KillerInstinct > 0 {
		// Killer Instinct: you and your pet gain 1% hit and crit chance per rank.
		ki := float64(hunter.Talents.KillerInstinct)
		hunter.AddStats(stats.Stats{
			stats.MeleeCrit:  ki * core.CritRatingPerCritChance,
			stats.SpellCrit:  ki * core.SpellCritRatingPerCritChance,
			stats.MeleeHit:   ki * core.MeleeHitRatingPerHitChance,
			stats.SpellHit:   ki * core.SpellHitRatingPerHitChance,
		})
		if hunter.pet != nil {
			hunter.pet.AddStats(stats.Stats{
				stats.MeleeCrit: ki * core.CritRatingPerCritChance,
				stats.MeleeHit:  ki * core.MeleeHitRatingPerHitChance,
			})
		}
	}

	if hunter.Talents.LethalShots > 0 {
		lethalBonus := 1 * float64(hunter.Talents.LethalShots) * core.CritRatingPerCritChance
		hunter.OnSpellRegistered(func(spell *core.Spell) {
			if spell.Flags.Matches(SpellFlagShot) {
				spell.BonusCritRating += lethalBonus
			}
		})
		hunter.AutoAttacks.RangedConfig().BonusCritRating += lethalBonus
	}

	if hunter.Talents.RangedWeaponSpecialization > 0 {
		mult := 1 + 0.02*float64(hunter.Talents.RangedWeaponSpecialization)
		hunter.OnSpellRegistered(func(spell *core.Spell) {
			if spell.ProcMask.Matches(core.ProcMaskRanged) && spell.SpellCode != SpellCode_HunterSerpentSting {
				spell.DamageMultiplier *= mult
			}
		})
	}

	if hunter.Talents.Survivalist > 0 {
		hunter.MultiplyStat(stats.Health, 1.0+0.02*float64(hunter.Talents.Survivalist))
	}

	if hunter.Talents.LightningReflexes > 0 {
		bonus := 0.03 * float64(hunter.Talents.LightningReflexes)
		hunter.MultiplyStat(stats.Agility, 1.0+bonus)
		hunter.PseudoStats.RangedSpeedMultiplier *= 1.0 + bonus
		hunter.PseudoStats.MeleeSpeedMultiplier *= 1.0 + bonus
	}

	hunter.applyEfficiency()
	hunter.applyTrapMastery()
	hunter.applyCleverTraps()
	hunter.applyReconnaissance()
	hunter.applySavageFlurry()
	hunter.applyBrutality()
}

func (hunter *Hunter) applyReconnaissance() {
	if hunter.Talents.Reconnaissance == 0 {
		return
	}
	// +3% damage per rank while standing still ~10s. The sim hunter is stationary, so it is always active.
	hunter.PseudoStats.DamageDealtMultiplier *= 1 + 0.03*float64(hunter.Talents.Reconnaissance)
}

func (hunter *Hunter) applySavageFlurry() {
	if hunter.Talents.SavageFlurry == 0 {
		return
	}
	bonus := 1 + 0.03*float64(hunter.Talents.SavageFlurry)
	hunter.PseudoStats.MeleeSpeedMultiplier *= bonus
	hunter.PseudoStats.RangedSpeedMultiplier *= bonus
	if hunter.pet != nil {
		hunter.pet.PseudoStats.MeleeSpeedMultiplier *= bonus
	}
}

func (hunter *Hunter) applyBrutality() {
	if hunter.Talents.Brutality == 0 {
		return
	}
	crit := float64(hunter.Talents.Brutality) * core.CritRatingPerCritChance
	hunter.AddStat(stats.MeleeCrit, crit)
	if hunter.pet != nil {
		hunter.pet.AddStat(stats.MeleeCrit, crit)
	}
}

func (hunter *Hunter) applyFrenzy() {
	if hunter.Talents.Frenzy == 0 {
		return
	}

	procChance := 0.2 * float64(hunter.Talents.Frenzy)

	procAura := hunter.pet.RegisterAura(core.Aura{
		Label:    "Frenzy Proc",
		ActionID: core.ActionID{SpellID: 19625},
		Duration: time.Second * 15,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.MultiplyAttackSpeed(sim, 1.3)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.MultiplyAttackSpeed(sim, 1/1.3)
		},
	})

	hunter.pet.RegisterAura(core.Aura{
		Label:    "Frenzy",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, spellResult *core.SpellResult) {
			if !spellResult.Outcome.Matches(core.OutcomeCrit) {
				return
			}
			if procChance == 1 || sim.RandomFloat("Frenzy") < procChance {
				procAura.Activate(sim)
			}
		},
	})
}

func (hunter *Hunter) registerBestialWrathCD() {
	if !hunter.Talents.BestialWrath {
		return
	}

	actionID := core.ActionID{SpellID: 19574}

	hunter.BestialWrathPetAura = hunter.pet.RegisterAura(core.Aura{
		Label:    "Bestial Wrath Pet",
		ActionID: actionID,
		Duration: time.Second * 18,
	}).AttachMultiplicativePseudoStatBuff(&hunter.pet.PseudoStats.DamageDealtMultiplier, 1.5)

	bwSpell := hunter.RegisterSpell(core.SpellConfig{
		ActionID: actionID,
		Flags:    core.SpellFlagAPL,

		ManaCost: core.ManaCostOptions{
			BaseCost: 0.12,
		},

		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    hunter.NewTimer(),
				Duration: time.Minute * 2,
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			hunter.BestialWrathPetAura.Activate(sim)
		},
	})

	hunter.AddMajorCooldown(core.MajorCooldown{
		Spell: bwSpell,
		Type:  core.CooldownTypeDPS,
	})
}

func (hunter *Hunter) mortalShots() float64 {
	return 0.10 * float64(hunter.Talents.MortalShots)
}

func (hunter *Hunter) applyTrapMastery() {
	if int32(0) /*TrapMastery removed*/ == 0 {
		return
	}

	hunter.OnSpellRegistered(func(spell *core.Spell) {
		if spell.Flags.Matches(SpellFlagTrap) {
			spell.BonusHitRating += 5 * float64(int32(0) /*TrapMastery removed*/)
		}
	})
}

func (hunter *Hunter) applyCleverTraps() {
	if hunter.Talents.CleverTraps == 0 {
		return
	}

	hunter.OnSpellRegistered(func(spell *core.Spell) {
		if spell.Flags.Matches(SpellFlagTrap) {
			spell.DamageMultiplier *= 1 + 0.15*float64(hunter.Talents.CleverTraps)
		}
	})
}

func (hunter *Hunter) applyEfficiency() {
	hunter.OnSpellRegistered(func(spell *core.Spell) {
		// applies to Stings, Shots, and Volley
		if spell.Cost != nil && spell.Flags.Matches(SpellFlagSting|SpellFlagShot) || spell.SpellCode == SpellCode_HunterVolley {
			spell.Cost.Multiplier -= 3 * hunter.Talents.Efficiency
		}
	})
}
