package cmd

import (
	"context"
	"fmt"
	"os"

	"xeet/pkg/api"
	"xeet/pkg/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.design/x/clipboard"
)

var cfgFile string

// Version information
var (
	appVersion   string
	appCommit    string
	appBuildTime string
)

var rootCmd = &cobra.Command{
	Use:   "xeet",
	Short: "Terminal interface for posting to X.com",
	RunE:  runSimple,
}

func Execute() error {
	return rootCmd.Execute()
}

// SetVersion sets the version information
func SetVersion(version, commit, buildTime string) {
	appVersion = version
	appCommit = commit
	appBuildTime = buildTime
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.xeet.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")

	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

type model struct {
	textInput string
	cursor    int
	err       error
	posted    bool
	posting   bool
	hasImage  bool
	imageData []byte
}

func initialModel() model {
	return model{}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "ctrl+c" || msg.String() == "q":
			return m, tea.Quit
		case msg.Type == tea.KeyEnter && !msg.Alt:
			if !m.posting && len(m.textInput) > 0 {
				m.posting = true
				return m, postTweet(m.textInput, m.imageData)
			}
		case msg.Type == tea.KeyEnter && msg.Alt:
			if !m.posted {
				m.textInput = m.textInput[:m.cursor] + "\n" + m.textInput[m.cursor:]
				m.cursor++
			}
		case msg.String() == "alt+enter" || msg.String() == "shift+enter" || msg.Type == tea.KeyCtrlJ:
			// ctrl + j for vim bros
			if !m.posted {
				m.textInput = m.textInput[:m.cursor] + "\n" + m.textInput[m.cursor:]
				m.cursor++
			}
		case msg.String() == "ctrl+v" || msg.Type == tea.KeyCtrlV:
			if !m.posted {
				return m, pasteFromClipboard()
			}
		case msg.String() == "backspace":
			if !m.posted && len(m.textInput) > 0 && m.cursor > 0 {
				m.textInput = m.textInput[:m.cursor-1] + m.textInput[m.cursor:]
				m.cursor--
			}
		case msg.String() == "left":
			if !m.posted && m.cursor > 0 {
				m.cursor--
			}
		case msg.String() == "right":
			if !m.posted && m.cursor < len(m.textInput) {
				m.cursor++
			}
		default:
			if m.posted {
				m.posted = false
				m.textInput = ""
				m.cursor = 0
				m.err = nil
				m.hasImage = false
				m.imageData = nil
				// Handle the current key press
				if len(msg.String()) == 1 && len(m.textInput) < 280 {
					m.textInput = msg.String()
					m.cursor = 1
				}
				return m, nil
			}

			if !m.posted && len(msg.String()) == 1 && len(m.textInput) < 280 {
				m.textInput = m.textInput[:m.cursor] + msg.String() + m.textInput[m.cursor:]
				m.cursor++
			}
		}
	case pasteResult:
		if msg.imageData != nil {
			m.hasImage = true
			m.imageData = msg.imageData
		} else if msg.text != "" {
			newText := m.textInput[:m.cursor] + msg.text + m.textInput[m.cursor:]
			if len(newText) <= 280 {
				m.textInput = newText
				m.cursor += len(msg.text)
			}
		}
	case postResult:
		m.posting = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.posted = true
			m.hasImage = false
			m.imageData = nil
		}
	}
	return m, nil
}

func (m model) View() string {
	asciiArt := `
██╗  ██╗███████╗███████╗████████╗
╚██╗██╔╝██╔════╝██╔════╝╚══██╔══╝
 ╚███╔╝ █████╗  █████╗     ██║   
 ██╔██╗ ██╔══╝  ██╔══╝     ██║   
██╔╝ ██╗███████╗███████╗   ██║   
╚═╝  ╚═╝╚══════╝╚══════╝   ╚═╝   
`

	artStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3b82f6")).
		Bold(true).
		Margin(0, 0, 1, 0)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3b82f6")).
		Padding(1, 2).
		Width(60).
		Height(6)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3b82f6")).
		Bold(true)

	if m.posted {
		return artStyle.Render(asciiArt) + "\n" +
			boxStyle.Render(
				titleStyle.Render("Posted!")+"\n\n"+
					"Press any key for new tweet, q to quit",
			)
	}

	if m.posting {
		return artStyle.Render(asciiArt) + "\n" +
			boxStyle.Render(
				titleStyle.Render("Posting...")+"\n\n"+
					m.textInput,
			)
	}

	if m.err != nil {
		return artStyle.Render(asciiArt) + "\n" +
			boxStyle.Render(
				titleStyle.Render("Error")+"\n\n"+
					fmt.Sprintf("%v", m.err)+"\n\n"+
					"Press q to quit",
			)
	}

	text := m.textInput
	if m.cursor < len(text) {
		text = text[:m.cursor] + "|" + text[m.cursor:]
	} else {
		text += "|"
	}

	charCount := fmt.Sprintf("%d/280", len(m.textInput))

	displayText := text
	if m.hasImage {
		displayText = text + "\n[image1.jpg]"
	}

	return artStyle.Render(asciiArt) + "\n" +
		boxStyle.Render(
			displayText+"\n\n"+
				charCount+" • Enter to post • Ctrl+C to quit",
		)
}

type postResult struct {
	err error
}

type pasteResult struct {
	text      string
	imageData []byte
}

func postTweet(text string, imageData []byte) tea.Cmd {
	return func() tea.Msg {
		configMgr, err := config.NewConfigManager()
		if err != nil {
			return postResult{err: err}
		}

		cfg, err := configMgr.Load()
		if err != nil {
			return postResult{err: err}
		}

		if cfg.AccessToken == "" {
			return postResult{err: fmt.Errorf("run 'xeet auth' first")}
		}

		client := api.NewClient(cfg)
		err = client.PostTweetWithMedia(context.Background(), text, imageData)
		return postResult{err: err}
	}
}

func pasteFromClipboard() tea.Cmd {
	return func() tea.Msg {

		imageData := clipboard.Read(clipboard.FmtImage)
		if len(imageData) > 0 {
			return pasteResult{imageData: imageData}
		}

		textData := clipboard.Read(clipboard.FmtText)
		if len(textData) > 0 {
			return pasteResult{text: string(textData)}
		}

		return pasteResult{}
	}
}

func runSimple(cmd *cobra.Command, args []string) error {

	err := clipboard.Init()
	if err != nil {
		fmt.Printf("Failed to initialize clipboard: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(initialModel())
	_, err = p.Run()
	if err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
	return nil
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".xeet")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil && viper.GetBool("verbose") {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
