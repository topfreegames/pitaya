import pitaya from 'k6/x/pitaya';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 10 },   // 30秒内增加到10个虚拟用户
    { duration: '1m', target: 50 },    // 1分钟内增加到50个虚拟用户
    { duration: '30s', target: 100 },   // 30秒内增加到100个虚拟用户
    { duration: '1m', target: 100 },   // 保持100个虚拟用户1分钟
    { duration: '30s', target: 50 },    // 30秒内减少到50个虚拟用户
    { duration: '30s', target: 0 },     // 30秒内减少到0个虚拟用户
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],  // 95%的请求在500ms内完成
    http_req_failed: ['rate<0.05'],    // 错误率低于5%
  },
};

const opts = {
  handshakeData: {
    sys: {
      clientVersion: "1.0.0",
      clientBuildNumber: "1",
      platform: "windows"
    },
    user: {
      fiu: `test_user_${__VU}_${__ITER}`,
      bundleId: "com.warincrising.game",
      deviceType: "pc",
      language: "en",
      osVersion: "10.0",
      region: "US",
      stack: "production"
    }
  },
  requestTimeoutMs: 5000,
  useTLS: false,
};

let pitayaClient = null;
let roomId = null;
let playerId = null;

export function setup() {
  console.log('Starting stress test for War Inc Rising');
}

export default async function () {
  playerId = `stress_test_${__VU}_${__ITER}`;

  if (!pitayaClient) {
    pitayaClient = new pitaya.Client(opts);
  }

  if (!pitayaClient.isConnected()) {
    pitayaClient.connect("127.0.0.1:3250");
    check(pitayaClient.isConnected(), { 'Connected to server': (r) => r === true });
  }

  sleep(1);

  const connectRes = await pitayaClient.request("connector.connector.Connect", {
    player_id: playerId,
    auth_token: "test_token",
    client_version: "1.0.0",
    device_id: `device_${__VU}_${__ITER}`
  });
  check(connectRes, { 'Connect successful': (r) => r !== undefined });

  sleep(0.5);

  if (!roomId) {
    const createRoomRes = await pitayaClient.request("room.room.CreateRoom", {
      room_type: "normal",
      max_players: 4
    });
    check(createRoomRes, { 'Create room successful': (r) => r !== undefined && r.room_id !== undefined });

    if (createRoomRes && createRoomRes.room_id) {
      roomId = createRoomRes.room_id;
    }
  }

  sleep(0.5);

  if (roomId) {
    const joinRoomRes = await pitayaClient.request("room.room.JoinRoom", {
      room_id: roomId
    });
    check(joinRoomRes, { 'Join room successful': (r) => r !== undefined });

    sleep(0.5);

    const startGameRes = await pitayaClient.request("room.room.StartGame", {
      room_id: roomId
    });
    check(startGameRes, { 'Start game successful': (r) => r !== undefined });

    sleep(0.5);

    for (let i = 0; i < 5; i++) {
      const moveRes = await pitayaClient.request("room.room.Move", {
        target_position: {
          x: Math.random() * 100,
          y: 0,
          z: Math.random() * 100
        },
        timestamp: Date.now()
      });
      check(moveRes, { `Move ${i} successful`: (r) => r !== undefined });

      sleep(0.2);

      const attackRes = await pitayaClient.request("room.room.Attack", {
        target_id: `enemy_${i}`,
        skill_id: 0,
        target_position: {
          x: Math.random() * 100,
          y: 0,
          z: Math.random() * 100
        },
        timestamp: Date.now()
      });
      check(attackRes, { `Attack ${i} successful`: (r) => r !== undefined });

      sleep(0.2);

      const skillRes = await pitayaClient.request("room.room.UseSkill", {
        skill_id: 1,
        target_position: {
          x: Math.random() * 100,
          y: 0,
          z: Math.random() * 100
        },
        timestamp: Date.now()
      });
      check(skillRes, { `Use skill ${i} successful`: (r) => r !== undefined });

      sleep(0.2);
    }

    const gameStateRes = await pitayaClient.request("room.room.GetRoomInfo", {
      room_id: roomId
    });
    check(gameStateRes, { 'Get game state successful': (r) => r !== undefined });

    sleep(0.5);

    const leaveRoomRes = await pitayaClient.request("room.room.LeaveRoom", {
      room_id: roomId
    });
    check(leaveRoomRes, { 'Leave room successful': (r) => r !== undefined });

    sleep(0.5);
  }

  pitayaClient.disconnect();
  pitayaClient = null;
  roomId = null;
}

export function teardown(data) {
  console.log('Stress test completed');
}
