package game

import (
	"fmt"
	"sync"
	"time"

	"github.com/topfreegames/pitaya/v3/game/internal/proto"
)

type WaveConfig struct {
	WaveNumber      int32
	EnemyCount      int32
	EnemyType       string
	EnemyBehavior   EnemyBehavior
	SpawnInterval   time.Duration
	EnemyHealth     int32
	EnemyAttackPower int32
	EnemySpeed      float64
	EnemyRange      float64
}

type WaveManager struct {
	roomID        string
	currentWave   int32
	waves         map[int32]*WaveConfig
	enemyAISystem *EnemyAISystem
	mu            sync.RWMutex
	isActive      bool
	waveStartTime time.Time
}

func NewWaveManager(roomID string, enemyAISystem *EnemyAISystem) *WaveManager {
	wm := &WaveManager{
		roomID:        roomID,
		currentWave:   0,
		waves:         make(map[int32]*WaveConfig),
		enemyAISystem: enemyAISystem,
		isActive:      false,
	}

	wm.initializeWaves()
	return wm
}

func (wm *WaveManager) initializeWaves() {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	wm.waves[1] = &WaveConfig{
		WaveNumber:       1,
		EnemyCount:       5,
		EnemyType:        "basic",
		EnemyBehavior:    BehaviorAggressive,
		SpawnInterval:    2 * time.Second,
		EnemyHealth:      50,
		EnemyAttackPower: 10,
		EnemySpeed:       5.0,
		EnemyRange:       30.0,
	}

	wm.waves[2] = &WaveConfig{
		WaveNumber:       2,
		EnemyCount:       8,
		EnemyType:        "fast",
		EnemyBehavior:    BehaviorAggressive,
		SpawnInterval:    1 * time.Second,
		EnemyHealth:      40,
		EnemyAttackPower: 15,
		EnemySpeed:       8.0,
		EnemyRange:       25.0,
	}

	wm.waves[3] = &WaveConfig{
		WaveNumber:       3,
		EnemyCount:       10,
		EnemyType:        "tank",
		EnemyBehavior:    BehaviorDefensive,
		SpawnInterval:    3 * time.Second,
		EnemyHealth:      100,
		EnemyAttackPower: 20,
		EnemySpeed:       3.0,
		EnemyRange:       35.0,
	}

	wm.waves[4] = &WaveConfig{
		WaveNumber:       4,
		EnemyCount:       15,
		EnemyType:        "mixed",
		EnemyBehavior:    BehaviorPatrol,
		SpawnInterval:    1 * time.Second,
		EnemyHealth:      60,
		EnemyAttackPower: 18,
		EnemySpeed:       6.0,
		EnemyRange:       30.0,
	}

	wm.waves[5] = &WaveConfig{
		WaveNumber:       5,
		EnemyCount:       20,
		EnemyType:        "boss",
		EnemyBehavior:    BehaviorAggressive,
		SpawnInterval:    2 * time.Second,
		EnemyHealth:      200,
		EnemyAttackPower: 30,
		EnemySpeed:       4.0,
		EnemyRange:       40.0,
	}
}

func (wm *WaveManager) StartWave(waveNumber int32) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if wm.isActive {
		return fmt.Errorf("wave already in progress")
	}

	config, exists := wm.waves[waveNumber]
	if !exists {
		return fmt.Errorf("wave not found: %d", waveNumber)
	}

	wm.currentWave = waveNumber
	wm.isActive = true
	wm.waveStartTime = time.Now()

	go wm.spawnWave(config)

	return nil
}

func (wm *WaveManager) spawnWave(config *WaveConfig) {
	for i := int32(0); i < config.EnemyCount; i++ {
		enemy := &EnemyAI{
			EnemyID:     fmt.Sprintf("enemy_%d_%d", config.WaveNumber, i),
			EnemyType:   config.EnemyType,
			Behavior:    config.EnemyBehavior,
			Position:    &proto.Position{X: float32(i * 10), Y: 0, Z: 0},
			Health:      config.EnemyHealth,
			MaxHealth:   config.EnemyHealth,
			AttackPower: config.EnemyAttackPower,
			Speed:       config.EnemySpeed,
			Range:       config.EnemyRange,
			LastAction:  time.Now(),
		}

		wm.enemyAISystem.AddEnemy(enemy)

		if i < config.EnemyCount-1 {
			time.Sleep(config.SpawnInterval)
		}
	}
}

func (wm *WaveManager) CompleteWave() {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	wm.isActive = false
}

func (wm *WaveManager) GetCurrentWave() int32 {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	return wm.currentWave
}

func (wm *WaveManager) GetWaveConfig(waveNumber int32) (*WaveConfig, error) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	config, exists := wm.waves[waveNumber]
	if !exists {
		return nil, fmt.Errorf("wave not found: %d", waveNumber)
	}

	return config, nil
}

func (wm *WaveManager) IsWaveActive() bool {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	return wm.isActive
}

func (wm *WaveManager) GetWaveProgress() (int32, int32) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	if !wm.isActive {
		return 0, 0
	}

	config, exists := wm.waves[wm.currentWave]
	if !exists {
		return 0, 0
	}

	enemyCount := wm.enemyAISystem.GetEnemyCount()
	remaining := config.EnemyCount - int32(enemyCount)

	return remaining, config.EnemyCount
}

func (wm *WaveManager) GetNextWave() int32 {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	nextWave := wm.currentWave + 1
	if _, exists := wm.waves[nextWave]; !exists {
		return 0
	}

	return nextWave
}

func (wm *WaveManager) GetTotalWaves() int32 {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	return int32(len(wm.waves))
}

func (wm *WaveManager) GetWaveTimeRemaining() time.Duration {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	if !wm.isActive {
		return 0
	}

	config, exists := wm.waves[wm.currentWave]
	if !exists {
		return 0
	}

	expectedDuration := time.Duration(config.EnemyCount) * config.SpawnInterval
	elapsed := time.Since(wm.waveStartTime)

	remaining := expectedDuration - elapsed
	if remaining < 0 {
		remaining = 0
	}

	return remaining
}

func (wm *WaveManager) Reset() {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	wm.currentWave = 0
	wm.isActive = false
	wm.enemyAISystem.Clear()
}

func (wm *WaveManager) GetWaveReward(waveNumber int32) []*proto.Reward {
	_, err := wm.GetWaveConfig(waveNumber)
	if err != nil {
		return nil
	}

	rewards := []*proto.Reward{
		{Type: "coins", Amount: waveNumber * 100},
		{Type: "exp", Amount: waveNumber * 50},
	}

	if waveNumber%5 == 0 {
		rewards = append(rewards, &proto.Reward{Type: "gems", Amount: 10})
	}

	return rewards
}
