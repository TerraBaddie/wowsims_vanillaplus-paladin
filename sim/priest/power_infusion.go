package priest

import (
	"github.com/wowsims/classic/sim/core"
)

// Power Infusion (DBC spell 10060): +15% spell damage and healing done for 30s, 3 min cooldown.
// TODO: allow casting Power Infusion on another raid member instead of only self-buffing.
func (priest *Priest) registerPowerInfusionCD() {
	if !priest.Talents.PowerInfusion {
		return
	}

	actionID := core.ActionID{SpellID: 10060, Tag: priest.Index}
	piAura := core.PowerInfusionAura(&priest.Unit, priest.Index)

	piSpell := priest.RegisterSpell(core.SpellConfig{
		ActionID: actionID,
		Flags:    core.SpellFlagNoOnCastComplete | core.SpellFlagAPL,

		ManaCost: core.ManaCostOptions{
			BaseCost: 0.16,
		},

		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    priest.NewTimer(),
				Duration: core.PowerInfusionCD,
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			piAura.Activate(sim)
		},
	})

	priest.AddMajorCooldown(core.MajorCooldown{
		Spell: piSpell,
		Type:  core.CooldownTypeDPS,
	})
}
