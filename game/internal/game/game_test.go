package game

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/v3/game/internal/proto"
)

func TestMovementSystem_ValidateMove(t *testing.T) {
	ms := NewMovementSystem("test-room")

	currentPos := &proto.Position{X: 0, Y: 0, Z: 0}
	targetPos := &proto.Position{X: 10, Y: 10, Z: 0}
	timestamp := time.Now().Unix()

	err := ms.ValidateMove("player1", currentPos, targetPos, timestamp)
	assert.NoError(t, err)

	farTargetPos := &proto.Position{X: 2000, Y: 2000, Z: 0}
	err = ms.ValidateMove("player1", currentPos, farTargetPos, timestamp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum")
}

func TestMovementSystem_CalculateDistance(t *testing.T) {
	ms := NewMovementSystem("test-room")

	pos1 := &proto.Position{X: 0, Y: 0, Z: 0}
	pos2 := &proto.Position{X: 3, Y: 4, Z: 0}

	distance := ms.CalculateDistance(pos1, pos2)
	assert.InDelta(t, 5.0, distance, 0.01)
}

func TestMovementSystem_CalculateMovementCost(t *testing.T) {
	ms := NewMovementSystem("test-room")

	cost := ms.CalculateMovementCost(10.5)
	assert.Equal(t, int32(11), cost)
}

func TestCombatSystem_CalculateDamage(t *testing.T) {
	cs := NewCombatSystem("test-room")

	attacker := &CombatStats{
		AttackPower:  20,
		DefensePower: 5,
		CurrentHealth: 100,
		MaxHealth:    100,
	}

	defender := &CombatStats{
		AttackPower:  10,
		DefensePower: 10,
		CurrentHealth: 100,
		MaxHealth:    100,
	}

	damage := cs.CalculateDamage(attacker, defender)
	assert.Greater(t, damage, int32(0))
	assert.LessOrEqual(t, damage, int32(20))
}

func TestCombatSystem_ValidateAttack(t *testing.T) {
	cs := NewCombatSystem("test-room")

	attackerPos := &proto.Position{X: 0, Y: 0, Z: 0}
	targetPos := &proto.Position{X: 10, Y: 10, Z: 0}
	timestamp := time.Now().Unix()

	err := cs.ValidateAttack("player1", "player2", attackerPos, targetPos, timestamp)
	assert.NoError(t, err)

	farTargetPos := &proto.Position{X: 100, Y: 100, Z: 0}
	err = cs.ValidateAttack("player1", "player2", attackerPos, farTargetPos, timestamp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestCombatSystem_ProcessAttack(t *testing.T) {
	cs := NewCombatSystem("test-room")

	attacker := &CombatStats{
		PlayerID:     "player1",
		AttackPower:  20,
		DefensePower: 5,
		CurrentHealth: 100,
		MaxHealth:    100,
		Score:        0,
	}

	defender := &CombatStats{
		PlayerID:     "player2",
		AttackPower:  10,
		DefensePower: 10,
		CurrentHealth: 50,
		MaxHealth:    100,
		Score:        0,
	}

	result := cs.ProcessAttack(attacker, defender)
	assert.True(t, result.Success)
	assert.Greater(t, result.DamageDealt, int32(0))
	assert.LessOrEqual(t, defender.CurrentHealth, int32(50))
}

func TestSkillSystem_GetSkill(t *testing.T) {
	ss := NewSkillSystem("test-room")

	skill, err := ss.GetSkill(1)
	assert.NoError(t, err)
	assert.Equal(t, int32(1), skill.ID)
	assert.Equal(t, "Fireball", skill.Name)

	_, err = ss.GetSkill(999)
	assert.Error(t, err)
}

func TestSkillSystem_ValidateUseSkill(t *testing.T) {
	ss := NewSkillSystem("test-room")

	lastUsed := make(map[int32]time.Time)

	err := ss.ValidateUseSkill("player1", 1, 100, lastUsed)
	assert.NoError(t, err)

	err = ss.ValidateUseSkill("player1", 1, 10, lastUsed)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not enough mana")

	lastUsed[1] = time.Now()
	err = ss.ValidateUseSkill("player1", 1, 100, lastUsed)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "on cooldown")
}

func TestSkillSystem_UseSkill(t *testing.T) {
	ss := NewSkillSystem("test-room")

	lastUsed := make(map[int32]time.Time)

	newMana, cooldown, err := ss.UseSkill(1, 100, lastUsed)
	assert.NoError(t, err)
	assert.Equal(t, int32(80), newMana)
	assert.Greater(t, cooldown, time.Duration(0))
}

func TestSkillSystem_GetCooldownRemaining(t *testing.T) {
	ss := NewSkillSystem("test-room")

	lastUsed := make(map[int32]time.Time)

	cooldown := ss.GetCooldownRemaining(1, lastUsed)
	assert.Equal(t, time.Duration(0), cooldown)

	lastUsed[1] = time.Now()
	cooldown = ss.GetCooldownRemaining(1, lastUsed)
	assert.Greater(t, cooldown, time.Duration(0))
}

func TestEnemyAISystem_AddEnemy(t *testing.T) {
	eas := NewEnemyAISystem("test-room")

	enemy := &EnemyAI{
		EnemyID:   "enemy1",
		EnemyType: "basic",
		Behavior:  BehaviorAggressive,
		Position:  &proto.Position{X: 0, Y: 0, Z: 0},
		Health:    100,
		MaxHealth: 100,
	}

	err := eas.AddEnemy(enemy)
	assert.NoError(t, err)

	err = eas.AddEnemy(enemy)
	assert.Error(t, err)
}

func TestEnemyAISystem_RemoveEnemy(t *testing.T) {
	eas := NewEnemyAISystem("test-room")

	enemy := &EnemyAI{
		EnemyID:   "enemy1",
		EnemyType: "basic",
		Behavior:  BehaviorAggressive,
		Position:  &proto.Position{X: 0, Y: 0, Z: 0},
		Health:    100,
		MaxHealth: 100,
	}

	eas.AddEnemy(enemy)
	eas.RemoveEnemy("enemy1")

	_, err := eas.GetEnemy("enemy1")
	assert.Error(t, err)
}

func TestEnemyAISystem_UpdateAllEnemies(t *testing.T) {
	eas := NewEnemyAISystem("test-room")

	enemy := &EnemyAI{
		EnemyID:   "enemy1",
		EnemyType: "basic",
		Behavior:  BehaviorAggressive,
		Position:  &proto.Position{X: 0, Y: 0, Z: 0},
		Health:    100,
		MaxHealth: 100,
		Speed:     5.0,
		Range:     30.0,
	}

	eas.AddEnemy(enemy)

	players := map[string]*proto.Position{
		"player1": {X: 10, Y: 10, Z: 0},
	}

	eas.UpdateAllEnemies(players, 100*time.Millisecond)

	updatedEnemy, _ := eas.GetEnemy("enemy1")
	assert.NotEqual(t, float64(0), updatedEnemy.Position.X)
}

func TestWaveManager_StartWave(t *testing.T) {
	eas := NewEnemyAISystem("test-room")
	wm := NewWaveManager("test-room", eas)

	err := wm.StartWave(1)
	assert.NoError(t, err)
	assert.True(t, wm.IsWaveActive())
	assert.Equal(t, int32(1), wm.GetCurrentWave())

	err = wm.StartWave(2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already in progress")
}

func TestWaveManager_CompleteWave(t *testing.T) {
	eas := NewEnemyAISystem("test-room")
	wm := NewWaveManager("test-room", eas)

	wm.StartWave(1)
	wm.CompleteWave()

	assert.False(t, wm.IsWaveActive())
}

func TestWaveManager_GetWaveConfig(t *testing.T) {
	eas := NewEnemyAISystem("test-room")
	wm := NewWaveManager("test-room", eas)

	config, err := wm.GetWaveConfig(1)
	assert.NoError(t, err)
	assert.Equal(t, int32(1), config.WaveNumber)
	assert.Equal(t, int32(5), config.EnemyCount)

	_, err = wm.GetWaveConfig(999)
	assert.Error(t, err)
}

func TestWaveManager_GetWaveProgress(t *testing.T) {
	eas := NewEnemyAISystem("test-room")
	wm := NewWaveManager("test-room", eas)

	remaining, total := wm.GetWaveProgress()
	assert.Equal(t, int32(0), remaining)
	assert.Equal(t, int32(0), total)

	wm.StartWave(1)
	time.Sleep(100 * time.Millisecond)

	remaining, total = wm.GetWaveProgress()
	assert.Greater(t, total, int32(0))
}

func TestGameState_AddPlayer(t *testing.T) {
	gs := NewGameState("test-room")

	position := &proto.Position{X: 0, Y: 0, Z: 0}
	err := gs.AddPlayer("player1", position)
	assert.NoError(t, err)

	err = gs.AddPlayer("player1", position)
	assert.Error(t, err)
}

func TestGameState_MovePlayer(t *testing.T) {
	gs := NewGameState("test-room")

	position := &proto.Position{X: 0, Y: 0, Z: 0}
	gs.AddPlayer("player1", position)

	targetPos := &proto.Position{X: 10, Y: 10, Z: 0}
	response, err := gs.MovePlayer("player1", targetPos, time.Now().Unix())
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, targetPos, response.NewPosition)
}

func TestGameState_Attack(t *testing.T) {
	gs := NewGameState("test-room")

	position := &proto.Position{X: 0, Y: 0, Z: 0}
	gs.AddPlayer("player1", position)
	gs.AddPlayer("player2", &proto.Position{X: 10, Y: 10, Z: 0})

	response, err := gs.Attack("player1", "player2", 0, &proto.Position{X: 10, Y: 10, Z: 0}, time.Now().Unix())
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Greater(t, response.DamageDealt, int32(0))
}

func TestGameState_UseSkill(t *testing.T) {
	gs := NewGameState("test-room")

	position := &proto.Position{X: 0, Y: 0, Z: 0}
	gs.AddPlayer("player1", position)

	response, err := gs.UseSkill("player1", 1, &proto.Position{X: 10, Y: 10, Z: 0}, time.Now().Unix())
	assert.NoError(t, err)
	assert.True(t, response.Success)
}

func TestGameState_StartGame(t *testing.T) {
	gs := NewGameState("test-room")

	position := &proto.Position{X: 0, Y: 0, Z: 0}
	gs.AddPlayer("player1", position)

	err := gs.StartGame()
	assert.NoError(t, err)
	assert.Equal(t, proto.RoomStatus_PLAYING, gs.GetStatus())
}

func TestGameState_Update(t *testing.T) {
	gs := NewGameState("test-room")

	position := &proto.Position{X: 0, Y: 0, Z: 0}
	gs.AddPlayer("player1", position)
	gs.StartGame()

	gs.Update(100 * time.Millisecond)
	assert.Greater(t, gs.GetGameTime(), time.Duration(0))
}

func TestGameState_StartNextWave(t *testing.T) {
	gs := NewGameState("test-room")

	position := &proto.Position{X: 0, Y: 0, Z: 0}
	gs.AddPlayer("player1", position)
	gs.StartGame()

	err := gs.StartNextWave()
	assert.NoError(t, err)
	assert.Equal(t, int32(1), gs.GetCurrentWave())
}

func TestGameState_GetStateUpdate(t *testing.T) {
	gs := NewGameState("test-room")

	position := &proto.Position{X: 0, Y: 0, Z: 0}
	gs.AddPlayer("player1", position)

	update := gs.GetStateUpdate()
	assert.NotNil(t, update)
	assert.Equal(t, "test-room", update.RoomId)
	assert.Len(t, update.Players, 1)
}

func TestGameState_IsGameOver(t *testing.T) {
	gs := NewGameState("test-room")

	position := &proto.Position{X: 0, Y: 0, Z: 0}
	gs.AddPlayer("player1", position)

	assert.False(t, gs.IsGameOver())

	gs.StartGame()
	assert.False(t, gs.IsGameOver())

	gs.EndGame()
	assert.True(t, gs.IsGameOver())
}

func TestGameState_Reset(t *testing.T) {
	gs := NewGameState("test-room")

	position := &proto.Position{X: 0, Y: 0, Z: 0}
	gs.AddPlayer("player1", position)
	gs.StartGame()

	gs.Reset()

	assert.Equal(t, proto.RoomStatus_WAITING, gs.GetStatus())
	assert.Equal(t, int32(0), gs.GetScore())
	assert.Equal(t, int32(0), gs.GetCurrentWave())
}
