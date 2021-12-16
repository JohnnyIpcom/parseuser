package cmd

import (
	"context"

	"github.com/johnnyipcom/parseuser/reddit"
	"github.com/spf13/cobra"
)

func newUserCommand(ctx context.Context, client *reddit.Reddit) *cobra.Command {
	userCmd := &cobra.Command{
		Use:   "user",
		Short: "get all media from user",
		Long:  "get all media from user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dirname, err := cmd.Flags().GetString("output")
			if err != nil {
				return err
			}

			return client.Download(ctx, dirname, client.GetURLsFromUser(ctx, args[0]))
		},
	}

	userCmd.Flags().StringP("output", "o", "./downloaded", "output directory")
	return userCmd
}
