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

package image

import (
	"fmt"
	"os"
)

// stderr prints messages to standard error. There is no need to add a
// trailing newline as it will be added automatically. Also, this
// function prepends a "rkt: " string to the message.
func stderr(format string, a ...interface{}) {
	prefixedFormat := fmt.Sprintf("rkt: %s", format)
	out := fmt.Sprintf(prefixedFormat, a...)
	fmt.Fprintln(os.Stderr, strings.TrimSuffix(out, "\n"))
}
