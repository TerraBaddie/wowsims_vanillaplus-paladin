package priest

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
)

const MindFlayRanks = 6
const MindFlayTicks = 5

var MindFlaySpellId = [MindFlayRanks + 1]int32{0, 15407, 17311, 17312, 17313, 17314, 18807}
var MindFlayTickSpellId = [MindFlayRanks + 1]int32{0, 16568, 7378, 17316, 17317, 17318, 18808}

// Private server extends Mind Flay to a 5-tick/5-sec channel (retail Classic is 3).
// Confirmed via CSV's/Spell.csv: EffectAmplitude=1000ms, DurationIndex=7 (5000ms),
// consistently across all 6 ranks. Total damage = (EffectBasePoints+1) * 5 ticks.
var MindFlayBaseDamage = [MindFlayRanks + 1]float64{0, 160, 275, 400, 550, 700, 900}
var MindFlayManaCost = [MindFlayRanks + 1]float64{0, 45, 70, 100, 135, 165, 205}
var MindFlayLevel = [MindFlayRanks + 1]int{0, 20, 28, 36, 44, 52, 60}

func (priest *Priest) registerMindFlay() {
	if !priest.Talents.MindFlay {
		return
	}

	priest.MindFlay = make([][]*core.Spell, MindFlayRanks+1)

	for rank := 1; rank <= MindFlayRanks; rank++ {
		priest.MindFlay[rank] = make([]*core.Spell, MindFlayTicks+1)

		var tick int32
		for tick = 0; tick < MindFlayTicks; tick++ {
			config := priest.newMindFlaySpellConfig(rank, tick)

			if config.RequiredLevel <= int(priest.Level) {
				priest.MindFlay[rank][tick] = priest.RegisterSpell(config)
			}
		}
	}
}

func (priest *Priest) newMindFlaySpellConfig(rank int, tickIdx int32) core.SpellConfig {
	ticks := tickIdx
	flags := SpellFlagPriest | core.SpellFlagChanneled | core.SpellFlagBinary
	if tickIdx == 0 {
		ticks = MindFlayTicks
		flags |= core.SpellFlagAPL
	}

	spellId := MindFlaySpellId[rank]
	baseDamage := MindFlayBaseDamage[rank] / float64(ticks)
	manaCost := MindFlayManaCost[rank]
	level := MindFlayLevel[rank]

	spellCoeff := 0.15 // classic penalty for mf having a slow effect

	tickLength := time.Second

	return core.SpellConfig{
		SpellCode:   SpellCode_PriestMindFlay,
		ActionID:    core.ActionID{SpellID: spellId}.WithTag(tickIdx),
		SpellSchool: core.SpellSchoolShadow,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellDamage,
		Flags:       flags,

		RequiredLevel: level,
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: fmt.Sprintf("MindFlay-%d-%d", rank, tickIdx),
			},
			NumberOfTicks:       ticks,
			TickLength:          tickLength,
			AffectedByCastSpeed: false,
			BonusCoefficient:    spellCoeff,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, baseDamage, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHit)
			if result.Landed() {
				priest.AddShadowWeavingStack(sim, target)
				spell.Dot(target).Apply(sim)
			}
			spell.DealOutcome(sim, result)
		},

		ExpectedTickDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, _ bool) *core.SpellResult {
			baseDamage := baseDamage / MindFlayTicks
			result := spell.CalcPeriodicDamage(sim, target, baseDamage, spell.OutcomeExpectedMagicAlwaysHit)
			return result
		},
	}
}
