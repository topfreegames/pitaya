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

package groups

import "testing"

func TestMemoryCreateDuplicatedGroup(t *testing.T) {
	testCreateDuplicatedGroup(memoryGroupService, t)
}

func TestMemoryCreateGroup(t *testing.T) {
	testCreateGroup(memoryGroupService, t)
}

func TestMemoryCreateGroupWithTTL(t *testing.T) {
	testCreateGroupWithTTL(memoryGroupService, t)
}

func TestMemoryGroupAddMember(t *testing.T) {
	testGroupAddMember(memoryGroupService, t)
}

func TestMemoryGroupAddDuplicatedMember(t *testing.T) {
	testGroupAddDuplicatedMember(memoryGroupService, t)
}

func TestMemoryGroupContainsMember(t *testing.T) {
	testGroupContainsMember(memoryGroupService, t)
}

func TestMemoryRemove(t *testing.T) {
	testRemove(memoryGroupService, t)
}

func TestMemoryDelete(t *testing.T) {
	testDelete(memoryGroupService, t)
}

func TestMemoryRemoveAll(t *testing.T) {
	testRemoveAll(memoryGroupService, t)
}

func TestMemoryCount(t *testing.T) {
	testCount(memoryGroupService, t)
}

func TestMemoryMembers(t *testing.T) {
	testMembers(memoryGroupService, t)
}
