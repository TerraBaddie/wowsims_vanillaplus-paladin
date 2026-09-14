package priest

import (
	"slices"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

const (
	// Keep these ordered by ID
	MarkOfTheVeteranFirebird    = 26166
	MarkOfTheVeteranRattlesnake = 26175
	CassandrasTome              = 231509
)

func init() {
	core.AddEffectsToTest = false

	// Keep these ordered by name

	// Equip: Increases spell damage by up to 8% of your total Intellect and
	// healing done by up to 15% of your total Spirit.
	markOfTheVeteranEffect := func(agent core.Agent) {
		priest := agent.(PriestAgent).GetPriest()
		priest.AddStatDependency(stats.Intellect, stats.SpellPower, 0.08)
		priest.AddStatDependency(stats.Spirit, stats.HealingPower, 0.15)
	}
	core.NewItemEffect(MarkOfTheVeteranFirebird, markOfTheVeteranEffect)
	core.NewItemEffect(MarkOfTheVeteranRattlesnake, markOfTheVeteranEffect)

	// https://www.wowhead.com/classic/item=231509/cassandras-tome
	core.NewItemEffect(CassandrasTome, func(agent core.Agent) {
		priest := agent.(PriestAgent).GetPriest()

		actionID := core.ActionID{ItemID: CassandrasTome}
		duration := time.Second * 15
		affectedSpells := []*core.Spell{}

		buffAura := priest.RegisterAura(core.Aura{
			ActionID: actionID,
			Label:    "Cassandra's Tome",
			Duration: duration,
			OnInit: func(aura *core.Aura, sim *core.Simulation) {
				affectedSpells = core.FilterSlice(priest.Spellbook, func(spell *core.Spell) bool {
					return spell.Flags.Matches(SpellFlagPriest) && !spell.Flags.Matches(core.SpellFlagPureDot|core.SpellFlagChanneled)
				})
			},
			OnGain: func(aura *core.Aura, sim *core.Simulation) {
				for _, spell := range affectedSpells {
					spell.BonusCritRating += 100 * core.SpellCritRatingPerCritChance
				}
			},
			OnExpire: func(aura *core.Aura, sim *core.Simulation) {
				for _, spell := range affectedSpells {
					spell.BonusCritRating -= 100 * core.SpellCritRatingPerCritChance
				}
			},
			OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
				if slices.Contains(affectedSpells, spell) {
					aura.Deactivate(sim)
				}
			},
		})

		spell := priest.RegisterSpell(core.SpellConfig{
			ActionID: actionID,
			Flags:    core.SpellFlagNoOnCastComplete | core.SpellFlagOffensiveEquipment,
			Cast: core.CastConfig{
				CD: core.Cooldown{
					Timer:    priest.NewTimer(),
					Duration: time.Minute * 2,
				},
				// Does not seem to share the offensive trinket timer
			},
			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				buffAura.Activate(sim)
			},
		})

		priest.AddMajorCooldown(core.MajorCooldown{
			Spell: spell,
			Type:  core.CooldownTypeDPS,
		})
	})

	core.AddEffectsToTest = true
}
