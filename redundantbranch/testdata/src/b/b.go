// Copyright 2019 Axel Wagner
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package b

import "fmt"

func TestUselessBreak() {
	var (
		x  int
		ch chan int
	)

	switch x {
	case 1:
		break // want `break does not affect control flow`
	case 2:
		if 1 == 2 {
			break
		}
		fmt.Println("foo")
	}

	for {
		select {
		case _, ok := <-ch:
			if !ok {
				break // want `break does not affect control flow`
			}
		}
	}

EvLoop:
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				break EvLoop
			}
		}
	}

	switch x {
	case 1:
		break // want `break does not affect control flow`
	case 2:
		if 1 == 2 {
			break
		}
		fmt.Println("baz")
	}
}
