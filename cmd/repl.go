/*
Copyright 2024 Wildlife Studios

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
	"github.com/topfreegames/pitaya/v3/repl"
)

var docsRoute string
var fileName string
var prettyJSON bool

// replCmd opens pitaya REPL tool
var replCmd = &cobra.Command{
	Use:   "repl",
	Short: "starts pitaya repl tool",
	Long:  `starts pitaya repl tool`,
	Run: func(cmd *cobra.Command, args []string) {
		repl.Start(docsRoute, fileName, prettyJSON)
	},
}

func init() {
	replCmd.Flags().StringVarP(&docsRoute, "docs", "d", "", "route containing the documentation")
	replCmd.Flags().StringVarP(&fileName, "filename", "f", "", "file containing the commands to run")
	replCmd.Flags().BoolVarP(&prettyJSON, "pretty", "p", false, "print pretty jsons")
	rootCmd.AddCommand(replCmd)
}
