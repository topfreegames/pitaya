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
	"github.com/spf13/cobra"
	"github.com/topfreegames/pitaya/v3/pkg/config"
	"github.com/topfreegames/pitaya/v3/sidecar"
)

var sidecarConfig *config.SidecarConfig

// sidecarCmd represents the start command
var sidecarCmd = &cobra.Command{
	Use:   "sidecar",
	Short: "starts pitaya in sidecar mode",
	Long:  `starts pitaya in sidecar mode`,
	Run: func(cmd *cobra.Command, args []string) {
		sc := sidecar.NewSidecar(sidecarConfig.CallTimeout)

		pitayaConfig := config.NewDefaultBuilderConfig()
		server := sidecar.NewServer(sc, *pitayaConfig)

		server.Start(sidecarConfig.Bind, sidecarConfig.BindProtocol)
	},
}

func init() {
	sidecarConfig = config.NewDefaultSidecarConfig()
	sidecarCmd.Flags().StringVarP(&sidecarConfig.Bind, "bind", "b", sidecarConfig.Bind, "bind address of the sidecar")
	sidecarCmd.Flags().StringVarP(&sidecarConfig.BindProtocol, "bindProtocol", "p", sidecarConfig.BindProtocol, "bind  protocol of the sidecar")
	rootCmd.AddCommand(sidecarCmd)
}
