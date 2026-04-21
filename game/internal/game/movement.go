package game

import (
	"fmt"
	"math"
	"time"

	"github.com/topfreegames/pitaya/v3/game/internal/proto"
)

const (
	MaxMovementSpeed = 10.0
	MovementCostPerUnit = 1.0
	MaxDistance = 1000.0
)

type MovementSystem struct {
	roomID string
}

func NewMovementSystem(roomID string) *MovementSystem {
	return &MovementSystem{
		roomID: roomID,
	}
}

func (ms *MovementSystem) ValidateMove(playerID string, currentPos, targetPos *proto.Position, timestamp int64) error {
	if currentPos == nil || targetPos == nil {
		return fmt.Errorf("invalid position")
	}

	distance := ms.CalculateDistance(currentPos, targetPos)
	if distance > MaxDistance {
		return fmt.Errorf("movement distance exceeds maximum")
	}

	if time.Now().Unix()-timestamp > 5 {
		return fmt.Errorf("movement request too old")
	}

	return nil
}

func (ms *MovementSystem) CalculateDistance(pos1, pos2 *proto.Position) float64 {
	dx := float64(pos2.X - pos1.X)
	dy := float64(pos2.Y - pos1.Y)
	dz := float64(pos2.Z - pos1.Z)
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func (ms *MovementSystem) CalculateMovementCost(distance float64) int32 {
	return int32(math.Ceil(distance * MovementCostPerUnit))
}

func (ms *MovementSystem) InterpolatePosition(currentPos, targetPos *proto.Position, progress float64) *proto.Position {
	return &proto.Position{
		X: currentPos.X + float32(float64(targetPos.X-currentPos.X)*progress),
		Y: currentPos.Y + float32(float64(targetPos.Y-currentPos.Y)*progress),
		Z: currentPos.Z + float32(float64(targetPos.Z-currentPos.Z)*progress),
	}
}

func (ms *MovementSystem) ClampPosition(pos *proto.Position, minX, maxX, minY, maxY, minZ, maxZ float64) *proto.Position {
	return &proto.Position{
		X: float32(math.Max(minX, math.Min(maxX, float64(pos.X)))),
		Y: float32(math.Max(minY, math.Min(maxY, float64(pos.Y)))),
		Z: float32(math.Max(minZ, math.Min(maxZ, float64(pos.Z)))),
	}
}
