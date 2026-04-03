package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var fsPwdCmd = &cobra.Command{
	Use:   "pwd",
	Short: "Print current working directory",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		session := initFsSession(ctx)
		if fsJSON {
			printJSON(map[string]string{"cwd": session.Pwd()})
		} else {
			fmt.Println(session.Pwd())
		}
	},
}
