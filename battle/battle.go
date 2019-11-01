package battle

import (
	"subway/models"
	"subway/tool"
)

const (
	BattleResultWin   int8 = 1
	BattleResultLose  int8 = 2
	BattleResultEqual int8 = 3
)

type BattleResult struct {
	Result int8 // 1 胜利  2 失败   3  平局
	Items  []*BattleItem
}

type BattleItem struct {
	MilliSeconds int32
	FromHero     ReportHero
	ToHeros      []ReportHero
	Skill        ReportSkill
}

// 用于战斗记录
type ReportHero struct {
	HeroId  string
	HP      int32
	Effect  *BattleInfo
	Deffect *BattleInfo
}

type ReportSkill struct {
	SkillId string
}

func BattleGuanKa(uid string, gkId int) *BattleResult {

	// 获取关卡阵容
	gkHeros := make([]*models.Hero, 0)
	for _, h := range models.GetGuanKa(gkId).Heros {
		gkHeros = append(gkHeros, h.Hero)
	}
	// 获取阵容
	selfHeros := models.GetSelectedHeros(uid)

	res := Battle(selfHeros, gkHeros)

	if res.Result == BattleResultWin {
		u, _ := models.GetUser(uid)
		u.SetGuanKaId(gkId)
	}

	return res
}

func Battle(heros1 []*models.Hero, heros2 []*models.Hero) *BattleResult {
	result := BattleResult{Items: []*BattleItem{}}

	if heros1 == nil || heros2 == nil {
		return &result
	}

	context := NewBattleContext()
	battleInitialize(context, heros1, heros2)
	// 计算装备加成
	executeEquipEffect(context)
	// 执行被动技能
	executePassiveSkill(context)
	// 执行主动技能
	res := executeActiveSkill(context)

	result.Result = res
	result.Items = context.Items

	return &result
}

// 战斗初始化
func battleInitialize(context *BattleContext, heros1 []*models.Hero, heros2 []*models.Hero) {
	selfHeros := make([]*Hero, 0)
	otherHeros := make([]*Hero, 0)

	for _, h := range heros1 {
		hh := new(models.Hero)
		tool.Clone(h, hh)
		selfHeros = append(selfHeros, &Hero{Hero: hh, Group: 1, Runing: BattleInfo{}, MaxHP: hh.Props.HP, MaxMP: MaxMP})
	}

	for _, h := range heros2 {
		hh := new(models.Hero)
		tool.Clone(h, hh)
		otherHeros = append(otherHeros, &Hero{Hero: hh, Group: 2, Runing: BattleInfo{}, MaxHP: hh.Props.HP, MaxMP: MaxMP})
	}

	context.SelfHeros = selfHeros
	context.OtherHeros = otherHeros
}

// 计算装备加成
func executeEquipEffect(context *BattleContext) {
	for _, h := range context.SelfHeros {
		for _, e := range h.Equips {
			equipEffect(h, e)
		}
	}

	for _, h := range context.OtherHeros {
		for _, e := range h.Equips {
			equipEffect(h, e)
		}
	}
}

// 执行被动技能
func executePassiveSkill(context *BattleContext) {
	context.MilliSeconds = -1

	executeHeroSkills(context.SelfHeros, context)
	executeHeroSkills(context.OtherHeros, context)
}

// 执行一个英雄的技能
func executeHeroSkills(heros []*Hero, context *BattleContext) {
	for _, h := range heros {

		if !canExecute(h) {
			executeDebuffer(h)
			continue
		}

		// 普攻
		ExecuteSkill(h, nil, context)

		if !canExecuteSkill(h) {
			executeDebuffer(h)
			continue
		}
		for _, s := range h.Skills {
			ExecuteSkill(h, &Skill{s}, context)
		}
	}
}

func canExecute(h *Hero) bool {
	return h.Runing.Dizzy == 0
}

func canExecuteSkill(h *Hero) bool {
	return h.Runing.Silence == 0
}

func executeDebuffer(h *Hero) {
	h.SetDizzy(h.Runing.Dizzy - BattleLogicRate)
	h.SetSilence(h.Runing.Silence - BattleLogicRate)
}

// 执行主动技能
func executeActiveSkill(context *BattleContext) int8 {
	milliSeconds := BattleLogicRate
	result := BattleResultEqual
	for isLive(context.SelfHeros) && isLive(context.OtherHeros) {
		// 执行技能
		context.MilliSeconds = milliSeconds
		executeHeroSkills(context.SelfHeros, context)

		// 如果对面没人了
		if !isLive(context.OtherHeros) {
			result = BattleResultWin
			break
		}

		executeHeroSkills(context.OtherHeros, context)

		milliSeconds += BattleLogicRate
	}
	// 如果己方没人了
	if !isLive(context.SelfHeros) {
		result = BattleResultLose
	}

	return result
}

func equipEffect(h *Hero, e *models.Equip) {
	if e.Status == models.EquipStatusWearOff {
		return
	}

	h.Props.HP += e.Info.HP
	h.MaxHP += e.Info.HP
	h.MaxMP -= e.Info.MP

	h.Props.Strength += e.Info.Strength
	if h.Info.Type == models.HeroTypeStrength {
		if h.Props.AD > 0 {
			h.Props.AD += e.Info.Strength * 2
		}
		if h.Props.AP > 0 {
			h.Props.AP += e.Info.Strength * 2
		}
	}
	h.Props.Agility += e.Info.Agility
	if h.Info.Type == models.HeroTypeAgility {
		if h.Props.AD > 0 {
			h.Props.AD += e.Info.Agility * 2
		}
		if h.Props.AP > 0 {
			h.Props.AP += e.Info.Agility * 2
		}
	}
	h.Props.Intelligent += e.Info.Intelligent
	if h.Info.Type == models.HeroTypeIntelligent {
		if h.Props.AD > 0 {
			h.Props.AD += e.Info.Intelligent * 2
		}
		if h.Props.AP > 0 {
			h.Props.AP += e.Info.Intelligent * 2
		}
	}

	h.Props.AD += e.Info.AD
	h.Props.ADDef += e.Info.ADDef
}

func isLive(heros []*Hero) bool {
	if heros == nil {
		return false
	}
	for _, hero := range heros {
		if hero.Props.HP > 0 {
			return true
		}
	}
	return false
}
