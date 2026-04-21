package main

import (
	"fmt"
	"log"
	"time"

	"game-client/client"
)

func main() {
	fmt.Println("=== War Inc Rising 链路测试 ===")

	testCases := []struct {
		name string
		fn   func() error
	}{
		{"测试1: 基础连接测试", testBasicConnection},
		{"测试2: 创建房间测试", testCreateRoom},
		{"测试3: 加入房间测试", testJoinRoom},
		{"测试4: 游戏流程测试", testGameFlow},
		{"测试5: 完整游戏流程测试", testCompleteGameFlow},
	}

	for _, tc := range testCases {
		fmt.Printf("\n--- %s ---\n", tc.name)
		err := tc.fn()
		if err != nil {
			log.Printf("❌ 测试失败: %v\n", err)
		} else {
			log.Printf("✅ 测试通过\n")
		}
		time.Sleep(1 * time.Second)
	}

	fmt.Println("\n=== 链路测试完成 ===")
}

func testBasicConnection() error {
	config := &client.Config{
		ServerAddr: "127.0.0.1:3250",
		PlayerID:   "test_player_1",
	}

	gameClient, err := client.NewGameClient(config)
	if err != nil {
		return fmt.Errorf("创建客户端失败: %w", err)
	}
	defer gameClient.Close()

	err = gameClient.Connect()
	if err != nil {
		return fmt.Errorf("连接服务器失败: %w", err)
	}
	fmt.Println("✓ 成功连接到服务器")

	err = gameClient.ConnectToGame()
	if err != nil {
		return fmt.Errorf("连接游戏失败: %w", err)
	}
	fmt.Println("✓ 成功连接到游戏")

	err = gameClient.SendHeartbeat()
	if err != nil {
		return fmt.Errorf("发送心跳失败: %w", err)
	}
	fmt.Println("✓ 心跳发送成功")

	time.Sleep(2 * time.Second)

	return nil
}

func testCreateRoom() error {
	config := &client.Config{
		ServerAddr: "127.0.0.1:3250",
		PlayerID:   "test_player_2",
	}

	gameClient, err := client.NewGameClient(config)
	if err != nil {
		return fmt.Errorf("创建客户端失败: %w", err)
	}
	defer gameClient.Close()

	err = gameClient.Connect()
	if err != nil {
		return fmt.Errorf("连接服务器失败: %w", err)
	}

	err = gameClient.ConnectToGame()
	if err != nil {
		return fmt.Errorf("连接游戏失败: %w", err)
	}

	resp, err := gameClient.CreateRoom("normal", 4)
	if err != nil {
		return fmt.Errorf("创建房间失败: %w", err)
	}
	fmt.Printf("✓ 成功创建房间: %s\n", resp.RoomId)

	roomInfo, err := gameClient.GetRoomInfo()
	if err != nil {
		return fmt.Errorf("get room info failed: %w", err)
	}
	fmt.Printf("✓ Room info: players=%d, status=%v\n", roomInfo.RoomInfo.CurrentPlayers, roomInfo.RoomInfo.Status)

	return nil
}

func testJoinRoom() error {
	config1 := &client.Config{
		ServerAddr: "127.0.0.1:3250",
		PlayerID:   "test_player_3",
	}

	gameClient1, err := client.NewGameClient(config1)
	if err != nil {
		return fmt.Errorf("创建客户端1失败: %w", err)
	}
	defer gameClient1.Close()

	err = gameClient1.Connect()
	if err != nil {
		return fmt.Errorf("客户端1连接失败: %w", err)
	}

	err = gameClient1.ConnectToGame()
	if err != nil {
		return fmt.Errorf("客户端1连接游戏失败: %w", err)
	}

	createResp, err := gameClient1.CreateRoom("normal", 4)
	if err != nil {
		return fmt.Errorf("创建房间失败: %w", err)
	}
	fmt.Printf("✓ 玩家1创建房间: %s\n", createResp.RoomId)

	config2 := &client.Config{
		ServerAddr: "127.0.0.1:3250",
		PlayerID:   "test_player_4",
	}

	gameClient2, err := client.NewGameClient(config2)
	if err != nil {
		return fmt.Errorf("创建客户端2失败: %w", err)
	}
	defer gameClient2.Close()

	err = gameClient2.Connect()
	if err != nil {
		return fmt.Errorf("客户端2连接失败: %w", err)
	}

	err = gameClient2.ConnectToGame()
	if err != nil {
		return fmt.Errorf("客户端2连接游戏失败: %w", err)
	}

	joinResp, err := gameClient2.JoinRoom(createResp.RoomId)
	if err != nil {
		return fmt.Errorf("join room failed: %w", err)
	}
	fmt.Printf("✓ Player2 joined room: %s\n", joinResp.RoomInfo.RoomId)

	roomInfo, err := gameClient1.GetRoomInfo()
	if err != nil {
		return fmt.Errorf("get room info failed: %w", err)
	}
	fmt.Printf("✓ Room current players: %d\n", roomInfo.RoomInfo.CurrentPlayers)

	return nil
}

func testGameFlow() error {
	config := &client.Config{
		ServerAddr: "127.0.0.1:3250",
		PlayerID:   "test_player_5",
	}

	gameClient, err := client.NewGameClient(config)
	if err != nil {
		return fmt.Errorf("创建客户端失败: %w", err)
	}
	defer gameClient.Close()

	err = gameClient.Connect()
	if err != nil {
		return fmt.Errorf("连接服务器失败: %w", err)
	}

	err = gameClient.ConnectToGame()
	if err != nil {
		return fmt.Errorf("连接游戏失败: %w", err)
	}

	_, err = gameClient.CreateRoom("normal", 4)
	if err != nil {
		return fmt.Errorf("创建房间失败: %w", err)
	}
	fmt.Println("✓ 创建房间成功")

	startResp, err := gameClient.StartGame()
	if err != nil {
		return fmt.Errorf("start game failed: %w", err)
	}
	fmt.Printf("✓ Game started: %v\n", startResp.Success)

	moveResp, err := gameClient.MovePlayer(10.0, 0.0, 5.0)
	if err != nil {
		return fmt.Errorf("move player failed: %w", err)
	}
	fmt.Printf("✓ Player moved successfully: %v\n", moveResp.Success)

	attackResp, err := gameClient.Attack("enemy_1")
	if err != nil {
		return fmt.Errorf("attack failed: %w", err)
	}
	fmt.Printf("✓ Attack successful: damage=%d\n", attackResp.DamageDealt)

	skillResp, err := gameClient.UseSkill(1)
	if err != nil {
		return fmt.Errorf("use skill failed: %w", err)
	}
	fmt.Printf("✓ Skill used successfully: cooldown=%d\n", skillResp.CooldownRemaining)

	gameState, err := gameClient.GetGameState()
	if err != nil {
		return fmt.Errorf("get game state failed: %w", err)
	}
	fmt.Printf("✓ Game state: wave=%d, score=%d\n", gameState.CurrentWave, gameState.Score)

	return nil
}

func testCompleteGameFlow() error {
	config1 := &client.Config{
		ServerAddr: "127.0.0.1:3250",
		PlayerID:   "test_player_6",
	}

	gameClient1, err := client.NewGameClient(config1)
	if err != nil {
		return fmt.Errorf("创建客户端1失败: %w", err)
	}
	defer gameClient1.Close()

	err = gameClient1.Connect()
	if err != nil {
		return fmt.Errorf("客户端1连接失败: %w", err)
	}

	err = gameClient1.ConnectToGame()
	if err != nil {
		return fmt.Errorf("客户端1连接游戏失败: %w", err)
	}

	createResp, err := gameClient1.CreateRoom("normal", 4)
	if err != nil {
		return fmt.Errorf("创建房间失败: %w", err)
	}
	fmt.Printf("✓ 玩家1创建房间: %s\n", createResp.RoomId)

	config2 := &client.Config{
		ServerAddr: "127.0.0.1:3250",
		PlayerID:   "test_player_7",
	}

	gameClient2, err := client.NewGameClient(config2)
	if err != nil {
		return fmt.Errorf("创建客户端2失败: %w", err)
	}
	defer gameClient2.Close()

	err = gameClient2.Connect()
	if err != nil {
		return fmt.Errorf("客户端2连接失败: %w", err)
	}

	err = gameClient2.ConnectToGame()
	if err != nil {
		return fmt.Errorf("客户端2连接游戏失败: %w", err)
	}

	_, err = gameClient2.JoinRoom(createResp.RoomId)
	if err != nil {
		return fmt.Errorf("玩家2加入房间失败: %w", err)
	}
	fmt.Println("✓ 玩家2加入房间")

	_, err = gameClient1.StartGame()
	if err != nil {
		return fmt.Errorf("开始游戏失败: %w", err)
	}
	fmt.Println("✓ 游戏开始")

	for i := 0; i < 5; i++ {
		_, err = gameClient1.MovePlayer(float32(i*10), 0.0, 5.0)
		if err != nil {
			return fmt.Errorf("玩家1移动失败: %w", err)
		}

		_, err = gameClient2.MovePlayer(float32(i*10+5), 0.0, 5.0)
		if err != nil {
			return fmt.Errorf("玩家2移动失败: %w", err)
		}

		_, err = gameClient1.Attack("enemy_1")
		if err != nil {
			return fmt.Errorf("玩家1攻击失败: %w", err)
		}

		_, err = gameClient2.UseSkill(1)
		if err != nil {
			return fmt.Errorf("player2 use skill failed: %w", err)
		}

		time.Sleep(500 * time.Millisecond)
	}
	fmt.Println("✓ 游戏进行中...")

	gameState, err := gameClient1.GetGameState()
	if err != nil {
		return fmt.Errorf("获取游戏状态失败: %w", err)
	}
	fmt.Printf("✓ 最终游戏状态: 波次=%d, 分数=%d, 敌人剩余=%d\n",
		gameState.CurrentWave, gameState.Score, gameState.EnemiesRemaining)

	ranking, err := gameClient1.GetRanking(10)
	if err != nil {
		return fmt.Errorf("get ranking failed: %w", err)
	}
	fmt.Printf("✓ Ranking: %d rooms\n", len(ranking.Rooms))

	playerRanking, err := gameClient1.GetPlayerRanking()
	if err != nil {
		return fmt.Errorf("get player ranking failed: %w", err)
	}
	fmt.Printf("✓ Player ranking: %d rooms\n", len(playerRanking.Rooms))

	_, err = gameClient1.LeaveRoom()
	if err != nil {
		return fmt.Errorf("玩家1离开房间失败: %w", err)
	}
	fmt.Println("✓ 玩家1离开房间")

	_, err = gameClient2.LeaveRoom()
	if err != nil {
		return fmt.Errorf("玩家2离开房间失败: %w", err)
	}
	fmt.Println("✓ 玩家2离开房间")

	return nil
}
