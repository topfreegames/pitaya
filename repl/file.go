// Copyright (c) Wildlife Studios. All Rights Reserved.
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

package repl

import (
	"bufio"
	"errors"
	"log"
	"os"
	"strings"

	"github.com/topfreegames/pitaya/v3/pkg/logger"
)

func executeFromFile(fileName string) {
	var err error
	defer func() {
		if err != nil {
			logger.Log.Errorf("error: %s", err.Error())
		}
	}()

	var file *os.File
	file, err = os.Open(fileName)
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		command := scanner.Text()
		err = executeCommand(command)
		if err != nil {
			return
		}
	}

	err = scanner.Err()
	if err != nil {
		return
	}
}

func executeCommand(command string) error {
	parts := strings.Split(command, " ")

	switch parts[0] {
	case "connect":
		return connect(parts[1], func(data []byte) {
			log.Printf("sv-> %s\n", string(data))
			wait.Done()
		})

	case "sethandshake":
		return setHandshake(parts[1:])

	case "request":
		wait.Add(1)
		if err := request(parts[1:]); err != nil {
			return err
		}
		wait.Wait()
		return nil

	case "notify":
		return notify(parts[1:])

	case "push":
		return push(parts[1:])

	case "disconnect":
		disconnect()
		return nil

	default:
		return errors.New("command not found")
	}
}
