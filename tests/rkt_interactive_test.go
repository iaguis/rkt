// Copyright 2015 The rkt Authors
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

package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/ThomasRooney/gexpect"
)

var interactiveTests = []struct {
	testName     string
	aciBuildArgs []string
	rktArgs      string
	say          string
	expect       string
}{
	{
		`Check tty without interactive`,
		[]string{"--exec=/inspect --check-tty"},
		`--debug --insecure-skip-verify run rkt-inspect-interactive.aci`,
		``,
		`stdin is not a terminal`,
	},
	{
		`Check tty without interactive (with parameter)`,
		[]string{"--exec=/inspect"},
		`--debug --insecure-skip-verify run rkt-inspect-interactive.aci -- --check-tty`,
		``,
		`stdin is not a terminal`,
	},
	{
		`Check tty with interactive`,
		[]string{"--exec=/inspect --check-tty"},
		`--debug --insecure-skip-verify run --interactive rkt-inspect-interactive.aci`,
		``,
		`stdin is a terminal`,
	},
	{
		`Check tty with interactive (with parameter)`,
		[]string{"--exec=/inspect"},
		`--debug --insecure-skip-verify run --interactive rkt-inspect-interactive.aci -- --check-tty`,
		``,
		`stdin is a terminal`,
	},
	{
		`Reading from stdin`,
		[]string{"--exec=/inspect --read-stdin"},
		`--debug --insecure-skip-verify run --interactive rkt-inspect-interactive.aci`,
		`Saluton`,
		`Received text: Saluton`,
	},
	{
		`Reading from stdin (with parameter)`,
		[]string{"--exec=/inspect"},
		`--debug --insecure-skip-verify run --interactive rkt-inspect-interactive.aci -- --read-stdin`,
		`Saluton`,
		`Received text: Saluton`,
	},
}

func TestInteractive(t *testing.T) {
	ctx := newRktRunCtx()
	defer ctx.cleanup()

	for i, tt := range interactiveTests {
		t.Logf("Running test #%v: %v", i, tt.testName)

		aciFileName := "rkt-inspect-interactive.aci"
		patchTestACI(aciFileName, tt.aciBuildArgs...)
		defer os.Remove(aciFileName)

		rktCmd := fmt.Sprintf("%s %s", ctx.cmd(), tt.rktArgs)
		t.Logf("Command: %v", rktCmd)
		child, err := gexpect.Spawn(rktCmd)
		if err != nil {
			t.Fatalf("Cannot exec rkt #%v: %v", i, err)
		}
		if tt.say != "" {
			err = child.ExpectTimeout("Enter text:", time.Minute)
			if err != nil {
				t.Fatalf("Waited for the prompt but not found #%v: %v\n\nrkt output:\n%s", i, err, string(child.Collect()))
			}

			err = child.SendLine(tt.say)
			if err != nil {
				t.Fatalf("Failed to send %q on the prompt #%v: %v", tt.say, i, err)
			}
		}

		err = child.ExpectTimeout(tt.expect, time.Minute)
		if err != nil {
			t.Fatalf("Expected %q but not found #%v: %v\n\nrkt output:\n%s", tt.expect, i, err, string(child.Collect()))
		}

		err = child.Wait()
		if err != nil {
			t.Fatalf("rkt didn't terminate correctly: %v", err)
		}
	}
}
