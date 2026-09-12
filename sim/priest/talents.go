package priest

import (
	"slices"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

func (priest *Priest) ApplyTalents() {
	// Discipline
	priest.registerInnerFocus()
	priest.applyMentalAgility()
	priest.applyForceOfWill()

	if priest.Talents.SilentResolve > 0 {
		// DBC: 10/20/30% threat reduction (all spells).
		priest.PseudoStats.ThreatMultiplier *= 1 - (.10 * float64(priest.Talents.SilentResolve))
	}

	if priest.Talents.ImprovedPowerWordFortitude > 0 {
		priest.MultiplyStat(stats.Stamina, 1.0+.15*float64(priest.Talents.ImprovedPowerWordFortitude))
	}

	// DBC: 10/20/30% of mana regen continues while casting.
	priest.PseudoStats.SpiritRegenRateCasting = []float64{0.0, 0.10, 0.20, 0.30}[priest.Talents.Meditation]

	if priest.Talents.MentalStrength > 0 {
		// DBC: 3/6/9/12/15% Intellect.
		priest.MultiplyStat(stats.Intellect, 1.0+0.03*float64(priest.Talents.MentalStrength))
	}

	if priest.Talents.ImprovedMemory > 0 {
		// DBC: 5/10/15/20/25% casting speed (haste).
		// NOTE: the fixed Shadow APL doesn't currently take advantage of extra
		// haste (Mind Flay ticks aren't hasted and the rotation is timing-tuned),
		// so this can look like a slight DPS loss until the rotation is retuned.
		priest.MultiplyCastSpeed(1.0 + 0.05*float64(priest.Talents.ImprovedMemory))
	}

	priest.applyConcentration()
	priest.applyTwinDisciplines()

	// Holy
	priest.applyInspiration()
	priest.applyHolySpecialization()
	priest.applySearingLight()

	priest.PseudoStats.SchoolDamageTakenMultiplier.MultiplyMagicSchools(1 - 0.02*float64(priest.Talents.SpellWarding))

	if priest.Talents.SpiritualGuidance > 0 {
		// DBC: 10/20% of Spirit as spell power.
		priest.AddStatDependency(stats.Spirit, stats.SpellPower, 0.10*float64(priest.Talents.SpiritualGuidance))
	}

	if priest.Talents.Faith > 0 {
		// DBC: 6/12/18/24/30% Spirit.
		priest.MultiplyStat(stats.Spirit, 1.0+0.06*float64(priest.Talents.Faith))
	}

	if priest.Talents.SpellFocus > 0 {
		// DBC: +2%/rank spell hit (2 ranks).
		bonusHit := 2 * float64(priest.Talents.SpellFocus) * core.SpellHitRatingPerHitChance
		priest.OnSpellRegistered(func(spell *core.Spell) {
			if spell.Flags.Matches(SpellFlagPriest) {
				spell.BonusHitRating += bonusHit
			}
		})
	}

	if priest.Talents.PurifyingLight > 0 {
		// DBC: +50%/rank crit strike damage bonus for Holy spells (2 ranks).
		priest.OnSpellRegistered(func(spell *core.Spell) {
			if spell.Flags.Matches(SpellFlagPriest) && spell.SpellSchool.Matches(core.SpellSchoolHoly) {
				spell.CritDamageBonus += 0.5 * float64(priest.Talents.PurifyingLight)
			}
		})
	}

	// Shadow
	priest.registerVampiricEmbraceSpell()
	priest.registerShadowform()
	priest.applySpiritTap()
	priest.applyShadowAffinity()
	priest.applyShadowFocus()
	priest.applyShadowWeaving()
	priest.applyMindOverlord()
	priest.applyImprovedMindFlay()
	priest.applyBurntSoul()
	priest.applyDarkness()
}

func (priest *Priest) applyMentalAgility() {
	if priest.Talents.MentalAgility == 0 {
		return
	}

	priest.OnSpellRegistered(func(spell *core.Spell) {
		// DBC: 5/10/15% cost reduction on ALL priest spells (not just instants).
		if spell.Cost != nil && spell.Flags.Matches(SpellFlagPriest) {
			spell.Cost.Multiplier -= 5 * priest.Talents.MentalAgility
		}
	})
}

func (priest *Priest) applyForceOfWill() {
	if priest.Talents.ForceOfWill == 0 {
		return
	}

	priest.OnSpellRegistered(func(spell *core.Spell) {
		if spell.Flags.Matches(SpellFlagPriest) {
			// DBC: +2% spell damage per rank, +1% spell crit per rank.
			spell.DamageMultiplierAdditive += 0.02 * float64(priest.Talents.ForceOfWill)
			spell.BonusCritRating += 1 * float64(priest.Talents.ForceOfWill) * core.CritRatingPerCritChance
		}
	})
}

func (priest *Priest) applyHolySpecialization() {
	if priest.Talents.HolySpecialization == 0 {
		return
	}

	priest.OnSpellRegistered(func(spell *core.Spell) {
		if spell.Flags.Matches(SpellFlagPriest) && spell.SpellSchool.Matches(core.SpellSchoolHoly) {
			spell.BonusCritRating += 1 * float64(priest.Talents.HolySpecialization) * core.CritRatingPerCritChance
		}
	})
}

func (priest *Priest) applyInspiration() {
	if priest.Talents.Inspiration == 0 {
		return
	}

	auras := make([]*core.Aura, len(priest.Env.AllUnits))
	for _, unit := range priest.Env.AllUnits {
		if !priest.IsOpponent(unit) {
			aura := core.InspirationAura(unit, priest.Talents.Inspiration)
			auras[unit.UnitIndex] = aura
		}
	}

	priest.RegisterAura(core.Aura{
		Label:    "Inspiration Talent",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnHealDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if slices.Contains([]int32{SpellCode_PriestFlashHeal, SpellCode_PriestHeal, SpellCode_PriestGreaterHeal}, spell.SpellCode) {
				auras[result.Target.UnitIndex].Activate(sim)
			}
		},
	})
}

func (priest *Priest) applySearingLight() {
	if priest.Talents.SearingLight == 0 {
		return
	}

	priest.OnSpellRegistered(func(spell *core.Spell) {
		if spell.SpellCode == SpellCode_PriestSmite || spell.SpellCode == SpellCode_PriestHolyFire {
			// DBC: +3% Smite/Holy Fire damage per rank.
			spell.DamageMultiplierAdditive += 0.03 * float64(priest.Talents.SearingLight)
		}
	})
}

func (priest *Priest) applySpiritTap() {
	if priest.Talents.SpiritTap == 0 {
		return
	}

	// DBC values match (100% Spirit, +50% casting regen, 15s).
	// TODO: model the 50%/100% on-kill proc chance per rank instead of a flat aura.

	spellID := []int32{0, 15270, 15335, 15336, 15337, 15338}[priest.Talents.SpiritTap]
	statDep := priest.NewDynamicMultiplyStat(stats.Spirit, 2.0)

	priest.SpiritTapAura = priest.RegisterAura(core.Aura{
		ActionID: core.ActionID{SpellID: spellID},
		Label:    "Spirit Tap",
		Duration: time.Second * 15,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			priest.EnableDynamicStatDep(sim, statDep)
			priest.PseudoStats.SpiritRegenRateCasting += 0.50
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			priest.DisableDynamicStatDep(sim, statDep)
			priest.PseudoStats.SpiritRegenRateCasting -= 0.50
		},
	})
}

func (priest *Priest) applyShadowAffinity() {
	if priest.Talents.ShadowAffinity == 0 {
		return
	}

	priest.OnSpellRegistered(func(spell *core.Spell) {
		if spell.Flags.Matches(SpellFlagPriest) || spell.SpellSchool.Matches(core.SpellSchoolShadow) {
			// DBC: 10/20/30% Shadow threat reduction.
			spell.ThreatMultiplier *= 1 - 0.10*float64(priest.Talents.ShadowAffinity)
		}
	})
}

func (priest *Priest) applyShadowFocus() {
	if priest.Talents.ShadowFocus == 0 {
		return
	}

	bonusHit := 2 * float64(priest.Talents.ShadowFocus) * core.SpellHitRatingPerHitChance
	priest.OnSpellRegistered(func(spell *core.Spell) {
		if spell.Flags.Matches(SpellFlagPriest) || spell.SpellSchool.Matches(core.SpellSchoolShadow) {
			spell.BonusHitRating += bonusHit
		}
	})
}

func (priest *Priest) applyShadowWeaving() {
	if priest.Talents.ShadowWeaving == 0 {
		return
	}

	priest.ShadowWeavingAuras = priest.NewEnemyAuraArray(func(unit *core.Unit) *core.Aura {
		return core.ShadowWeavingAura(unit, int(priest.Talents.ShadowWeaving))
	})

	procChance := 0.2 * float64(priest.Talents.ShadowWeaving)

	priest.ShadowWeavingProc = priest.GetOrRegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: core.ShadowWeavingSpellIDs[int(priest.Talents.ShadowWeaving)]},
		Flags:       core.SpellFlagNoOnCastComplete | core.SpellFlagNoMetrics,
		SpellSchool: core.SpellSchoolShadow,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealOutcome(sim, target, spell.OutcomeMagicHit)
			if !result.Landed() {
				return
			}

			if procChance == 1.0 || sim.RollWithLabel(0, 1, "ShadowWeaving") < procChance {
				priest.ShadowWeavingAuras.Get(target).Activate(sim)
				priest.ShadowWeavingAuras.Get(target).AddStack(sim)
			}
		},
	})
}

func (priest *Priest) AddShadowWeavingStack(sim *core.Simulation, target *core.Unit) {
	if priest.ShadowWeavingProc == nil {
		return
	}

	priest.ShadowWeavingProc.Cast(sim, target)
}

func (priest *Priest) applyDarkness() {
	if priest.Talents.Darkness == 0 {
		return
	}

	multiplier := 0.02 * float64(priest.Talents.Darkness)

	priest.RegisterAura(core.Aura{
		Label: "Darkness",
		OnInit: func(aura *core.Aura, sim *core.Simulation) {
			baseDamageAffectedSpells := core.FilterSlice(
				core.Flatten(
					[][]*core.Spell{
						priest.MindBlast,
						priest.DevouringPlague,
					},
				),
				func(spell *core.Spell) bool { return spell != nil },
			)

			fullDamageAffectedSpells := core.FilterSlice(
				core.Flatten(
					[][]*core.Spell{
						priest.ShadowWordPain,
					},
				),
				func(spell *core.Spell) bool { return spell != nil },
			)

			for _, spells := range priest.MindFlay {
				fullDamageAffectedSpells = append(
					fullDamageAffectedSpells,
					core.FilterSlice(spells, func(spell *core.Spell) bool { return spell != nil })...,
				)
			}

			for _, spell := range baseDamageAffectedSpells {
				spell.BaseDamageMultiplierAdditive += multiplier
			}

			for _, spell := range fullDamageAffectedSpells {
				spell.DamageMultiplierAdditive += multiplier
			}
		},
	})
}

func (priest *Priest) registerInnerFocus() {
	if !priest.Talents.InnerFocus {
		return
	}

	actionID := core.ActionID{SpellID: 14751}

	priest.InnerFocusAura = priest.RegisterAura(core.Aura{
		Label:    "Inner Focus",
		ActionID: actionID,
		Duration: core.NeverExpires,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range priest.Spellbook {
				if spell.Flags.Matches(SpellFlagPriest) && spell.Cost != nil {
					// DBC: next spell is free and gains +100% crit chance (guaranteed crit).
					spell.Cost.Multiplier -= 100
					spell.BonusCritRating += 100 * core.SpellCritRatingPerCritChance
				}
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range priest.Spellbook {
				if spell.Flags.Matches(SpellFlagPriest) && spell.Cost != nil {
					spell.Cost.Multiplier += 100
					spell.BonusCritRating -= 100 * core.SpellCritRatingPerCritChance
				}
			}
		},
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			if spell.Flags.Matches(SpellFlagPriest) {
				// Remove the buff and put skill on CD
				aura.Deactivate(sim)
				priest.InnerFocus.CD.Use(sim)
				priest.UpdateMajorCooldowns()
			}
		},
	})

	priest.InnerFocus = priest.RegisterSpell(core.SpellConfig{
		ActionID: actionID,
		Flags:    core.SpellFlagNoOnCastComplete | core.SpellFlagAPL,

		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    priest.NewTimer(),
				Duration: time.Minute * 3,
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			priest.InnerFocusAura.Activate(sim)
		},
	})

	priest.AddMajorCooldown(core.MajorCooldown{
		Spell: priest.InnerFocus,
		Type:  core.CooldownTypeDPS,
	})
}

func (priest *Priest) registerShadowform() {
	if !priest.Talents.Shadowform {
		return
	}

	actionID := core.ActionID{SpellID: 15473}

	// TODO: DBC also grants -20% Shadow damage taken (effect 2) - add when a tank/healer priest sim needs it.

	priest.ShadowformAura = priest.RegisterAura(core.Aura{
		Label:    "Shadowform",
		ActionID: actionID,
		Duration: core.NeverExpires,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			// DBC: +20% Shadow damage done.
			aura.Unit.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexShadow] *= 1.20
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexShadow] /= 1.20
		},
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			if spell.SpellSchool.Matches(core.SpellSchoolHoly) {
				aura.Deactivate(sim)
			}
		},
	})

	priest.Shadowform = priest.RegisterSpell(core.SpellConfig{
		ActionID: actionID,
		Flags:    core.SpellFlagNoOnCastComplete | core.SpellFlagAPL,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: 0,
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			priest.ShadowformAura.Activate(sim)
		},
	})
}

// Concentration (DBC): 4/7/10% chance on a damage spell to enter Clearcasting,
// making the next damage spell free (trigger spell 33807 = -100% mana cost).
func (priest *Priest) applyConcentration() {
	if priest.Talents.Concentration == 0 {
		return
	}

	procChance := []float64{0, 0.04, 0.07, 0.10}[priest.Talents.Concentration]

	clearcasting := priest.RegisterAura(core.Aura{
		Label:    "Clearcasting",
		ActionID: core.ActionID{SpellID: 33807},
		Duration: time.Second * 15,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.PseudoStats.SchoolCostMultiplier.AddToMagicSchools(-100)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.PseudoStats.SchoolCostMultiplier.AddToMagicSchools(100)
		},
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			if aura.RemainingDuration(sim) == aura.Duration {
				return
			}
			if spell.Flags.Matches(SpellFlagPriest) && spell.Cost != nil && spell.DefaultCast.Cost > 0 {
				aura.Deactivate(sim)
			}
		},
	})

	priest.RegisterAura(core.Aura{
		Label:    "Concentration Talent",
		Duration: core.NeverExpires,
		OnReset:  func(aura *core.Aura, sim *core.Simulation) { aura.Activate(sim) },
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !result.Landed() || !spell.Flags.Matches(SpellFlagPriest) || !spell.ProcMask.Matches(core.ProcMaskSpellDamage) {
				return
			}
			if sim.Proc(procChance, "Concentration") {
				clearcasting.Activate(sim)
			}
		},
	})
}

// Twin Disciplines (DBC): 10% chance after a Holy damage spell to make the next
// Shadow damage spell free, and vice versa.
func (priest *Priest) applyTwinDisciplines() {
	if priest.Talents.TwinDisciplines == 0 {
		return
	}

	const procChance = 0.10

	makeFreeSpellAura := func(label string, school core.SpellSchool) *core.Aura {
		schoolIdx := school.GetSchoolIndex()
		return priest.RegisterAura(core.Aura{
			Label:    label,
			Duration: time.Second * 15,
			OnGain: func(aura *core.Aura, sim *core.Simulation) {
				aura.Unit.PseudoStats.SchoolCostMultiplier[schoolIdx] -= 100
			},
			OnExpire: func(aura *core.Aura, sim *core.Simulation) {
				aura.Unit.PseudoStats.SchoolCostMultiplier[schoolIdx] += 100
			},
			OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
				if aura.RemainingDuration(sim) == aura.Duration {
					return
				}
				if spell.Flags.Matches(SpellFlagPriest) && spell.SpellSchool.Matches(school) && spell.Cost != nil && spell.DefaultCast.Cost > 0 {
					aura.Deactivate(sim)
				}
			},
		})
	}

	freeShadow := makeFreeSpellAura("Twin Disciplines (Shadow)", core.SpellSchoolShadow)
	freeHoly := makeFreeSpellAura("Twin Disciplines (Holy)", core.SpellSchoolHoly)

	priest.RegisterAura(core.Aura{
		Label:    "Twin Disciplines Talent",
		Duration: core.NeverExpires,
		OnReset:  func(aura *core.Aura, sim *core.Simulation) { aura.Activate(sim) },
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !result.Landed() || !spell.Flags.Matches(SpellFlagPriest) || !spell.ProcMask.Matches(core.ProcMaskSpellDamage) {
				return
			}
			if spell.SpellSchool.Matches(core.SpellSchoolHoly) && sim.Proc(procChance, "Twin Disciplines") {
				freeShadow.Activate(sim)
			} else if spell.SpellSchool.Matches(core.SpellSchoolShadow) && sim.Proc(procChance, "Twin Disciplines") {
				freeHoly.Activate(sim)
			}
		},
	})
}

// Mind Overlord (DBC): reduces the Mana cost of Mind Blast and Mind Flay by
// 6/12/18/24/30% (also Mind Control / Mind Vision, which aren't simmed).
func (priest *Priest) applyMindOverlord() {
	if priest.Talents.MindOverlord == 0 {
		return
	}

	reduction := 6 * priest.Talents.MindOverlord
	priest.OnSpellRegistered(func(spell *core.Spell) {
		if spell.Cost == nil {
			return
		}
		if spell.SpellCode == SpellCode_PriestMindBlast || spell.SpellCode == SpellCode_PriestMindFlay {
			spell.Cost.Multiplier -= reduction
		}
	})
}

// Improved Mind Flay (DBC): 30/60/90% chance to avoid pushback while channeling Mind Flay.
func (priest *Priest) applyImprovedMindFlay() {
	if priest.Talents.ImprovedMindFlay == 0 {
		return
	}

	reduction := 0.30 * float64(priest.Talents.ImprovedMindFlay)
	priest.OnSpellRegistered(func(spell *core.Spell) {
		if spell.SpellCode == SpellCode_PriestMindFlay {
			spell.PushbackReduction += reduction
		}
	})
}

// Burnt Soul (DBC): 10/20/30% chance on cast to regenerate 3% of max mana over 9s
// (the DBC also costs 9% health, which is ignored here).
func (priest *Priest) applyBurntSoul() {
	if priest.Talents.BurntSoul == 0 {
		return
	}

	procChance := 0.10 * float64(priest.Talents.BurntSoul)
	manaMetrics := priest.NewManaMetrics(core.ActionID{SpellID: 33859})

	priest.RegisterAura(core.Aura{
		Label:    "Burnt Soul Talent",
		Duration: core.NeverExpires,
		OnReset:  func(aura *core.Aura, sim *core.Simulation) { aura.Activate(sim) },
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			if !spell.Flags.Matches(SpellFlagPriest) || !spell.ProcMask.Matches(core.ProcMaskSpellDamage) {
				return
			}
			if sim.Proc(procChance, "Burnt Soul") {
				manaPerTick := 0.01 * priest.MaxMana()
				core.StartPeriodicAction(sim, core.PeriodicActionOptions{
					Period:   time.Second * 3,
					NumTicks: 3,
					OnAction: func(sim *core.Simulation) {
						priest.AddMana(sim, manaPerTick, manaMetrics)
					},
				})
			}
		},
	})
}
