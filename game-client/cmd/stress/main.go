package main

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"game-client/client"
)

type StressTestConfig struct {
	NumUsers        int
	TestDuration    time.Duration
	RequestsPerUser int
	ServerAddr      string
}

type StressTestStats struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalLatency    int64
	MinLatency      int64
	MaxLatency      int64
}

func main() {
	config := &StressTestConfig{
		NumUsers:        50,
		TestDuration:    2 * time.Minute,
		RequestsPerUser: 100,
		ServerAddr:      "127.0.0.1:3250",
	}

	fmt.Println("=== War Inc Rising Stress Test ===")
	fmt.Printf("Configuration:\n")
	fmt.Printf("  Users: %d\n", config.NumUsers)
	fmt.Printf("  Duration: %v\n", config.TestDuration)
	fmt.Printf("  Requests per user: %d\n", config.RequestsPerUser)
	fmt.Printf("  Server: %s\n\n", config.ServerAddr)

	stats := &StressTestStats{
		MinLatency: int64(^uint64(0) >> 1), // Max int64
	}

	var wg sync.WaitGroup
	startTime := time.Now()

	for i := 0; i < config.NumUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			runUserTest(userID, config, stats)
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	printStats(stats, duration, config)
}

func runUserTest(userID int, config *StressTestConfig, stats *StressTestStats) {
	playerID := fmt.Sprintf("stress_user_%d", userID)

	clientConfig := &client.Config{
		ServerAddr: config.ServerAddr,
		PlayerID:   playerID,
	}

	gameClient, err := client.NewGameClient(clientConfig)
	if err != nil {
		log.Printf("User %d: Failed to create client: %v", userID, err)
		return
	}
	defer gameClient.Close()

	err = gameClient.Connect()
	if err != nil {
		log.Printf("User %d: Failed to connect: %v", userID, err)
		return
	}

	err = gameClient.ConnectToGame()
	if err != nil {
		log.Printf("User %d: Failed to connect to game: %v", userID, err)
		return
	}

	createRoomResp, err := gameClient.CreateRoom("normal", 4)
	if err != nil {
		log.Printf("User %d: Failed to create room: %v", userID, err)
		return
	}

	roomID := createRoomResp.RoomId
	_ = roomID // Use roomID to avoid unused variable warning

	startGameResp, err := gameClient.StartGame()
	if err != nil {
		log.Printf("User %d: Failed to start game: %v", userID, err)
		// Continue with movement test even if game start fails
	} else if !startGameResp.Success {
		log.Printf("User %d: Game start failed", userID)
		// Continue with movement test even if game start fails
	}

	for i := 0; i < config.RequestsPerUser; i++ {
		start := time.Now()

		_, err = gameClient.MovePlayer(float32(i*10), 0.0, 5.0)
		latency := time.Since(start).Milliseconds()

		atomic.AddInt64(&stats.TotalRequests, 1)
		atomic.AddInt64(&stats.TotalLatency, latency)

		if err == nil {
			atomic.AddInt64(&stats.SuccessRequests, 1)
		} else {
			atomic.AddInt64(&stats.FailedRequests, 1)
		}

		// Update min/max latency
		for {
			currentMin := atomic.LoadInt64(&stats.MinLatency)
			if latency >= currentMin || atomic.CompareAndSwapInt64(&stats.MinLatency, currentMin, latency) {
				break
			}
		}

		for {
			currentMax := atomic.LoadInt64(&stats.MaxLatency)
			if latency <= currentMax || atomic.CompareAndSwapInt64(&stats.MaxLatency, currentMax, latency) {
				break
			}
		}

		time.Sleep(10 * time.Millisecond)
	}

	_, err = gameClient.LeaveRoom()
	if err != nil {
		log.Printf("User %d: Failed to leave room: %v", userID, err)
	}
}

func printStats(stats *StressTestStats, duration time.Duration, config *StressTestConfig) {
	fmt.Println("\n=== Stress Test Results ===")
	fmt.Printf("Test Duration: %v\n", duration)
	fmt.Printf("Total Users: %d\n", config.NumUsers)
	fmt.Printf("Total Requests: %d\n", stats.TotalRequests)
	fmt.Printf("Successful Requests: %d\n", stats.SuccessRequests)
	fmt.Printf("Failed Requests: %d\n", stats.FailedRequests)
	fmt.Printf("Success Rate: %.2f%%\n", float64(stats.SuccessRequests)/float64(stats.TotalRequests)*100)
	fmt.Printf("Requests per Second: %.2f\n", float64(stats.TotalRequests)/duration.Seconds())

	if stats.TotalRequests > 0 {
		avgLatency := float64(stats.TotalLatency) / float64(stats.TotalRequests)
		fmt.Printf("Average Latency: %.2f ms\n", avgLatency)
		fmt.Printf("Min Latency: %d ms\n", stats.MinLatency)
		fmt.Printf("Max Latency: %d ms\n", stats.MaxLatency)
	}

	fmt.Println("\n=== Test Completed ===")
}
