package hunter

import (
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

///////////////////////////////////////////////////////////////////////////
//                            Classic Phase 2
///////////////////////////////////////////////////////////////////////////

// https://www.wowhead.com/classic/item-set=543/champions-pursuance
var ItemSetChampionsPursuance = core.NewItemSet(core.ItemSet{
	Name: "Champion's Pursuance",
	Bonuses: map[int32]core.ApplyEffect{
		// +20 Stamina.
		2: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.Stamina, 20)
		},
		// Reduces the cooldown of your Concussive Shot by 1 sec.
		4: func(agent core.Agent) {
			// Nothing to do
		},
		// +20 Agility.
		6: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.Agility, 20)
		},
	},
})

// https://www.wowhead.com/classic/item-set=550/lieutenant-commanders-pursuance
var ItemSetLieutenantCommandersPursuance = core.NewItemSet(core.ItemSet{
	Name: "Lieutenant Commander's Pursuance",
	Bonuses: map[int32]core.ApplyEffect{
		// +20 Stamina.
		2: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.Stamina, 20)
		},
		// Reduces the cooldown of your Concussive Shot by 1 sec.
		4: func(agent core.Agent) {
			// Nothing to do
		},
		// +20 Agility.
		6: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.Agility, 20)
		},
	},
})

///////////////////////////////////////////////////////////////////////////
//                            Classic Phase 3
///////////////////////////////////////////////////////////////////////////

var ItemSetWarlordsPursuit = core.NewItemSet(core.ItemSet{
	Name: "Warlord's Pursuit",
	Bonuses: map[int32]core.ApplyEffect{
		// +20 Agi
		2: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.Agility, 20)
		},
		// Reduces the cooldown of your Concussive Shot by 1 sec.
		3: func(agent core.Agent) {
			// Nothing to do
		},
		// 20 Stamina
		6: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.Stamina, 20)
		},
	},
})

var ItemSetFieldMarshalsPursuit = core.NewItemSet(core.ItemSet{
	Name: "Field Marshal's Pursuit",
	Bonuses: map[int32]core.ApplyEffect{
		// +20 Agi
		2: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.Agility, 20)
		},
		// Reduces the cooldown of your Concussive Shot by 1 sec.
		3: func(agent core.Agent) {
			// Nothing to do
		},
		// 20 stamina
		6: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.Stamina, 20)
		},
	},
})