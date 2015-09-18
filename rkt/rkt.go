// Copyright 2014 The rkt Authors
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
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/spf13/cobra"
	"github.com/coreos/rkt/common"
	"github.com/coreos/rkt/pkg/keystore"
	"github.com/coreos/rkt/pkg/multicall"
	"github.com/coreos/rkt/rkt/config"
)

const (
	cliName        = "rkt"
	cliDescription = "rkt, the application container runner"

	defaultDataDir = "/var/lib/rkt"
)

const (
	bash_completion_func = `__rkt_parse_image()
{
    local rkt_output
    if rkt_output=$(rkt image list --no-legend 2>/dev/null); then
        out=($(echo "${rkt_output}" | awk '{print $1}'))
        COMPREPLY=( $( compgen -W "${out[*]}" -- "$cur" ) )
    fi
}

__rkt_parse_list()
{
    local rkt_output
    if rkt_output=$(rkt list --no-legend 2>/dev/null); then
        if [[ -n "$1" ]]; then
            out=($(echo "${rkt_output}" | grep ${1} | awk '{print $1}'))
        else
            out=($(echo "${rkt_output}" | awk '{print $1}'))
        fi
        COMPREPLY=( $( compgen -W "${out[*]}" -- "$cur" ) )
    fi
}

__custom_func() {
    case ${last_command} in
        rkt_image_export | \
        rkt_image_extract | \
        rkt_image_cat-manifest | \
        rkt_image_render | \
        rkt_image_rm | \
        rkt_run | \
        rkt_prepare)
            __rkt_parse_image
            return
            ;;
        rkt_run-prepared)
            __rkt_parse_list prepared
            return
            ;;
        rkt_enter)
            __rkt_parse_list running
            return
            ;;
        rkt_rm)
            __rkt_parse_list "prepare\|exited"
            return
            ;;
        rkt_status)
            __rkt_parse_list
            return
            ;;
        *)
            ;;
    esac
}
`
)

var (
	tabOut      *tabwriter.Writer
	globalFlags = struct {
		Dir                string
		SystemConfigDir    string
		LocalConfigDir     string
		Debug              bool
		Help               bool
		InsecureSkipVerify bool
		TrustKeysFromHttps bool
	}{}

	cmdExitCode int
)

var cmdRkt = &cobra.Command{
	Use:   "rkt [command]",
	Short: cliDescription,
	BashCompletionFunction: bash_completion_func,
}

func init() {
	cmdRkt.PersistentFlags().BoolVar(&globalFlags.Debug, "debug", false, "Print out more debug information to stderr")
	cmdRkt.PersistentFlags().StringVar(&globalFlags.Dir, "dir", defaultDataDir, "rkt data directory")
	cmdRkt.PersistentFlags().StringVar(&globalFlags.SystemConfigDir, "system-config", common.DefaultSystemConfigDir, "system configuration directory")
	cmdRkt.PersistentFlags().StringVar(&globalFlags.LocalConfigDir, "local-config", common.DefaultLocalConfigDir, "local configuration directory")
	cmdRkt.PersistentFlags().BoolVar(&globalFlags.InsecureSkipVerify, "insecure-skip-verify", false, "skip all TLS, image or fingerprint verification")
	cmdRkt.PersistentFlags().BoolVar(&globalFlags.TrustKeysFromHttps, "trust-keys-from-https", true, "automatically trust gpg keys fetched from https")
}

func init() {
	tabOut = new(tabwriter.Writer)
	tabOut.Init(os.Stdout, 0, 8, 1, '\t', 0)

	cobra.EnablePrefixMatching = true
}

// runWrapper return a func(cmd *cobra.Command, args []string) that internally
// will add command function return code and the reinsertion of the "--" flag
// terminator.
func runWrapper(cf func(cmd *cobra.Command, args []string) (exit int)) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		cmdExitCode = cf(cmd, args)
	}
}

func main() {
	// check if rkt is executed with a multicall command
	multicall.MaybeExec()

	cmdRkt.SetUsageFunc(usageFunc)

	// Make help just show the usage
	cmdRkt.SetHelpTemplate(`{{.UsageString}}`)

	cmdRkt.Execute()
	os.Exit(cmdExitCode)
}

func stderr(format string, a ...interface{}) {
	out := fmt.Sprintf(format, a...)
	fmt.Fprintln(os.Stderr, strings.TrimSuffix(out, "\n"))
}

func stdout(format string, a ...interface{}) {
	out := fmt.Sprintf(format, a...)
	fmt.Fprintln(os.Stdout, strings.TrimSuffix(out, "\n"))
}

// where pod directories are created and locked before moving to prepared
func embryoDir() string {
	return filepath.Join(globalFlags.Dir, "pods", "embryo")
}

// where pod trees reside during (locked) and after failing to complete preparation (unlocked)
func prepareDir() string {
	return filepath.Join(globalFlags.Dir, "pods", "prepare")
}

// where pod trees reside upon successful preparation
func preparedDir() string {
	return filepath.Join(globalFlags.Dir, "pods", "prepared")
}

// where pod trees reside once run
func runDir() string {
	return filepath.Join(globalFlags.Dir, "pods", "run")
}

// where pod trees reside once exited & marked as garbage by a gc pass
func exitedGarbageDir() string {
	return filepath.Join(globalFlags.Dir, "pods", "exited-garbage")
}

// where never-executed pod trees reside once marked as garbage by a gc pass (failed prepares, expired prepareds)
func garbageDir() string {
	return filepath.Join(globalFlags.Dir, "pods", "garbage")
}

func getKeystore() *keystore.Keystore {
	if globalFlags.InsecureSkipVerify {
		return nil
	}
	config := keystore.NewConfig(globalFlags.SystemConfigDir, globalFlags.LocalConfigDir)
	return keystore.New(config)
}

func getConfig() (*config.Config, error) {
	return config.GetConfigFrom(globalFlags.SystemConfigDir, globalFlags.LocalConfigDir)
}
