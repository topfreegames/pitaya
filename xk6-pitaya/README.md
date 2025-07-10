# xk6-pitaya

`xk6-pitaya` is a [k6](https://go.k6.io/k6) extension that provides a [Pitaya](https://github.com/topfreegames/pitaya) client implementation.

# Usage

## Building the k6 binary

### From the repository (local build)
```shell
# Clone the repository and navigate to it
git clone https://github.com/topfreegames/pitaya.git
cd pitaya/xk6-pitaya

# Build with the latest commit
xk6 build \
  --with github.com/topfreegames/pitaya/xk6-pitaya/v2@v2.11.16 \
  --replace github.com/topfreegames/pitaya/v2=github.com/topfreegames/pitaya/v2@v2.11.16 \
  --output ./k6-pitaya
```

### From anywhere (remote build)
```shell
# Build from any directory using a specific version
xk6 build \
  --with github.com/topfreegames/pitaya/xk6-pitaya/v2@v2.11.16 \
  --replace github.com/topfreegames/pitaya/v2=github.com/topfreegames/pitaya/v2@v2.11.16 \
  --output ./k6-pitaya
```

## Building the k6 docker image

```shell
# Build with a specific version
docker build --build-arg pitaya_revision=v2.11.16 -t xk6-pitaya .
```

## Requirements

- **Go 1.23+** (required for k6 v1.1.0+ compatibility)
- **xk6** (latest version recommended)
- **Pitaya v2.11.16+** (for the client implementation)

## Compatibility

This extension is compatible with:
- **k6 v1.1.0+** (uses sobek JavaScript engine)
- **Pitaya v2.x** (uses the v2 client API)
- **Go 1.23+** (required for the latest k6)

## Example usage

```javascript
import pitaya from 'k6/x/pitaya';
import { check } from 'k6';

export const options = {
  vus: 10,
  duration: '10s',
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

  var res = await pitayaClient.request("room.room.entry")
  check(res.result, { 'contains an result field': (r) => r !== undefined })
  check(res.result, { 'result is ok': (r) => r === "ok" })

  var res = await pitayaClient.request("room.room.setsessiondata", { data: {"testKey": "testVal"} })
  check(res, { 'res is success': (r) => String.fromCharCode.apply(null,r) === "success"} )
  var res = await pitayaClient.request("room.room.getsessiondata")
  check(res.Data, { 'res contains set data': (r) => r.testKey === "testVal"} )
  res = await pitayaClient.request("room.room.join")
  check(res.result, { 'result from join is successful': (r) => r === "success"} )
  res = await pitayaClient.consumePush("onMembers", 500)
  check(res.Members, { 'res contains a member group': (m) => m !== undefined } )
  res = await pitayaClient.request("room.room.leave")
  check(res, { 'result from leave is successful': (r) => String.fromCharCode.apply(null,r) === "success"})

 pitayaClient.disconnect()
}

export function teardown() {
}
```

## Running the scenario 1 example

```shell

# spin up pitaya dependencies
make ensure-testing-deps

# run pitaya server, backend and frontend
make run-cluster-example-backend
make run-cluster-example-frontend

# run k6 scenario
./k6-pitaya run ./examples/scenario1.js
```

# Metrics

This extension will add the following metrics to the k6 output:

- `pitaya_client_request_duration_ms`: Histogram of request durations in milliseconds
    - `success`: If the request was successful or not
    - `route`: The route of the request
- `pitaya_client_request_timeout_count`: Counter of timedout requests
    - `route`: The route of the request

# Protobuf client support

This extension does not support pitaya running with protobuf serialization. For loadtesting your server with this, use the json serializer.

```go
builder.Serializer = json.NewSerializer()
```

Or just don't set it, since json is the default serializer.

# Migration from older versions

If you're upgrading from an older version of xk6-pitaya:

1. **Update your build command** to use the new module path (`/v2`)
2. **Use a compatible k6 version** (v1.1.0+ recommended)
3. **Update your Go version** to 1.23+ if needed

The extension now uses the **sobek JavaScript engine** (instead of goja) for better compatibility with the latest k6 versions.

# Additional Documentation

All k6 documentation also applies to this extension. See https://k6.io/docs/ for more information.

# Running distributed tests

It is possible to run distributed tests using k6 and this extension. To do so you can refer to the [k6 documentation](https://k6.io/docs/testing-guides/running-distributed-tests/) and use the binary generated from this repo as the k6 binary. There's a prebuilt docker image available at [tfgco/xk6-pitaya](https://hub.docker.com/r/tfgco/xk6-pitaya) that you can use as well.
