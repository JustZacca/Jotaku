package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/JustZacca/jotaku/internal/api"
	"github.com/JustZacca/jotaku/internal/config"
	"github.com/JustZacca/jotaku/internal/crypto"
	"github.com/JustZacca/jotaku/internal/db"
	"github.com/JustZacca/jotaku/internal/i18n"
	"github.com/JustZacca/jotaku/internal/ui"
	"golang.org/x/term"
)

func main() {
	// Show logo on startup
	printLogo()

	configPath := config.DefaultConfigPath()

	// Check if config exists, if not run first-time setup
	if !config.ConfigExists(configPath) {
		if err := firstTimeSetup(configPath); err != nil {
			fmt.Fprintf(os.Stderr, "Setup error: %v\n", err)
			os.Exit(1)
		}
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Apply language setting
	if cfg.Language != "" {
		i18n.SetLanguage(i18n.Language(cfg.Language))
	}

	// Prompt for master password
	password, err := promptPassword()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T().Error, err)
		os.Exit(1)
	}

	// Setup encryption
	var enc *crypto.Encryptor
	salt, err := cfg.GetSalt()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T().Error, err)
		os.Exit(1)
	}

	if salt == nil {
		salt, err = crypto.GenerateSalt()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T().Error, err)
			os.Exit(1)
		}
		cfg.SetSalt(salt)
		if err := cfg.Save(configPath); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T().Error, err)
			os.Exit(1)
		}
	}

	enc = crypto.NewEncryptor(password, salt)

	// Auto-login if server is configured
	if cfg.Server.URL != "" && cfg.Server.Enabled {
		if err := autoLogin(cfg, password, configPath); err != nil {
			// Non-fatal: continue in offline mode
			fmt.Fprintf(os.Stderr, "Server: %v (%s)\n", err, i18n.T().Offline)
		}
	}

	// Initialize database
	database, err := db.New(cfg.DBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T().Error, err)
		os.Exit(1)
	}
	defer database.Close()

	// Start TUI
	m := ui.NewModel(database, enc, cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T().Error, err)
		os.Exit(1)
	}
}

func printLogo() {
	fmt.Println()
	fmt.Println("       ██╗ ██████╗ ████████╗ █████╗ ██╗  ██╗██╗   ██╗")
	fmt.Println("       ██║██╔═══██╗╚══██╔══╝██╔══██╗██║ ██╔╝██║   ██║")
	fmt.Println("       ██║██║   ██║   ██║   ███████║█████╔╝ ██║   ██║")
	fmt.Println("  ██   ██║██║   ██║   ██║   ██╔══██║██╔═██╗ ██║   ██║")
	fmt.Println("  ╚█████╔╝╚██████╔╝   ██║   ██║  ██║██║  ██╗╚██████╔╝")
	fmt.Println("   ╚════╝  ╚═════╝    ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝")
	fmt.Println()
}

func firstTimeSetup(configPath string) error {
	fmt.Println("  Welcome to Jotaku! / Benvenuto in Jotaku!")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// Ask for language
	fmt.Println("  Select language / Seleziona lingua:")
	fmt.Println("  [1] English")
	fmt.Println("  [2] Italiano")
	fmt.Print("  > ")

	choice, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	choice = strings.TrimSpace(choice)

	language := "en"
	if choice == "2" {
		language = "it"
	}

	// Set language immediately for subsequent prompts
	i18n.SetLanguage(i18n.Language(language))

	// Create default config
	cfg := &config.Config{
		DBPath:   config.DefaultDBPath(),
		Language: language,
		Theme:    "dark",
	}

	// Save config
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println()
	if language == "it" {
		fmt.Println("  Configurazione creata!")
		fmt.Println("  Modifica config.yml per personalizzare.")
	} else {
		fmt.Println("  Configuration created!")
		fmt.Println("  Edit config.yml to customize.")
	}
	fmt.Println()

	return nil
}

func promptPassword() (string, error) {
	fmt.Print(i18n.T().MasterPassword)

	if term.IsTerminal(int(os.Stdin.Fd())) {
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return "", fmt.Errorf("%s: %w", i18n.T().Error, err)
		}
		return strings.TrimSpace(string(password)), nil
	}

	reader := bufio.NewReader(os.Stdin)
	password, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T().Error, err)
	}
	return strings.TrimSpace(password), nil
}

func autoLogin(cfg *config.Config, masterPassword string, configPath string) error {
	client := api.NewClient(cfg.Server.URL)

	// Check if server is reachable
	if err := client.Ping(); err != nil {
		return fmt.Errorf("server unreachable")
	}

	// If we have a token, validate it
	if cfg.Server.Token != "" {
		client.SetToken(cfg.Server.Token)
		// Token exists, assume it's valid (will fail on sync if not)
		return nil
	}

	// If we have username but no token, try login
	if cfg.Server.Username != "" {
		resp, err := client.Login(cfg.Server.Username, masterPassword)
		if err != nil {
			return fmt.Errorf("login failed")
		}
		cfg.Server.Token = resp.Token
		cfg.Save(configPath)
		return nil
	}

	// No username configured - skip auto-login
	// User needs to configure server.username in config.yml
	return fmt.Errorf("no username configured")
}
