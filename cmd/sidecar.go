/*
Copyright © 2021 Wildlife Studios

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"github.com/topfreegames/pitaya/v2/pkg/config"
	"github.com/topfreegames/pitaya/v2/sidecar"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var debug bool
var bind string
var bindProtocol string

// sidecarCmd represents the start command
var sidecarCmd = &cobra.Command{
	Use:   "sidecar",
	Short: "starts pitaya in sidecar mode",
	Long:  `starts pitaya in sidecar mode`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.NewBuilderConfig(config.NewConfig())
		sidecar := sidecar.NewSidecar(*cfg, debug)
		sidecar.StartSidecar(bind, bindProtocol)
	},
}

func init() {
	tmpDir := os.TempDir()
	sidecarCmd.Flags().BoolVarP(&debug, "debug", "d", false, "turn debug on")
	sidecarCmd.Flags().StringVarP(&bind, "bind", "b", filepath.FromSlash(fmt.Sprintf("%s/pitaya.sock", strings.TrimSuffix(tmpDir, "/"))), "bind address of the sidecar")
	sidecarCmd.Flags().StringVarP(&bindProtocol, "bindProtocol", "p", "unix", "bind address of the sidecar")
	rootCmd.AddCommand(sidecarCmd)
}
