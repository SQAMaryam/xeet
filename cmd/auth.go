package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

	"xeet/pkg/api"
	"xeet/pkg/config"

	"github.com/dghubble/oauth1"
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
	fmt.Println("\nSetting up X.com authentication...")
	fmt.Println()
	
	// Check if user wants easy OAuth flow or manual setup
	prompt := promptui.Select{
		Label: "Choose authentication method",
		Items: []string{
			"Easy Setup (PIN-based OAuth with browser)",
			"Manual Setup (enter all 4 API keys)",
		},
	}
	
	choice, _, err := prompt.Run()
	if err != nil {
		return fmt.Errorf("selection failed: %w", err)
	}
	
	if choice == 0 {
		return runEasyAuth()
	} else {
		return runManualAuth()
	}
}

func runEasyAuth() error {
	fmt.Println("\nEasy PIN-based OAuth Setup")
	fmt.Println("You still need API keys from X.com, but this is much easier!")
	fmt.Println()
	
	// Get API keys first
	fmt.Println("STEP 1: Get your app's API keys")
	fmt.Println("Go to https://developer.x.com/ and create an app")
	fmt.Println()
	
	apiKeyPrompt := promptui.Prompt{
		Label: "API Key (Consumer Key)",
		Validate: func(input string) error {
			if len(input) == 0 {
				return fmt.Errorf("cannot be empty")
			}
			return nil
		},
	}
	
	apiKey, err := apiKeyPrompt.Run()
	if err != nil {
		return err
	}
	
	apiSecretPrompt := promptui.Prompt{
		Label: "API Secret (Consumer Secret)",
		Mask:  '*',
		Validate: func(input string) error {
			if len(input) == 0 {
				return fmt.Errorf("cannot be empty")
			}
			return nil
		},
	}
	
	apiSecret, err := apiSecretPrompt.Run()
	if err != nil {
		return err
	}
	
	fmt.Println("\nSTEP 2: Authorize with X.com")
	fmt.Println()
	
	// Set up OAuth config for PIN-based flow
	oauthConfig := oauth1.Config{
		ConsumerKey:    apiKey,
		ConsumerSecret: apiSecret,
		CallbackURL:    "oob", // "out of band" for PIN-based flow
		Endpoint: oauth1.Endpoint{
			RequestTokenURL: "https://api.twitter.com/oauth/request_token",
			AuthorizeURL:    "https://api.twitter.com/oauth/authorize",
			AccessTokenURL:  "https://api.twitter.com/oauth/access_token",
		},
	}
	
	// Step 1: Get request token
	requestToken, requestSecret, err := oauthConfig.RequestToken()
	if err != nil {
		return fmt.Errorf("failed to get request token: %w", err)
	}
	
	// Step 2: Get authorization URL
	authURL, err := oauthConfig.AuthorizationURL(requestToken)
	if err != nil {
		return fmt.Errorf("failed to get authorization URL: %w", err)
	}
	
	fmt.Printf("Opening browser to: %s\n", authURL.String())
	fmt.Println("1. Click 'Authorize app' on X.com")
	fmt.Println("2. You'll see a PIN code")
	fmt.Println("3. Enter the PIN below")
	fmt.Println()
	
	// Open browser
	openBrowser(authURL.String())
	
	// Get PIN from user
	pinPrompt := promptui.Prompt{
		Label: "Enter PIN from X.com",
		Validate: func(input string) error {
			if len(input) == 0 {
				return fmt.Errorf("cannot be empty")
			}
			return nil
		},
	}
	
	pin, err := pinPrompt.Run()
	if err != nil {
		return err
	}
	
	// Step 3: Exchange PIN for access tokens
	accessToken, accessSecret, err := oauthConfig.AccessToken(requestToken, requestSecret, pin)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}
	
	// Save configuration
	configMgr, err := config.NewConfigManager()
	if err != nil {
		return err
	}
	
	cfg := &config.Config{
		APIKey:            apiKey,
		APISecret:         apiSecret,
		AccessToken:       accessToken,
		AccessTokenSecret: accessSecret,
	}
	
	if err := configMgr.Save(cfg); err != nil {
		return err
	}
	
	fmt.Print("\nTesting credentials...")
	client := api.NewClient(cfg)
	
	if err := client.VerifyCredentials(context.Background()); err != nil {
		return fmt.Errorf("test failed: %v", err)
	}
	
	fmt.Println(" Success!")
	fmt.Println("You're ready to use xeet!")
	
	return nil
}

// openBrowser tries to open the URL in the user's default browser
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		fmt.Printf("Please open this URL manually: %s\n", url)
		return
	}
	
	if err := cmd.Start(); err != nil {
		fmt.Printf("Couldn't open browser. Please visit: %s\n", url)
	}
}

func runManualAuth() error {
	fmt.Println("\nManual API Key Setup")
	fmt.Println()
	fmt.Println("STEP 1: Get API keys from https://developer.x.com/")
	fmt.Println("  • Create an app")
	fmt.Println("  • Go to 'Keys and Tokens'")
	fmt.Println("  • Generate all 4 credentials")
	fmt.Println()
	fmt.Println("STEP 2: Enter your credentials below")
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

	fmt.Print("\nTesting credentials...")
	client := api.NewClient(cfg)

	if err := client.VerifyCredentials(context.Background()); err != nil {
		return fmt.Errorf("test failed: %v", err)
	}

	fmt.Println(" Success!")
	fmt.Println("You're ready to use xeet!")

	return nil
}
