package cmd

import (
	"context"
	"fmt"

	"xeet/pkg/api"
	"xeet/pkg/config"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Setup X.com API credentials",
	RunE:  runAuth,
}

func init() {
	rootCmd.AddCommand(authCmd)
}

func runAuth(cmd *cobra.Command, args []string) error {
	fmt.Println("Get credentials from https://developer.x.com/")
	fmt.Println("1. Create an app")
	fmt.Println("2. Go to 'Keys and Tokens'")
	fmt.Println("3. Generate all 4 credentials")
	fmt.Println()

	prompts := []struct {
		label string
		mask  rune
	}{
		{"API Key", 0},
		{"API Secret", '*'},
		{"Access Token", 0},
		{"Access Token Secret", '*'},
	}

	credentials := make([]string, 4)

	for i, p := range prompts {
		prompt := promptui.Prompt{
			Label: p.label,
			Mask:  p.mask,
			Validate: func(input string) error {
				if len(input) == 0 {
					return fmt.Errorf("cannot be empty")
				}
				return nil
			},
		}

		result, err := prompt.Run()
		if err != nil {
			return err
		}
		credentials[i] = result
	}

	configMgr, err := config.NewConfigManager()
	if err != nil {
		return err
	}

	cfg := &config.Config{
		APIKey:            credentials[0],
		APISecret:         credentials[1],
		AccessToken:       credentials[2],
		AccessTokenSecret: credentials[3],
	}

	if err := configMgr.Save(cfg); err != nil {
		return err
	}

	fmt.Print("Testing...")
	client := api.NewClient(cfg)

	if err := client.VerifyCredentials(context.Background()); err != nil {
		return fmt.Errorf("test failed: %v", err)
	}

	fmt.Println(" OK!")
	fmt.Println("Ready to tweet.")

	return nil
}
