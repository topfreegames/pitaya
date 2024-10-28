import pitaya from 'k6/x/pitaya';
import { check, sleep } from 'k6';

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

export let options = {
    stages: [
        { target: 10, duration: '5s' },
    ],
    thresholds: {
        pitaya_client_request_duration_ms: ['p(95)<200'], // 95% of requests should be below 200ms
    },
}

const pitayaClient = new pitaya.Client(opts)

export default async () => {
    if (!pitayaClient.isConnected()) {
        pitayaClient.connectToQUIC("localhost:3250")
    }

    check(pitayaClient.isConnected(), { 'pitaya client is connected': (r) => r === true })

    var res = await pitayaClient.request("requestor.room.entry")
    check(res.result, { 'contains an result field': (r) => r !== undefined })
    check(res.result, { 'result is ok': (r) => r === "ok" })

    res = await pitayaClient.request("connector.setsessiondata", { data: { "testKey": "testVal" } })
    check(res.Msg, { 'res is success': (r) => r === "success" })

    res = await pitayaClient.request("connector.getsessiondata")
    check(res.Data, { 'res contains set data': (r) => r.testKey === "testVal" })

    res = await pitayaClient.request("requestor.room.join")
    res = await pitayaClient.consumePush("onMembers", 1000)
    check(res.members, { 'res contains a member group': (m) => m !== undefined })

    sleep(2)

    pitayaClient.disconnect()
}

export function teardown() {
}