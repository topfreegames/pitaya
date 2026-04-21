package game

import (
	"fmt"
	"sync"
	"time"
)

type SkillType int32

const (
	SkillTypeActive SkillType = 0
	SkillTypePassive SkillType = 1
)

type Skill struct {
	ID           int32
	Name         string
	Type         SkillType
	Cooldown     time.Duration
	ManaCost     int32
	Damage       int32
	Range        float64
	Radius       float64
	Duration     time.Duration
}

type SkillSystem struct {
	roomID  string
	skills  map[int32]*Skill
	mu      sync.RWMutex
}

func NewSkillSystem(roomID string) *SkillSystem {
	ss := &SkillSystem{
		roomID: roomID,
		skills: make(map[int32]*Skill),
	}

	ss.initializeSkills()
	return ss
}

func (ss *SkillSystem) initializeSkills() {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.skills[1] = &Skill{
		ID:       1,
		Name:     "Fireball",
		Type:     SkillTypeActive,
		Cooldown: 5 * time.Second,
		ManaCost: 20,
		Damage:   50,
		Range:    30.0,
		Radius:   5.0,
		Duration: 0,
	}

	ss.skills[2] = &Skill{
		ID:       2,
		Name:     "Heal",
		Type:     SkillTypeActive,
		Cooldown: 10 * time.Second,
		ManaCost: 30,
		Damage:   -30,
		Range:    20.0,
		Radius:   5.0,
		Duration: 0,
	}

	ss.skills[3] = &Skill{
		ID:       3,
		Name:     "Speed Boost",
		Type:     SkillTypeActive,
		Cooldown: 15 * time.Second,
		ManaCost: 25,
		Damage:   0,
		Range:    0,
		Radius:   0,
		Duration: 5 * time.Second,
	}
}

func (ss *SkillSystem) GetSkill(skillID int32) (*Skill, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	skill, exists := ss.skills[skillID]
	if !exists {
		return nil, fmt.Errorf("skill not found: %d", skillID)
	}

	return skill, nil
}

func (ss *SkillSystem) ValidateUseSkill(playerID string, skillID int32, playerMana int32, lastUsed map[int32]time.Time) error {
	skill, err := ss.GetSkill(skillID)
	if err != nil {
		return err
	}

	if skill.Type != SkillTypeActive {
		return fmt.Errorf("skill is not active")
	}

	if playerMana < skill.ManaCost {
		return fmt.Errorf("not enough mana")
	}

	if lastUsed, exists := lastUsed[skillID]; exists {
		if time.Since(lastUsed) < skill.Cooldown {
			return fmt.Errorf("skill is on cooldown")
		}
	}

	return nil
}

func (ss *SkillSystem) UseSkill(skillID int32, playerMana int32, lastUsed map[int32]time.Time) (int32, time.Duration, error) {
	skill, err := ss.GetSkill(skillID)
	if err != nil {
		return 0, 0, err
	}

	if err := ss.ValidateUseSkill("", skillID, playerMana, lastUsed); err != nil {
		return 0, 0, err
	}

	newMana := playerMana - skill.ManaCost
	lastUsed[skillID] = time.Now()

	return newMana, skill.Cooldown, nil
}

func (ss *SkillSystem) GetCooldownRemaining(skillID int32, lastUsed map[int32]time.Time) time.Duration {
	skill, err := ss.GetSkill(skillID)
	if err != nil {
		return 0
	}

	if lastUsed, exists := lastUsed[skillID]; exists {
		elapsed := time.Since(lastUsed)
		remaining := skill.Cooldown - elapsed
		if remaining > 0 {
			return remaining
		}
	}

	return 0
}

func (ss *SkillSystem) GetActiveSkills() []*Skill {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	var activeSkills []*Skill
	for _, skill := range ss.skills {
		if skill.Type == SkillTypeActive {
			activeSkills = append(activeSkills, skill)
		}
	}

	return activeSkills
}

func (ss *SkillSystem) CalculateSkillDamage(skillID int32, baseDamage int32) int32 {
	skill, err := ss.GetSkill(skillID)
	if err != nil {
		return baseDamage
	}

	return baseDamage + skill.Damage
}
