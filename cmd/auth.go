package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in and save credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("email")
		password, _ := cmd.Flags().GetString("password")

		if !cmd.Flags().Changed("password") {
			p, err := promptPassword("Password: ")
			if err != nil {
				return err
			}
			password = p
		}

		resp, err := newClient().Login(email, password)
		if err != nil {
			return err
		}

		cfg.Token = resp.Token
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("login succeeded but failed to save config: %w", err)
		}

		if isJSON() {
			data, _ := json.MarshalIndent(map[string]any{
				"id":    resp.User.ID,
				"name":  resp.User.Name,
				"email": resp.User.Email,
			}, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Logged in as %s (%s)\n", resp.User.Name, resp.User.Email)
		return nil
	},
}

var authRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Create a new account",
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("email")
		password, _ := cmd.Flags().GetString("password")
		name, _ := cmd.Flags().GetString("name")

		if !cmd.Flags().Changed("password") {
			p, err := promptPassword("Password: ")
			if err != nil {
				return err
			}
			password = p
		}

		resp, err := newClient().Register(email, password, name)
		if err != nil {
			return err
		}

		// Auto-login: save the token so the user doesn't have to run auth login.
		loginResp, err := newClient().Login(email, password)
		if err == nil {
			cfg.Token = loginResp.Token
			_ = cfg.Save()
		}

		if isJSON() {
			data, _ := json.MarshalIndent(map[string]any{
				"id":    resp.User.ID,
				"name":  resp.User.Name,
				"email": resp.User.Email,
			}, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Account created for %s (%s).\n", resp.User.Name, resp.User.Email)
		if loginResp != nil {
			fmt.Println("Logged in automatically.")
		}
		return nil
	},
}

var authProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Show the authenticated user's profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()

		user, err := newClient().Profile()
		if err != nil {
			return err
		}

		if isJSON() {
			data, _ := json.MarshalIndent(user, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("ID:      %d\n", user.ID)
		fmt.Printf("Name:    %s\n", user.Name)
		fmt.Printf("Email:   %s\n", user.Email)
		fmt.Printf("Created: %s\n", user.CreatedAt.Format(time.RFC3339))
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove saved credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg.Token = ""
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, "Logged out")
		return nil
	},
}

// promptPassword prints a prompt to stderr and reads a password with echo
// suppressed when stdin is a terminal.
func promptPassword(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)

	if term.IsTerminal(int(os.Stdin.Fd())) {
		pw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr) // newline after hidden input
		if err != nil {
			return "", err
		}
		return string(pw), nil
	}

	// Non-interactive: read normally (e.g. piped input).
	var password string
	_, err := fmt.Fscan(os.Stdin, &password)
	return password, err
}

func init() {
	authLoginCmd.Flags().String("email", "", "Email address")
	authLoginCmd.Flags().String("password", "", "Password (prompted if omitted)")
	authLoginCmd.MarkFlagRequired("email")

	authRegisterCmd.Flags().String("email", "", "Email address")
	authRegisterCmd.Flags().String("password", "", "Password (prompted if omitted)")
	authRegisterCmd.Flags().String("name", "", "Display name")
	authRegisterCmd.MarkFlagRequired("email")
	authRegisterCmd.MarkFlagRequired("name")

	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authRegisterCmd)
	authCmd.AddCommand(authProfileCmd)
	authCmd.AddCommand(authLogoutCmd)
}
