Pitaya CLI
==========

A REPL cli client made in go for pitaya.

## Installing

```
go install github.com/topfreegames/pitaya/v3
```

## Usage

For cli flags, run `pitaya-cli --help`

```
$ pitaya

Pitaya command line utilities

Usage:
  pitaya [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  repl        starts pitaya repl tool

Flags:
  -h, --help   help for pitaya

Use "pitaya [command] --help" for more information about a command.

$ pitaya repl

Pitaya REPL Client
>>> help

Commands:
  clear             clear the screen
  connect           connects to pitaya
  disconnect        disconnects from pitaya server
  exit              exit the program
  help              display help
  notify            makes a notify to pitaya server
  push              insert information of push return
  request           makes a request to pitaya server
  sethandshake      sets a handshake parameter
```

### Protobuf
For connecting to a server that uses protobuf as serializer the server must implement two routes:
- Docs: responsible for returning all handlers and the protos used on input and
  output;
- Descriptors: The list of protos descriptions, this will be used by the CLI to
  encode/decode the messages.

To implement those routes you can use some functions provided by pitaya, here is
a short example of both routes:
```go
import (
	// ...

	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/protos"
)

// Docs handler
func (c *MyHandler) Docs(ctx context.Context) (*protos.Doc, error) {
	d, err := pitaya.Documentation(true)
	if err != nil {
		return nil, fmt.Errorf("failed to generate documentation for pitaya routes: %w", err)
	}

	doc, err := json.Marshal(d)
	if err != nil {
		return nil, fmt.Errorf("failed to encode documentation JSON: %w", err)
	}

	return &protos.Doc{Doc: string(doc)}, nil
}

// Descriptors route
func (c *MyHandler) Descriptors(ctx context.Context, names *protos.ProtoNames) (*protos.ProtoDescriptors, error) {
	descriptors := make([][]byte, len(names.Name))

	for i, protoName := range names.Name {
		desc, err := pitaya.Descriptor(protoName)
		if err != nil {
			return nil, fmt.Errorf("failed to get descriptor for '%s': %w", protoName, err)
		}

		descriptors[i] = desc
	}

	return &protos.ProtoDescriptors{Desc: descriptors}, nil
}
```

When initilizing the CLI, you have to provide the docs route as the following:
```
pitaya-cli -docs connector.docsHandler.docs
```

NOTE: The descriptors handler is automatically discovered by the client.
It must only follow the signature mentioned earlier.

A full example of running pitaya-cli with protobuf:
```
pitaya-cli -docs connector.docsHandler.docs
>>> push connector.playerHandler.matchfound protos.FindMatchPush
>>> connect localhost:30124
>>> request connector.playerHandler.create
>>> request connector.playerHandler.findmatch {"RoomType":"xxxx"}
```

### Set handshake parameters

You can edit handshake parameters before connecting to the server.

You may pass the full handshake json:
```
Pitaya REPL Client
>>> sethandshake {"sys":{"clientVersion":"1.0.6", "clientBuildNumber":"999","platform":"ios"}}
```

Or edit one of three specific parameters:
```
Pitaya REPL Client
>>> sethandshake platform ios
>>> sethandshake buildNumber 999
>>> sethandshake version 1.0.6
```

### Read commands from file

It's possible to add a list of sequential requests into a file and pitaya-cli will execute them in order.

For example: commands.txt

```
connect localhost:3250
request connector.playerHandler.create
request connector.playerHandler.findmatch {"RoomType":"xxxx"}
```

Then run: `pitaya-cli --filename commands.txt`
