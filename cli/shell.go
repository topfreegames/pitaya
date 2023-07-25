// Copyright (c) TFG Co. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"github.com/abiosoft/ishell/v2"
)

func repl() {
	shell := ishell.New()
	configure(shell)

	shell.Println("Pitaya REPL Client")

	registerConnect(shell)
	registerDisconnect(shell)
	registerRequest(shell)
	registerNotify(shell)
	registerPush(shell)
	registerSetHandshake(shell)

	pushInfo = make(map[string]string)

	shell.Run()
}

func registerConnect(shell *ishell.Shell) {
	shell.AddCmd(&ishell.Cmd{
		Name: "connect",
		Help: "connects to pitaya",
		Func: func(c *ishell.Context) {
			var addr string
			if len(c.Args) == 0 {
				c.Print("address: ")
				addr = c.ReadLine()
			} else {
				addr = c.Args[0]
			}

			if err := connect(c, addr, func(data []byte) {
				c.Printf("sv->%s\n", string(data))
			}); err != nil {
				c.Err(err)
			}
		},
	})
}

func registerPush(shell *ishell.Shell) {
	shell.AddCmd(&ishell.Cmd{
		Name: "push",
		Help: "insert information of push return",
		Func: func(c *ishell.Context) {
			err := push(c, c.RawArgs[1:])
			if err != nil {
				c.Err(err)
			}
		},
	})
}

func registerRequest(shell *ishell.Shell) {
	shell.AddCmd(&ishell.Cmd{
		Name: "request",
		Help: "makes a request to pitaya server",
		Func: func(c *ishell.Context) {
			err := request(c, c.RawArgs[1:])
			if err != nil {
				c.Err(err)
			}
		},
	})
}

func registerNotify(shell *ishell.Shell) {
	shell.AddCmd(&ishell.Cmd{
		Name: "notify",
		Help: "makes a notify to pitaya server",
		Func: func(c *ishell.Context) {
			err := notify(c, c.RawArgs[1:])
			if err != nil {
				c.Err(err)
			}
		},
	})
}

func registerDisconnect(shell *ishell.Shell) {
	shell.AddCmd(&ishell.Cmd{
		Name: "disconnect",
		Help: "disconnects from pitaya server",
		Func: func(c *ishell.Context) {
			disconnect()
		},
	})
}

func registerSetHandshake(shell *ishell.Shell) {
	shell.AddCmd(&ishell.Cmd{
		Name: "sethandshake",
		Help: "sets a handshake parameter",
		Func: func(c *ishell.Context) {
			err := setHandshake(c, c.RawArgs[1:])
			if err != nil {
				c.Err(err)
			}
		},
	})
}
