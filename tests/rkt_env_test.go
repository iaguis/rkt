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
	"os"
	"strings"
	"testing"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/ThomasRooney/gexpect"
)

var envTests = []struct {
	runCmd    string
	runExpect string
	sleepCmd  string
	enterCmd  string
}{
	{
		`^RKT_BIN^ --debug --insecure-skip-verify run ./rkt-inspect-print-var-from-manifest.aci`,
		"VAR_FROM_MANIFEST=manifest",
		`^RKT_BIN^ --debug --insecure-skip-verify run --interactive ./rkt-inspect-sleep.aci`,
		`/bin/sh -c "^RKT_BIN^ --debug enter $(^RKT_BIN^ list --full|grep running|awk '{print $1}') /inspect --print-env=VAR_FROM_MANIFEST"`,
	},
	{
		`^RKT_BIN^ --debug --insecure-skip-verify run --set-env=VAR_OTHER=setenv ./rkt-inspect-print-var-other.aci`,
		"VAR_OTHER=setenv",
		`^RKT_BIN^ --debug --insecure-skip-verify run --interactive --set-env=VAR_OTHER=setenv ./rkt-inspect-sleep.aci`,
		`/bin/sh -c "^RKT_BIN^ --debug enter $(^RKT_BIN^ list --full|grep running|awk '{print $1}') /inspect --print-env=VAR_OTHER"`,
	},
	{
		`^RKT_BIN^ --debug --insecure-skip-verify run --set-env=VAR_FROM_MANIFEST=setenv ./rkt-inspect-print-var-from-manifest.aci`,
		"VAR_FROM_MANIFEST=setenv",
		`^RKT_BIN^ --debug --insecure-skip-verify run --interactive --set-env=VAR_FROM_MANIFEST=setenv ./rkt-inspect-sleep.aci`,
		`/bin/sh -c "^RKT_BIN^ --debug enter $(^RKT_BIN^ list --full|grep running|awk '{print $1}') /inspect --print-env=VAR_FROM_MANIFEST"`,
	},
	{
		`/bin/sh -c "export VAR_OTHER=host ; ^RKT_BIN^ --debug --insecure-skip-verify run --inherit-env=true ./rkt-inspect-print-var-other.aci"`,
		"VAR_OTHER=host",
		`/bin/sh -c "export VAR_OTHER=host ; ^RKT_BIN^ --debug --insecure-skip-verify run --interactive --inherit-env=true ./rkt-inspect-sleep.aci"`,
		`/bin/sh -c "export VAR_OTHER=host ; ^RKT_BIN^ --debug enter $(^RKT_BIN^ list --full|grep running|awk '{print $1}') /inspect --print-env=VAR_OTHER"`,
	},
	{
		`/bin/sh -c "export VAR_FROM_MANIFEST=host ; ^RKT_BIN^ --debug --insecure-skip-verify run --inherit-env=true ./rkt-inspect-print-var-from-manifest.aci"`,
		"VAR_FROM_MANIFEST=manifest",
		`/bin/sh -c "export VAR_FROM_MANIFEST=host ; ^RKT_BIN^ --debug --insecure-skip-verify run --interactive --inherit-env=true ./rkt-inspect-sleep.aci"`,
		`/bin/sh -c "export VAR_FROM_MANIFEST=host ; ^RKT_BIN^ --debug enter $(^RKT_BIN^ list --full|grep running|awk '{print $1}') /inspect --print-env=VAR_FROM_MANIFEST"`,
	},
	{
		`/bin/sh -c "export VAR_OTHER=host ; ^RKT_BIN^ --debug --insecure-skip-verify run --inherit-env=true --set-env=VAR_OTHER=setenv ./rkt-inspect-print-var-other.aci"`,
		"VAR_OTHER=setenv",
		`/bin/sh -c "export VAR_OTHER=host ; ^RKT_BIN^ --debug --insecure-skip-verify run --interactive --inherit-env=true --set-env=VAR_OTHER=setenv ./rkt-inspect-sleep.aci"`,
		`/bin/sh -c "export VAR_OTHER=host ; ^RKT_BIN^ --debug enter $(^RKT_BIN^ list --full|grep running|awk '{print $1}') /inspect --print-env=VAR_OTHER"`,
	},
}

func TestEnv(t *testing.T) {
	patchTestACI("rkt-inspect-print-var-from-manifest.aci", "--exec=/inspect --print-env=VAR_FROM_MANIFEST")
	defer os.Remove("rkt-inspect-print-var-from-manifest.aci")
	patchTestACI("rkt-inspect-print-var-other.aci", "--exec=/inspect --print-env=VAR_OTHER")
	defer os.Remove("rkt-inspect-print-var-other.aci")
	patchTestACI("rkt-inspect-sleep.aci", "--exec=/inspect --read-stdin")
	defer os.Remove("rkt-inspect-sleep.aci")
	ctx := newRktRunCtx()
	defer ctx.cleanup()

	for i, tt := range envTests {
		// 'run' tests
		runCmd := strings.Replace(tt.runCmd, "^RKT_BIN^", ctx.cmd(), -1)
		t.Logf("Running 'run' test #%v: %v", i, runCmd)
		child, err := gexpect.Spawn(runCmd)
		if err != nil {
			t.Fatalf("Cannot exec rkt #%v: %v", i, err)
		}

		child.Capture()

		err = child.Expect(tt.runExpect)
		if err != nil {
			t.Fatalf("Expected %q but not found: %v\n\nrkt output:\n%s", tt.runExpect, err, string(child.Collect()))
		}

		err = child.Wait()
		if err != nil {
			t.Fatalf("rkt didn't terminate correctly: %v", err)
		}

		// 'enter' tests
		sleepCmd := strings.Replace(tt.sleepCmd, "^RKT_BIN^", ctx.cmd(), -1)
		t.Logf("Running 'enter' test #%v: sleep: %v", i, sleepCmd)
		child, err = gexpect.Spawn(sleepCmd)
		if err != nil {
			t.Fatalf("Cannot exec rkt #%v: %v", i, err)
		}

		err = child.Expect("Enter text:")
		if err != nil {
			t.Fatalf("Waited for the prompt but not found #%v: %v\n\nrkt output:\n%s", i, err, string(child.Collect()))
		}

		enterCmd := strings.Replace(tt.enterCmd, "^RKT_BIN^", ctx.cmd(), -1)
		t.Logf("Running 'enter' test #%v: enter: %v", i, enterCmd)
		enterChild, err := gexpect.Spawn(enterCmd)
		if err != nil {
			t.Fatalf("Cannot exec rkt #%v: %v", i, err)
		}

		err = enterChild.Expect(tt.runExpect)
		if err != nil {
			t.Fatalf("Expected %q but not found: %v\n\nrkt output:\n%s", tt.runExpect, err, string(child.Collect()))
		}

		err = enterChild.Wait()
		if err != nil {
			t.Fatalf("rkt didn't terminate correctly: %v", err)
		}
		err = child.SendLine("Bye")
		if err != nil {
			t.Fatalf("rkt couldn't write to the container: %v", err)
		}
		err = child.Expect("Received text: Bye")
		if err != nil {
			t.Fatalf("Expected Bye but not found: %v\n\nrkt output:\n%s", err, string(child.Collect()))
		}

		err = child.Wait()
		if err != nil {
			t.Fatalf("rkt didn't terminate correctly: %v", err)
		}
		ctx.reset()
	}
}
