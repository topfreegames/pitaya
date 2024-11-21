import pitaya from 'k6/x/pitaya';
import { check, sleep } from 'k6';
import { Trend } from 'k6/metrics';

const opts = {
    handshakeData: {
        sys: {
            clientBuildNumber: "1",
            clientVersion: "9.0.0",
            platform: "ios"
        },
        user: {
            bundleId: "com.cronus.monkeys.inhouse",
            deviceType: "MacBook Pro",
            fiu: "d58360c4-b5b0-4bb2-8dae-4a30d6c8d1b1",
            language: "en",
            osVersion: "1.0",
            region: "US"
        }
    },
    requestTimeoutMs: 1000,
    useTLS: true,
}

let requestTimeEntryRoomTrend = new Trend('request_time_entry_room');
let requestTimeSessionDataTrend = new Trend('request_time_session_data');
let requestTimeJoinRoomTrend = new Trend('request_time_join_room');

export let options = {
    stages: [
        { target: 5, duration: '5s' },
    ],
    thresholds: {
        pitaya_client_request_duration_ms: ['p(95)<200'], // 95% of requests should be below 200ms
    },
}

const pitayaClient = new pitaya.Client(opts)

async function connectToServer() {
    await pitayaClient.connect("localhost:3250")

    check(pitayaClient.isConnected(), { 'pitaya client is connected': (r) => r === true })
}

async function entryRoom() {
    let startTime = Date.now();
    var res = await pitayaClient.request("requestor.room.entry")
    let duration = Date.now() - startTime;
    requestTimeEntryRoomTrend.add(duration);
    if (res.msg != "session is already bound to an uid") {
        check(res.result, { 'contains an result field': (r) => r !== undefined })
        check(res.result, { 'result is ok': (r) => r === "ok" })
    }
}

async function setAndRetrieveSessionData(sessionData) {
    let startTime = Date.now();
    var res = await pitayaClient.request("connector.setsessiondata", sessionData)
    let duration = Date.now() - startTime;
    requestTimeSessionDataTrend.add(duration);
    check(res.Msg, { 'res is success': (r) => r === "success" })

    res = await pitayaClient.request("connector.getsessiondata")
    check(res.Data, { 'res contains set data': (r) => r.testKey === sessionData.data.testKey })
}

async function joinRoom() {
    let startTime = Date.now();
    var res = await pitayaClient.request("requestor.room.join")
    let duration = Date.now() - startTime;
    requestTimeJoinRoomTrend.add(duration);
    if (res.msg === "onclose callbacks are not allowed on backend servers") {
        res = await pitayaClient.consumePush("onMembers", 1000)
        check(res.members, { 'res contains a member group': (m) => m !== undefined })
    }
}

export default async () => {

    await connectToServer();

    sleep(1);
    await entryRoom();

    sleep(1);
    await setAndRetrieveSessionData({ data: { "testKey": "testVal" } });

    await pitayaClient.disconnect()
}

export function teardown() {
}