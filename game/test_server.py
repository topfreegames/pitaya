#!/usr/bin/env python3
"""
简单的游戏服务器测试脚本
测试房间创建、加入、获取信息等基本功能
"""

import asyncio
import websockets
import json
import sys

class GameServerTester:
    def __init__(self, uri="ws://localhost:3250"):
        self.uri = uri
        self.room_id = None

    async def connect(self):
        """连接到游戏服务器"""
        try:
            self.ws = await websockets.connect(self.uri)
            print(f"✓ 已连接到 {self.uri}")
            return True
        except Exception as e:
            print(f"✗ 连接失败: {e}")
            return False

    async def send_message(self, route, data):
        """发送消息到服务器"""
        message = {
            "route": route,
            "data": data
        }
        try:
            await self.ws.send(json.dumps(message))
            response = await asyncio.wait_for(self.ws.recv(), timeout=5.0)
            return json.loads(response)
        except asyncio.TimeoutError:
            print(f"✗ 请求超时: {route}")
            return None
        except Exception as e:
            print(f"✗ 请求失败: {route}, 错误: {e}")
            return None

    async def test_create_room(self):
        """测试创建房间"""
        print("\n测试 1: 创建房间")
        response = await self.send_message("room.create", {
            "room_type": "normal",
            "max_players": 4,
            "difficulty": 1
        })
        
        if response and response.get("code") == 0:
            self.room_id = response.get("data", {}).get("room_id")
            print(f"✓ 房间创建成功")
            print(f"  房间ID: {self.room_id}")
            print(f"  最大玩家数: {response.get('data', {}).get('max_players')}")
            return True
        else:
            print(f"✗ 房间创建失败: {response}")
            return False

    async def test_join_room(self):
        """测试加入房间"""
        if not self.room_id:
            print("✗ 没有可用的房间ID")
            return False
            
        print("\n测试 2: 加入房间")
        response = await self.send_message("room.join", {
            "room_id": self.room_id
        })
        
        if response and response.get("code") == 0:
            print(f"✓ 成功加入房间")
            room_info = response.get("data", {}).get("room_info", {})
            print(f"  当前玩家数: {room_info.get('current_players')}")
            print(f"  房间状态: {room_info.get('status')}")
            return True
        else:
            print(f"✗ 加入房间失败: {response}")
            return False

    async def test_get_room_info(self):
        """测试获取房间信息"""
        if not self.room_id:
            print("✗ 没有可用的房间ID")
            return False
            
        print("\n测试 3: 获取房间信息")
        response = await self.send_message("room.get_info", {
            "room_id": self.room_id
        })
        
        if response and response.get("code") == 0:
            print(f"✓ 成功获取房间信息")
            room_info = response.get("data", {}).get("room_info", {})
            print(f"  房间ID: {room_info.get('room_id')}")
            print(f"  房间类型: {room_info.get('room_type')}")
            print(f"  当前玩家数: {room_info.get('current_players')}")
            print(f"  最大玩家数: {room_info.get('max_players')}")
            print(f"  房间状态: {room_info.get('status')}")
            print(f"  当前波次: {room_info.get('current_wave')}")
            print(f"  分数: {room_info.get('score')}")
            return True
        else:
            print(f"✗ 获取房间信息失败: {response}")
            return False

    async def test_list_rooms(self):
        """测试列出所有房间"""
        print("\n测试 4: 列出所有房间")
        response = await self.send_message("room.list", {
            "limit": 10,
            "offset": 0
        })
        
        if response and response.get("code") == 0:
            print(f"✓ 成功获取房间列表")
            rooms = response.get("data", {}).get("rooms", [])
            total_count = response.get("data", {}).get("total_count", 0)
            print(f"  总房间数: {total_count}")
            print(f"  返回房间数: {len(rooms)}")
            
            for i, room in enumerate(rooms[:3], 1):
                print(f"  房间 {i}:")
                print(f"    ID: {room.get('room_id')}")
                print(f"    玩家数: {room.get('current_players')}/{room.get('max_players')}")
                print(f"    状态: {room.get('status')}")
            return True
        else:
            print(f"✗ 获取房间列表失败: {response}")
            return False

    async def test_player_move(self):
        """测试玩家移动"""
        if not self.room_id:
            print("✗ 没有可用的房间ID")
            return False
            
        print("\n测试 5: 玩家移动")
        response = await self.send_message("room.move", {
            "target_position": {
                "x": 10.0,
                "y": 20.0,
                "z": 0.0
            }
        })
        
        if response and response.get("code") == 0:
            print(f"✓ 玩家移动成功")
            move_response = response.get("data", {})
            print(f"  新位置: {move_response.get('new_position')}")
            print(f"  移动消耗: {move_response.get('movement_cost')}")
            return True
        else:
            print(f"✗ 玩家移动失败: {response}")
            return False

    async def test_attack(self):
        """测试攻击"""
        if not self.room_id:
            print("✗ 没有可用的房间ID")
            return False
            
        print("\n测试 6: 攻击")
        response = await self.send_message("room.attack", {
            "target_id": "enemy_1",
            "skill_id": 1,
            "target_position": {
                "x": 15.0,
                "y": 25.0,
                "z": 0.0
            }
        })
        
        if response and response.get("code") == 0:
            print(f"✓ 攻击成功")
            attack_response = response.get("data", {})
            print(f"  造成伤害: {attack_response.get('damage_dealt')}")
            print(f"  目标被击杀: {attack_response.get('target_killed')}")
            print(f"  新分数: {attack_response.get('new_score')}")
            return True
        else:
            print(f"✗ 攻击失败: {response}")
            return False

    async def test_use_skill(self):
        """测试使用技能"""
        if not self.room_id:
            print("✗ 没有可用的房间ID")
            return False
            
        print("\n测试 7: 使用技能")
        response = await self.send_message("room.use_skill", {
            "skill_id": 1,
            "target_position": {
                "x": 12.0,
                "y": 22.0,
                "z": 0.0
            }
        })
        
        if response and response.get("code") == 0:
            print(f"✓ 技能使用成功")
            skill_response = response.get("data", {})
            print(f"  冷却时间: {skill_response.get('cooldown_remaining')}")
            print(f"  法力消耗: {skill_response.get('mana_cost')}")
            return True
        else:
            print(f"✗ 技能使用失败: {response}")
            return False

    async def run_all_tests(self):
        """运行所有测试"""
        print("=" * 60)
        print("游戏服务器功能测试")
        print("=" * 60)
        
        if not await self.connect():
            return False
        
        tests = [
            self.test_create_room,
            self.test_join_room,
            self.test_get_room_info,
            self.test_list_rooms,
            self.test_player_move,
            self.test_attack,
            self.test_use_skill,
        ]
        
        passed = 0
        failed = 0
        
        for test in tests:
            try:
                if await test():
                    passed += 1
                else:
                    failed += 1
            except Exception as e:
                print(f"✗ 测试异常: {e}")
                failed += 1
        
        print("\n" + "=" * 60)
        print(f"测试完成: {passed} 通过, {failed} 失败")
        print("=" * 60)
        
        try:
            await self.ws.close()
        except:
            pass
        
        return failed == 0

async def main():
    """主函数"""
    if len(sys.argv) > 1:
        uri = sys.argv[1]
    else:
        uri = "ws://localhost:3250"
    
    tester = GameServerTester(uri)
    success = await tester.run_all_tests()
    
    sys.exit(0 if success else 1)

if __name__ == "__main__":
    asyncio.run(main())
