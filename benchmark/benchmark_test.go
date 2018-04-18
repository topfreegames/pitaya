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

package benchmark

import (
	"os"
	"testing"

	"github.com/topfreegames/pitaya/client"
)

var c *client.Client

func TestMain(m *testing.M) {
	c = client.New(false)
	err := c.ConnectTo("localhost:32222")
	if err != nil {
		panic(err)
	}
	exit := m.Run()
	os.Exit(exit)
}

func BenchmarkCreateManyClients(b *testing.B) {
	// TODO start server
	// b.ResetTimer()

	for i := 0; i < b.N; i++ {
		g := client.New(false)
		err := g.ConnectTo("localhost:32222")
		if err != nil {
			b.Logf("failed to connect")
			b.FailNow()
		}
	}

}

func BenchmarkCreateManyClientsSendRequest(b *testing.B) {
	// TODO start server
	// b.ResetTimer()

	for i := 0; i < b.N; i++ {
		g := client.New(false)
		err := g.ConnectTo("localhost:32222")
		if err != nil {
			b.Logf("failed to connect")
			b.FailNow()
		}
		err = g.SendRequest("connector.testsvc.testrequestreceivereturnsraw", []byte("ola"))
		if err != nil {
			b.Logf("failed to send request to server")
			b.FailNow()
		}
		<-g.IncomingMsgChan
	}

}

func BenchmarkFrontHandlerWithSessionAndRawReturnsRaw(b *testing.B) {
	// TODO start server
	// b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := c.SendRequest("connector.testsvc.testrequestreceivereturnsraw", []byte("ola"))
		if err != nil {
			b.Logf("failed to send request to server")
			b.FailNow()
		}
		<-c.IncomingMsgChan
	}
}

func BenchmarkFrontHandlerWithSessionAndPtrReturnsPtr(b *testing.B) {
	// TODO start server
	// b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := c.SendRequest("connector.testsvc.testrequestreturnsptr", []byte(`{"msg":"bench single"}`))
		if err != nil {
			b.Logf("failed to send request to server")
			b.FailNow()
		}
		<-c.IncomingMsgChan
	}
}

func BenchmarkFrontHandlerWithSessionAndPtrReturnsPtrParallel(b *testing.B) {
	// TODO start server
	// b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := c.SendRequest("connector.testsvc.testrequestreturnsptr", []byte(`{"msg":"bench parall"}`))
			if err != nil {
				b.Logf("failed to send request to server")
				b.FailNow()
			}
			<-c.IncomingMsgChan
			//helpers.ShouldEventuallyReceive(b, c.IncomingMsgChan)
		}
	})
}

func BenchmarkFrontHandlerWithSessionOnlyReturnsPtr(b *testing.B) {
	// TODO start server
	// b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := c.SendRequest("connector.testsvc.testrequestonlysessionreturnsptr", []byte{})
		if err != nil {
			b.Logf("failed to send request to server")
			b.FailNow()
		}
		<-c.IncomingMsgChan
	}
}

func BenchmarkFrontHandlerWithSessionOnlyReturnsPtrParallel(b *testing.B) {
	// TODO start server
	// b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := c.SendRequest("connector.testsvc.testrequestonlysessionreturnsptr", []byte{})
			if err != nil {
				b.Logf("failed to send request to server")
				b.FailNow()
			}
			<-c.IncomingMsgChan
		}
	})
}

func BenchmarkBackHandlerWithSessionOnlyReturnsPtr(b *testing.B) {
	// TODO start server
	// b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := c.SendRequest("game.testsvc.testrequestonlysessionreturnsptr", []byte{})
		if err != nil {
			b.Logf("failed to send request to server")
			b.FailNow()
		}
		<-c.IncomingMsgChan
	}
}

func BenchmarkBackHandlerWithSessionOnlyReturnsPtrParallel(b *testing.B) {
	// TODO start server
	// b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := c.SendRequest("game.testsvc.testrequestonlysessionreturnsptr", []byte{})
			if err != nil {
				b.Logf("failed to send request to server")
				b.FailNow()
			}
			<-c.IncomingMsgChan
		}
	})
}
