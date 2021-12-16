package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/johnnyipcom/parseuser/reddit"
	"github.com/mitchellh/go-homedir"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Root struct {
	cfgFile string
	client  *reddit.Reddit
	version string
}

func NewRoot(version string) (*Root, error) {
	client, err := reddit.NewReadonly()
	if err != nil {
		return nil, err
	}

	root := &Root{
		version: version,
		client:  client,
	}

	cobra.OnInitialize(root.initConfig)
	return root, nil
}

func (r *Root) Run(ctx context.Context) error {
	rootCmd := &cobra.Command{
		Use:   "yars",
		Short: "Yet Another Reddit Scraper",
		Long:  "Yet Another Reddit Scraper",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, []string{})
		},
	}

	rootCmd.PersistentFlags().StringVarP(&r.cfgFile, "config", "c", "", "config file (default \"$HOME/.yars.yaml\")")
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "print version info",
		Long:  "print version info",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("yars %s\n", r.version)
		},
	}

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(newUserCommand(ctx, r.client))
	rootCmd.AddCommand(newSubredditCommand(ctx, r.client))

	return rootCmd.Execute()
}

func (r *Root) initConfig() {
	if r.cfgFile != "" {
		viper.SetConfigFile(r.cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".yars")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
