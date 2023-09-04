# xk6-pitaya

`xk6-pitaya` is a [k6](https://go.k6.io/k6) extension that provides a [Pitaya](https://github.com/topfreegames/pitaya) client implementation.

# Usage

## Building the k6 binary

```shell
xk6 build --with github.com/topfreegames/xk6-pitaya=.
```

## Example usage

```javascript
import pitaya from 'k6/x/pitaya
';
import { check } from 'k6';

export const options = {
  vus: 1,
  duration: '3s',
}

const opts = {
  handshakeData: {
    sys: {
      clientVersion: "1.0.1",
      clientBuildNumber: "1",
      platform: "android"
    },
    user: {
      fiu: "c0a78b27-dd34-4e0d-bff7-36168fce0df5",
      bundleId: "com.game.test",
      deviceType: "ios",
      language: "en",
      osVersion: "12.0",
      region: "US",
      stack: "green-stack"
    }
  },
  requestTimeoutMs: 1000,
  logLevel: "info",
  serializer: "json",
}

const pitayaClient = new pitaya.Client(opts)

export default async () => {
  if (!pitayaClient.isConnected()) {
    pitayaClient.connect("localhost:3250")
  }

  check(pitayaClient.isConnected(), { 'pitaya client is connected': (r) => r === true })

  var res = await pitayaClient.request("metagame.authenticationHandler.createAccount")
  check(res.code, { 'code is 200': (c) => c === "200" })
  check(res.id, { 'contains an id field': (i) => i !== undefined })
  check(res.securityToken, { 'contains a securityToken field': (i) => i !== undefined })

  res = await pitayaClient.request("metagame.authenticationHandler.authenticate", { id: res.id, securityToken: res.securityToken })
  check(res.code, { 'code is 200': (c) => c === "200" })
  check(res.additionalPayload, { 'contains an additionalPayload field': (i) => i !== undefined })

  pitayaClient.notify("metagame.exampleCustomHandler.testNotifyHandler")

  res = await pitayaClient.consumePush("testHandler.testPush", 500)
  check(res.msg, { 'msg is included in the push': (m) => m === m !== undefined })
  check(res.msg, { 'msg includes Hello': (m) => String(m).includes("Hello") })
}

export function teardown() {
  pitayaClient.disconnect()
}
```
