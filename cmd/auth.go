package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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

		resp, err := newClient().Login(email, password)
		if err != nil {
			return err
		}

		cfg.Token = resp.Token
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("login succeeded but failed to save config: %w", err)
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

		resp, err := newClient().Register(email, password, name)
		if err != nil {
			return err
		}

		cfg.Token = resp.Token
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("registration succeeded but failed to save config: %w", err)
		}

		fmt.Printf("Account created for %s (%s)\n", resp.User.Name, resp.User.Email)
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

		fmt.Printf("ID:    %s\n", user.ID)
		fmt.Printf("Name:  %s\n", user.Name)
		fmt.Printf("Email: %s\n", user.Email)
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

func init() {
	authLoginCmd.Flags().String("email", "", "Email address")
	authLoginCmd.Flags().String("password", "", "Password")
	authLoginCmd.MarkFlagRequired("email")
	authLoginCmd.MarkFlagRequired("password")

	authRegisterCmd.Flags().String("email", "", "Email address")
	authRegisterCmd.Flags().String("password", "", "Password")
	authRegisterCmd.Flags().String("name", "", "Display name")
	authRegisterCmd.MarkFlagRequired("email")
	authRegisterCmd.MarkFlagRequired("password")
	authRegisterCmd.MarkFlagRequired("name")

	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authRegisterCmd)
	authCmd.AddCommand(authProfileCmd)
	authCmd.AddCommand(authLogoutCmd)
}
