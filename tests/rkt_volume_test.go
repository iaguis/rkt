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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/ThomasRooney/gexpect"
)

var volTests = []struct {
	rktCmd string
	expect string
}{
	// Check that we can read files in the ACI
	{
		`/bin/sh -c "export FILE=/dir1/file ; ^RKT_BIN^ --debug --insecure-skip-verify run --inherit-env=true ./rkt-inspect-read-file.aci"`,
		`<<<dir1>>>`,
	},
	// Check that we can read files from a volume (both ro and rw)
	{
		`/bin/sh -c "export FILE=/dir1/file ; ^RKT_BIN^ --debug --insecure-skip-verify run --inherit-env=true --volume=dir1,kind=host,source=^TMPDIR^ ./rkt-inspect-vol-rw-read-file.aci"`,
		`<<<host>>>`,
	},
	{
		`/bin/sh -c "export FILE=/dir1/file ; ^RKT_BIN^ --debug --insecure-skip-verify run --inherit-env=true --volume=dir1,kind=host,source=^TMPDIR^ ./rkt-inspect-vol-ro-read-file.aci"`,
		`<<<host>>>`,
	},
	// Check that we can write to files in the ACI
	{
		`/bin/sh -c "export FILE=/dir1/file CONTENT=1 ; ^RKT_BIN^ --debug --insecure-skip-verify run --inherit-env=true ./rkt-inspect-write-file.aci"`,
		`<<<1>>>`,
	},
	// Check that we can write files to a volume (both ro and rw)
	{
		`/bin/sh -c "export FILE=/dir1/file CONTENT=2 ; ^RKT_BIN^ --debug --insecure-skip-verify run --inherit-env=true --volume=dir1,kind=host,source=^TMPDIR^ ./rkt-inspect-vol-rw-write-file.aci"`,
		`<<<2>>>`,
	},
	{
		`/bin/sh -c "export FILE=/dir1/file CONTENT=3 ; ^RKT_BIN^ --debug --insecure-skip-verify run --inherit-env=true --volume=dir1,kind=host,source=^TMPDIR^ ./rkt-inspect-vol-ro-write-file.aci"`,
		`Cannot write to file "/dir1/file": open /dir1/file: read-only file system`,
	},
	// Check that the volume still contain the file previously written
	{
		`/bin/sh -c "export FILE=/dir1/file ; ^RKT_BIN^ --debug --insecure-skip-verify run --inherit-env=true --volume=dir1,kind=host,source=^TMPDIR^ ./rkt-inspect-vol-ro-read-file.aci"`,
		`<<<2>>>`,
	},
}

func TestVolumes(t *testing.T) {
	patchTestACI("rkt-inspect-read-file.aci", "--exec=/inspect --read-file")
	defer os.Remove("rkt-inspect-read-file.aci")
	patchTestACI("rkt-inspect-write-file.aci", "--exec=/inspect --write-file --read-file")
	defer os.Remove("rkt-inspect-write-file.aci")
	patchTestACI("rkt-inspect-vol-rw-read-file.aci", "--exec=/inspect --read-file", "--mounts=dir1,path=/dir1,readOnly=false")
	defer os.Remove("rkt-inspect-vol-rw-read-file.aci")
	patchTestACI("rkt-inspect-vol-rw-write-file.aci", "--exec=/inspect --write-file --read-file", "--mounts=dir1,path=/dir1,readOnly=false")
	defer os.Remove("rkt-inspect-vol-rw-write-file.aci")
	patchTestACI("rkt-inspect-vol-ro-read-file.aci", "--exec=/inspect --read-file", "--mounts=dir1,path=/dir1,readOnly=true")
	defer os.Remove("rkt-inspect-vol-ro-read-file.aci")
	patchTestACI("rkt-inspect-vol-ro-write-file.aci", "--exec=/inspect --write-file --read-file", "--mounts=dir1,path=/dir1,readOnly=true")
	defer os.Remove("rkt-inspect-vol-ro-write-file.aci")
	ctx := newRktRunCtx()
	defer ctx.cleanup()

	tmpdir, err := ioutil.TempDir("", "rkt-tests.")
	if err != nil {
		t.Fatalf("Cannot create temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpdir)

	tmpfile := filepath.Join(tmpdir, "file")
	err = ioutil.WriteFile(tmpfile, []byte("host"), 0600)
	if err != nil {
		t.Fatalf("Cannot create temporary file: %v", err)
	}

	for i, tt := range volTests {
		cmd := strings.Replace(tt.rktCmd, "^TMPDIR^", tmpdir, -1)
		cmd = strings.Replace(cmd, "^RKT_BIN^", ctx.cmd(), -1)

		t.Logf("Running test #%v: %v", i, cmd)

		child, err := gexpect.Spawn(cmd)
		if err != nil {
			t.Fatalf("Cannot exec rkt #%v: %v", i, err)
		}

		err = expectWithOutput(child, tt.expect)
		if err != nil {
			fmt.Printf("Command: %s\n", cmd)
			t.Fatalf("Expected %q but not found #%v: %v", tt.expect, i, err)
		}

		err = child.Wait()
		if err != nil {
			t.Fatalf("rkt didn't terminate correctly: %v", err)
		}
	}
}
