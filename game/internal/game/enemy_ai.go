package game

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/topfreegames/pitaya/v3/game/internal/proto"
)

type EnemyBehavior int32

const (
	BehaviorAggressive EnemyBehavior = 0
	BehaviorDefensive EnemyBehavior = 1
	BehaviorPassive   EnemyBehavior = 2
	BehaviorPatrol    EnemyBehavior = 3
)

type EnemyAI struct {
	EnemyID     string
	EnemyType   string
	Behavior    EnemyBehavior
	Position    *proto.Position
	Health      int32
	MaxHealth   int32
	AttackPower int32
	Speed       float64
	Range       float64
	TargetID    string
	LastAction  time.Time
	Path        []*proto.Position
	PathIndex   int
}

type EnemyAISystem struct {
	roomID  string
	enemies map[string]*EnemyAI
	mu      sync.RWMutex
}

func NewEnemyAISystem(roomID string) *EnemyAISystem {
	return &EnemyAISystem{
		roomID:  roomID,
		enemies: make(map[string]*EnemyAI),
	}
}

func (eas *EnemyAISystem) AddEnemy(enemy *EnemyAI) error {
	eas.mu.Lock()
	defer eas.mu.Unlock()

	if _, exists := eas.enemies[enemy.EnemyID]; exists {
		return fmt.Errorf("enemy already exists: %s", enemy.EnemyID)
	}

	eas.enemies[enemy.EnemyID] = enemy
	return nil
}

func (eas *EnemyAISystem) RemoveEnemy(enemyID string) {
	eas.mu.Lock()
	defer eas.mu.Unlock()

	delete(eas.enemies, enemyID)
}

func (eas *EnemyAISystem) GetEnemy(enemyID string) (*EnemyAI, error) {
	eas.mu.RLock()
	defer eas.mu.RUnlock()

	enemy, exists := eas.enemies[enemyID]
	if !exists {
		return nil, fmt.Errorf("enemy not found: %s", enemyID)
	}

	return enemy, nil
}

func (eas *EnemyAISystem) UpdateAllEnemies(players map[string]*proto.Position, deltaTime time.Duration) {
	eas.mu.Lock()
	defer eas.mu.Unlock()

	for _, enemy := range eas.enemies {
		eas.updateEnemy(enemy, players, deltaTime)
	}
}

func (eas *EnemyAISystem) updateEnemy(enemy *EnemyAI, players map[string]*proto.Position, deltaTime time.Duration) {
	switch enemy.Behavior {
	case BehaviorAggressive:
		eas.updateAggressiveEnemy(enemy, players, deltaTime)
	case BehaviorDefensive:
		eas.updateDefensiveEnemy(enemy, players, deltaTime)
	case BehaviorPassive:
		eas.updatePassiveEnemy(enemy, players, deltaTime)
	case BehaviorPatrol:
		eas.updatePatrolEnemy(enemy, players, deltaTime)
	}

	enemy.LastAction = time.Now()
}

func (eas *EnemyAISystem) updateAggressiveEnemy(enemy *EnemyAI, players map[string]*proto.Position, deltaTime time.Duration) {
	closestPlayerID, closestDistance := eas.findClosestPlayer(enemy, players)

	if closestPlayerID != "" && closestDistance <= enemy.Range {
		enemy.TargetID = closestPlayerID
		eas.moveTowardsTarget(enemy, players[closestPlayerID], deltaTime)
	} else {
		enemy.TargetID = ""
	}
}

func (eas *EnemyAISystem) updateDefensiveEnemy(enemy *EnemyAI, players map[string]*proto.Position, deltaTime time.Duration) {
	closestPlayerID, closestDistance := eas.findClosestPlayer(enemy, players)

	if closestPlayerID != "" && closestDistance <= enemy.Range/2 {
		enemy.TargetID = closestPlayerID
		eas.moveTowardsTarget(enemy, players[closestPlayerID], deltaTime)
	} else {
		enemy.TargetID = ""
	}
}

func (eas *EnemyAISystem) updatePassiveEnemy(enemy *EnemyAI, players map[string]*proto.Position, deltaTime time.Duration) {
	closestPlayerID, closestDistance := eas.findClosestPlayer(enemy, players)

	if closestPlayerID != "" && closestDistance <= enemy.Range/4 {
		enemy.TargetID = closestPlayerID
		eas.moveAwayFromTarget(enemy, players[closestPlayerID], deltaTime)
	} else {
		enemy.TargetID = ""
	}
}

func (eas *EnemyAISystem) updatePatrolEnemy(enemy *EnemyAI, players map[string]*proto.Position, deltaTime time.Duration) {
	if len(enemy.Path) == 0 {
		return
	}

	if enemy.PathIndex >= len(enemy.Path) {
		enemy.PathIndex = 0
	}

	targetPos := enemy.Path[enemy.PathIndex]
	eas.moveTowardsPosition(enemy, targetPos, deltaTime)

	distance := eas.calculateDistance(enemy.Position, targetPos)
	if distance < 1.0 {
		enemy.PathIndex++
	}
}

func (eas *EnemyAISystem) findClosestPlayer(enemy *EnemyAI, players map[string]*proto.Position) (string, float64) {
	var closestPlayerID string
	var closestDistance float64 = math.MaxFloat64

	for playerID, playerPos := range players {
		distance := eas.calculateDistance(enemy.Position, playerPos)
		if distance < closestDistance {
			closestDistance = distance
			closestPlayerID = playerID
		}
	}

	return closestPlayerID, closestDistance
}

func (eas *EnemyAISystem) moveTowardsTarget(enemy *EnemyAI, targetPos *proto.Position, deltaTime time.Duration) {
	eas.moveTowardsPosition(enemy, targetPos, deltaTime)
}

func (eas *EnemyAISystem) moveTowardsPosition(enemy *EnemyAI, targetPos *proto.Position, deltaTime time.Duration) {
	dx := float64(targetPos.X - enemy.Position.X)
	dy := float64(targetPos.Y - enemy.Position.Y)
	dz := float64(targetPos.Z - enemy.Position.Z)

	distance := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if distance == 0 {
		return
	}

	speed := enemy.Speed * deltaTime.Seconds()
	if speed > distance {
		speed = distance
	}

	enemy.Position.X += float32((dx / distance) * speed)
	enemy.Position.Y += float32((dy / distance) * speed)
	enemy.Position.Z += float32((dz / distance) * speed)
}

func (eas *EnemyAISystem) moveAwayFromTarget(enemy *EnemyAI, targetPos *proto.Position, deltaTime time.Duration) {
	dx := float64(enemy.Position.X - targetPos.X)
	dy := float64(enemy.Position.Y - targetPos.Y)
	dz := float64(enemy.Position.Z - targetPos.Z)

	distance := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if distance == 0 {
		return
	}

	speed := enemy.Speed * deltaTime.Seconds()
	if speed > distance {
		speed = distance
	}

	enemy.Position.X += float32((dx / distance) * speed)
	enemy.Position.Y += float32((dy / distance) * speed)
	enemy.Position.Z += float32((dz / distance) * speed)
}

func (eas *EnemyAISystem) calculateDistance(pos1, pos2 *proto.Position) float64 {
	dx := float64(pos2.X - pos1.X)
	dy := float64(pos2.Y - pos1.Y)
	dz := float64(pos2.Z - pos1.Z)
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func (eas *EnemyAISystem) GetEnemyStates() []*proto.EnemyState {
	eas.mu.RLock()
	defer eas.mu.RUnlock()

	var states []*proto.EnemyState
	for _, enemy := range eas.enemies {
		status := proto.EnemyStatus_ALIVE
		if enemy.Health <= 0 {
			status = proto.EnemyStatus_DEAD
		}

		states = append(states, &proto.EnemyState{
			EnemyId:   enemy.EnemyID,
			EnemyType: enemy.EnemyType,
			Position:  enemy.Position,
			Health:    enemy.Health,
			MaxHealth: enemy.MaxHealth,
			Status:    status,
		})
	}

	return states
}

func (eas *EnemyAISystem) GetEnemyCount() int {
	eas.mu.RLock()
	defer eas.mu.RUnlock()

	return len(eas.enemies)
}

func (eas *EnemyAISystem) Clear() {
	eas.mu.Lock()
	defer eas.mu.Unlock()

	eas.enemies = make(map[string]*EnemyAI)
}
