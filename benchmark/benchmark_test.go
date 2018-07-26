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
	"fmt"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/topfreegames/pitaya/client"
)

var clients []*client.Client

func getClients(n, port int) []*client.Client {
	c := make([]*client.Client, n)
	for i := 0; i < n; i++ {
		c[i] = client.New(logrus.FatalLevel)
		err := c[i].ConnectTo(fmt.Sprintf("%s:%d", "localhost", port))
		if err != nil {
			panic(err)
		}
	}
	return c

}

func TestMain(m *testing.M) {
	exit := m.Run()
	os.Exit(exit)
}

func BenchmarkCreateManyClients(b *testing.B) {

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g := client.New(logrus.FatalLevel)
		err := g.ConnectTo(fmt.Sprintf("%s:%d", "localhost", 32222))
		defer g.Disconnect()
		if err != nil {
			b.Logf("failed to connect")
			b.FailNow()
		}
	}

}

func BenchmarkFrontHandlerWithSessionAndRawReturnsRaw(b *testing.B) {
	clients := getClients(1, 32222)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := clients[0].SendRequest("connector.testsvc.testrequestreceivereturnsraw", []byte("ola"))
		if err != nil {
			b.Logf("failed to send request to server")
			b.FailNow()
		}
		<-clients[0].IncomingMsgChan
	}
}

func BenchmarkFrontHandlerWithSessionAndPtrReturnsPtr(b *testing.B) {
	clients := getClients(1, 32222)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := clients[0].SendRequest("connector.testsvc.testrequestreturnsptr", []byte(`{"msg":"bench single"}`))
		if err != nil {
			b.Logf("failed to send request to server")
			b.FailNow()
		}
		<-clients[0].IncomingMsgChan
	}
}

func BenchmarkFrontHandlerWithSessionAndPtrReturnsPtrManyClientsParallel(b *testing.B) {
	numClients := 1000
	clients := getClients(numClients, 32222)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := clients[b.N%numClients].SendRequest("connector.testsvc.testrequestreturnsptr", []byte(`{"msg":"bench parall"}`))
			if err != nil {
				b.Logf("failed to send request to server")
				b.FailNow()
			}
			<-clients[b.N%numClients].IncomingMsgChan
		}
	})
}

func BenchmarkFrontHandlerWithSessionAndPtrReturnsPtrParallel(b *testing.B) {
	clients := getClients(1, 32222)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := clients[0].SendRequest("connector.testsvc.testrequestreturnsptr", []byte(`{"msg":"b"}`))
			if err != nil {
				b.Logf("failed to send request to server")
				b.FailNow()
			}
			<-clients[0].IncomingMsgChan
		}
	})
}

func BenchmarkFrontHandlerWithSessionOnlyReturnsPtr(b *testing.B) {
	clients := getClients(1, 32222)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := clients[0].SendRequest("connector.testsvc.testrequestonlysessionreturnsptr", []byte{})
		if err != nil {
			b.Logf("failed to send request to server")
			b.FailNow()
		}
		<-clients[0].IncomingMsgChan
	}
}

func BenchmarkFrontHandlerWithSessionOnlyReturnsPtrParallel(b *testing.B) {
	clients := getClients(1, 32222)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := clients[0].SendRequest("connector.testsvc.testrequestonlysessionreturnsptr", []byte{})
			if err != nil {
				b.Logf("failed to send request to server")
				b.FailNow()
			}
			<-clients[0].IncomingMsgChan
		}
	})
}

func BenchmarkBackHandlerWithSessionOnlyReturnsPtr(b *testing.B) {
	clients := getClients(1, 32222)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := clients[0].SendRequest("game.testsvc.testrequestonlysessionreturnsptr", []byte{})
		if err != nil {
			b.Logf("failed to send request to server")
			b.FailNow()
		}

		<-clients[0].IncomingMsgChan
	}
}

func BenchmarkBackHandlerWithSessionOnlyReturnsPtrParallel(b *testing.B) {
	clients := getClients(1, 32222)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := clients[0].SendRequest("game.testsvc.testrequestonlysessionreturnsptr", []byte{})
			if err != nil {
				b.Logf("failed to send request to server")
				b.FailNow()
			}
			<-clients[0].IncomingMsgChan
		}
	})
}

func BenchmarkBackHandlerWithSessionOnlyReturnsPtrParallelMultipleClients(b *testing.B) {
	numClients := 100
	clients := getClients(numClients, 32222)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := clients[b.N%numClients].SendRequest("game.testsvc.testrequestonlysessionreturnsptr", []byte{})
			if err != nil {
				b.Logf("failed to send request to server")
				b.FailNow()
			}
			<-clients[b.N%numClients].IncomingMsgChan
		}
	})
}
