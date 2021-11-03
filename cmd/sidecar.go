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
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/topfreegames/pitaya/pkg/config"
	"github.com/topfreegames/pitaya/sidecar"
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
		cfg := config.NewConfig()
		sidecar.StartSidecar(cfg, debug, bind, bindProtocol)
	},
}

func init() {
	sidecarCmd.Flags().BoolVarP(&debug, "debug", "d", false, "turn debug on")
	sidecarCmd.Flags().StringVarP(&bind, "bind", "b", filepath.FromSlash(fmt.Sprintf("%s/pitaya.sock", os.TempDir())), "bind address of the sidecar")
	sidecarCmd.Flags().StringVarP(&bindProtocol, "bindProtocol", "p", "unix", "bind address of the sidecar")
	rootCmd.AddCommand(sidecarCmd)
}
