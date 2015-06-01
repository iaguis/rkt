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

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/ThomasRooney/gexpect"
)

func TestExitCode(t *testing.T) {
	for i := 0; i < 3; i++ {
		t.Logf("%d\n", i)
		patchTestACI("rkt-inspect-exit.aci", fmt.Sprintf("--exec=/inspect --print-msg=Hello --exit-code=%d", i))
		defer os.Remove("rkt-inspect-exit.aci")
		ctx := newRktRunCtx()
		defer ctx.cleanup()

		cmd := fmt.Sprintf(`/bin/sh -c "`+
			`%s --debug --insecure-skip-verify run ./rkt-inspect-exit.aci ;`+
			`UUID=$(%s list --full|grep exited|awk '{print $1}') ;`+
			`echo -n 'status=' ;`+
			`%s status $UUID|grep '^sha512-.*=[0-9]*$'|cut -d= -f2"`,
			ctx.cmd(), ctx.cmd(), ctx.cmd())
		t.Logf("%s\n", cmd)
		child, err := gexpect.Spawn(cmd)
		if err != nil {
			t.Fatalf("Cannot exec rkt")
		}

		child.Capture()

		err = child.Expect(fmt.Sprintf("status=%d", i))
		if err != nil {
			t.Fatalf("Failed to get the status\n\nrkt output:\n%s", string(child.Collect()))
		}

		err = child.Wait()
		if err != nil {
			t.Fatalf("rkt didn't terminate correctly: %v", err)
		}
	}
}
