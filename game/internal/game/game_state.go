package game

import (
	"fmt"
	"sync"
	"time"

	"github.com/topfreegames/pitaya/v3/game/internal/proto"
)

type GameState struct {
	roomID          string
	currentWave     int32
	score           int32
	enemiesRemaining int32
	gameTime        time.Duration
	status          proto.RoomStatus
	startedAt       time.Time
	finishedAt      time.Time

	movementSystem *MovementSystem
	combatSystem   *CombatSystem
	skillSystem    *SkillSystem
	enemyAISystem  *EnemyAISystem
	waveManager    *WaveManager

	players        map[string]*PlayerCombatState
	playerSkills   map[string]map[int32]time.Time
	mu             sync.RWMutex
}

type PlayerCombatState struct {
	PlayerID      string
	Health        int32
	MaxHealth     int32
	Mana          int32
	MaxMana       int32
	AttackPower   int32
	DefensePower  int32
	Score         int32
	Position      *proto.Position
	LastAttack    time.Time
	LastMove      time.Time
}

func NewGameState(roomID string) *GameState {
	gs := &GameState{
		roomID:          roomID,
		currentWave:     0,
		score:           0,
		enemiesRemaining: 0,
		gameTime:        0,
		status:          proto.RoomStatus_WAITING,
		players:         make(map[string]*PlayerCombatState),
		playerSkills:    make(map[string]map[int32]time.Time),
	}

	gs.movementSystem = NewMovementSystem(roomID)
	gs.combatSystem = NewCombatSystem(roomID)
	gs.skillSystem = NewSkillSystem(roomID)
	gs.enemyAISystem = NewEnemyAISystem(roomID)
	gs.waveManager = NewWaveManager(roomID, gs.enemyAISystem)

	return gs
}

func (gs *GameState) AddPlayer(playerID string, position *proto.Position) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if _, exists := gs.players[playerID]; exists {
		return fmt.Errorf("player already exists: %s", playerID)
	}

	gs.players[playerID] = &PlayerCombatState{
		PlayerID:     playerID,
		Health:       100,
		MaxHealth:    100,
		Mana:         100,
		MaxMana:      100,
		AttackPower:  BaseAttackPower,
		DefensePower: BaseDefensePower,
		Score:        0,
		Position:     position,
		LastAttack:   time.Now(),
		LastMove:     time.Now(),
	}

	gs.playerSkills[playerID] = make(map[int32]time.Time)

	return nil
}

func (gs *GameState) RemovePlayer(playerID string) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	delete(gs.players, playerID)
	delete(gs.playerSkills, playerID)
}

func (gs *GameState) GetPlayer(playerID string) (*PlayerCombatState, error) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	player, exists := gs.players[playerID]
	if !exists {
		return nil, fmt.Errorf("player not found: %s", playerID)
	}

	return player, nil
}

func (gs *GameState) MovePlayer(playerID string, targetPos *proto.Position, timestamp int64) (*proto.MoveResponse, error) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	player, exists := gs.players[playerID]
	if !exists {
		return nil, fmt.Errorf("player not found: %s", playerID)
	}

	if err := gs.movementSystem.ValidateMove(playerID, player.Position, targetPos, timestamp); err != nil {
		return &proto.MoveResponse{
			Success: false,
		}, nil
	}

	distance := gs.movementSystem.CalculateDistance(player.Position, targetPos)
	cost := gs.movementSystem.CalculateMovementCost(distance)

	player.Position = targetPos
	player.LastMove = time.Now()

	return &proto.MoveResponse{
		Success:       true,
		NewPosition:   targetPos,
		MovementCost:  cost,
	}, nil
}

func (gs *GameState) Attack(playerID, targetID string, skillID int32, targetPos *proto.Position, timestamp int64) (*proto.AttackResponse, error) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	attacker, exists := gs.players[playerID]
	if !exists {
		return nil, fmt.Errorf("attacker not found: %s", playerID)
	}

	if err := gs.combatSystem.ValidateAttack(playerID, targetID, attacker.Position, targetPos, timestamp); err != nil {
		return &proto.AttackResponse{
			Success: false,
		}, nil
	}

	attackerStats := &CombatStats{
		PlayerID:     playerID,
		AttackPower:  attacker.AttackPower,
		DefensePower: attacker.DefensePower,
		CurrentHealth: attacker.Health,
		MaxHealth:    attacker.MaxHealth,
		Score:        attacker.Score,
		Position:     attacker.Position,
		LastAttack:   attacker.LastAttack,
	}

	damage := gs.combatSystem.CalculateDamage(attackerStats, attackerStats)
	attacker.LastAttack = time.Now()

	if skillID > 0 {
		damage = gs.skillSystem.CalculateSkillDamage(skillID, damage)
	}

	return &proto.AttackResponse{
		Success:      true,
		DamageDealt:  damage,
		TargetKilled: false,
		NewScore:     attacker.Score,
	}, nil
}

func (gs *GameState) UseSkill(playerID string, skillID int32, targetPos *proto.Position, timestamp int64) (*proto.UseSkillResponse, error) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	player, exists := gs.players[playerID]
	if !exists {
		return nil, fmt.Errorf("player not found: %s", playerID)
	}

	skillCooldowns, exists := gs.playerSkills[playerID]
	if !exists {
		return nil, fmt.Errorf("player skills not found: %s", playerID)
	}

	if err := gs.skillSystem.ValidateUseSkill(playerID, skillID, player.Mana, skillCooldowns); err != nil {
		return &proto.UseSkillResponse{
			Success: false,
		}, nil
	}

	newMana, cooldown, err := gs.skillSystem.UseSkill(skillID, player.Mana, skillCooldowns)
	if err != nil {
		return &proto.UseSkillResponse{
			Success: false,
		}, nil
	}

	player.Mana = newMana

	return &proto.UseSkillResponse{
		Success:           true,
		CooldownRemaining: int32(cooldown.Seconds()),
		ManaCost:          0,
	}, nil
}

func (gs *GameState) StartGame() error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if gs.status != proto.RoomStatus_WAITING {
		return fmt.Errorf("game is not in waiting status")
	}

	if len(gs.players) == 0 {
		return fmt.Errorf("no players in game")
	}

	gs.status = proto.RoomStatus_PLAYING
	gs.startedAt = time.Now()

	return nil
}

func (gs *GameState) Update(deltaTime time.Duration) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if gs.status != proto.RoomStatus_PLAYING {
		return
	}

	gs.gameTime += deltaTime

	players := make(map[string]*proto.Position)
	for playerID, player := range gs.players {
		players[playerID] = player.Position
	}

	gs.enemyAISystem.UpdateAllEnemies(players, deltaTime)

	gs.enemiesRemaining = int32(gs.enemyAISystem.GetEnemyCount())

	if gs.enemiesRemaining == 0 && gs.waveManager.IsWaveActive() {
		gs.waveManager.CompleteWave()
		gs.currentWave = gs.waveManager.GetCurrentWave()
		gs.score += gs.currentWave * 100
	}
}

func (gs *GameState) StartNextWave() error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if gs.status != proto.RoomStatus_PLAYING {
		return fmt.Errorf("game is not in playing status")
	}

	nextWave := gs.waveManager.GetNextWave()
	if nextWave == 0 {
		return fmt.Errorf("no more waves")
	}

	if err := gs.waveManager.StartWave(nextWave); err != nil {
		return err
	}

	gs.currentWave = nextWave

	return nil
}

func (gs *GameState) GetStateUpdate() *proto.GameStateUpdate {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	players := make([]*proto.PlayerState, 0, len(gs.players))
	for playerID, player := range gs.players {
		skills := make([]*proto.ActiveSkill, 0)
		for skillID := range gs.playerSkills[playerID] {
			cooldown := gs.skillSystem.GetCooldownRemaining(skillID, gs.playerSkills[playerID])
			skills = append(skills, &proto.ActiveSkill{
				SkillId:           skillID,
				CooldownRemaining: int32(cooldown.Seconds()),
				IsActive:          cooldown == 0,
			})
		}

		players = append(players, &proto.PlayerState{
			PlayerId: playerID,
			Health:   player.Health,
			MaxHealth: player.MaxHealth,
			Position: player.Position,
			Score:    player.Score,
			Skills:   skills,
		})
	}

	return &proto.GameStateUpdate{
		RoomId:           gs.roomID,
		CurrentWave:      gs.currentWave,
		EnemiesRemaining: gs.enemiesRemaining,
		Score:            gs.score,
		Enemies:          gs.enemyAISystem.GetEnemyStates(),
		Players:          players,
		ServerTime:       time.Now().Unix(),
	}
}

func (gs *GameState) GetScore() int32 {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	return gs.score
}

func (gs *GameState) GetCurrentWave() int32 {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	return gs.currentWave
}

func (gs *GameState) GetStatus() proto.RoomStatus {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	return gs.status
}

func (gs *GameState) GetGameTime() time.Duration {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	return gs.gameTime
}

func (gs *GameState) IsGameOver() bool {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	if gs.status == proto.RoomStatus_FINISHED {
		return true
	}

	if gs.status == proto.RoomStatus_PLAYING {
		if gs.waveManager.GetNextWave() == 0 && gs.enemiesRemaining == 0 {
			return true
		}
	}

	return false
}

func (gs *GameState) EndGame() {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	gs.status = proto.RoomStatus_FINISHED
	gs.finishedAt = time.Now()
}

func (gs *GameState) Reset() {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	gs.currentWave = 0
	gs.score = 0
	gs.enemiesRemaining = 0
	gs.gameTime = 0
	gs.status = proto.RoomStatus_WAITING
	gs.startedAt = time.Time{}
	gs.finishedAt = time.Time{}

	gs.players = make(map[string]*PlayerCombatState)
	gs.playerSkills = make(map[string]map[int32]time.Time)

	gs.waveManager.Reset()
}
