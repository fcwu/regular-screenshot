package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	rootCmd = &cobra.Command{
		Use:  "regular-screenshot",
		Long: "Take screenshot of the desktop and save it to a samba share regularly",
		RunE: serve,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			sambaHost := viper.GetString("samba_host")
			if sambaHost == "" {
				return errors.New("Samba host is required")
			}
			desktopUsername := viper.GetString("desktop_username")
			if desktopUsername == "" {
				return errors.New("Desktop username is required")
			}

			interval := viper.GetInt("interval")
			if interval < 1 {
				return errors.New("Interval must be greater than 0")
			}
			return nil
		},
	}
)

func main() {
	viper.AutomaticEnv()
	viper.SetEnvPrefix("DONG")

	rootCmd.Flags().StringP("samba-host", "", "", "your-samba-server:445")
	rootCmd.Flags().StringP("folder", "", "", "aaa/bbb/ccc")
	rootCmd.Flags().StringP("samba-username", "n", "", "")
	rootCmd.Flags().StringP("samba-password", "p", "", "")
	rootCmd.Flags().StringP("samba-share", "s", "", "shared folder")
	rootCmd.Flags().StringP("desktop-username", "", "", "user to be monitored")
	rootCmd.Flags().IntP("interval", "i", 5, "interval in seconds")
	rootCmd.Flags().BoolP("verbose", "v", false, "enable verbose mode")

	_ = viper.BindPFlag("samba_host", rootCmd.Flags().Lookup("samba-host"))
	_ = viper.BindPFlag("samba_username", rootCmd.Flags().Lookup("samba-username"))
	_ = viper.BindPFlag("samba_password", rootCmd.Flags().Lookup("samba-password"))
	_ = viper.BindPFlag("samba_share", rootCmd.Flags().Lookup("samba-share"))
	_ = viper.BindPFlag("desktop_username", rootCmd.Flags().Lookup("desktop-username"))
	_ = viper.BindPFlag("interval", rootCmd.Flags().Lookup("interval"))
	_ = viper.BindPFlag("folder", rootCmd.Flags().Lookup("folder"))

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		return
	}
}
