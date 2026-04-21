package game

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/topfreegames/pitaya/v3/game/internal/proto"
)

const (
	BaseAttackPower = 10
	BaseDefensePower = 5
	CriticalHitChance = 0.1
	CriticalHitMultiplier = 2.0
	AttackCooldown = 1000
)

type CombatSystem struct {
	roomID string
}

func NewCombatSystem(roomID string) *CombatSystem {
	return &CombatSystem{
		roomID: roomID,
	}
}

type CombatResult struct {
	Success      bool
	DamageDealt  int32
	TargetKilled bool
	NewScore     int32
}

func (cs *CombatSystem) CalculateDamage(attacker, defender *CombatStats) int32 {
	baseDamage := attacker.AttackPower - defender.DefensePower
	if baseDamage < 1 {
		baseDamage = 1
	}

	if rand.Float64() < CriticalHitChance {
		baseDamage = int32(float64(baseDamage) * CriticalHitMultiplier)
	}

	return baseDamage
}

func (cs *CombatSystem) ValidateAttack(attackerID, targetID string, attackerPos, targetPos *proto.Position, timestamp int64) error {
	if attackerID == targetID {
		return fmt.Errorf("cannot attack yourself")
	}

	if attackerPos == nil || targetPos == nil {
		return fmt.Errorf("invalid position")
	}

	distance := cs.CalculateDistance(attackerPos, targetPos)
	if distance > 50.0 {
		return fmt.Errorf("target out of range")
	}

	if time.Now().Unix()-timestamp > 5 {
		return fmt.Errorf("attack request too old")
	}

	return nil
}

func (cs *CombatSystem) CalculateDistance(pos1, pos2 *proto.Position) float64 {
	dx := float64(pos2.X - pos1.X)
	dy := float64(pos2.Y - pos1.Y)
	dz := float64(pos2.Z - pos1.Z)
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func (cs *CombatSystem) ProcessAttack(attacker, defender *CombatStats) *CombatResult {
	damage := cs.CalculateDamage(attacker, defender)
	defender.CurrentHealth -= damage

	result := &CombatResult{
		Success:      true,
		DamageDealt:  damage,
		TargetKilled: defender.CurrentHealth <= 0,
	}

	if result.TargetKilled {
		result.NewScore = attacker.Score + 100
		attacker.Score = result.NewScore
	}

	return result
}

type CombatStats struct {
	PlayerID     string
	AttackPower  int32
	DefensePower int32
	CurrentHealth int32
	MaxHealth    int32
	Score        int32
	Position     *proto.Position
	LastAttack   time.Time
}

func (cs *CombatSystem) CanAttack(stats *CombatStats) bool {
	return time.Since(stats.LastAttack) >= time.Duration(AttackCooldown)*time.Millisecond
}

func (cs *CombatSystem) UpdateLastAttack(stats *CombatStats) {
	stats.LastAttack = time.Now()
}
