package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/claudeswap/claudeswap/internal/profile"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newRemoveCmd(app *AppContext) *cobra.Command {
	var deleteDataFlag bool
	var forceFlag bool

	cmd := &cobra.Command{
		Use:   "remove <profile>",
		Short: "Remove a Claude profile",
		Long: `Remove a profile from ClaudeSwap metadata.
You can choose whether to keep or delete the local isolated configuration directory.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			p, err := app.Manager.GetProfile(target)
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Profile %q was not found.\n", target)
				printAvailableProfiles(app)
				os.Exit(ExitProfileNotFound)
			}

			// Non-interactive path with flags
			if forceFlag {
				return executeRemoval(app, p, deleteDataFlag)
			}

			// Interactive removal prompt
			fmt.Println("\nRemove Profile")
			fmt.Println("--------------")
			fmt.Printf("Profile:       %s\n", p.Name)
			fmt.Printf("Shortcut:      %s\n", p.Shortcut)
			fmt.Printf("Configuration: %s\n\n", p.ConfigDir)

			if deleteDataFlag {
				fmt.Printf("WARNING: You selected to delete all local configuration data.\nAre you sure you want to permanently delete %s? [y/N]: ", p.ConfigDir)
				reader := bufio.NewReader(os.Stdin)
				ans, _ := reader.ReadString('\n')
				if strings.ToLower(strings.TrimSpace(ans)) != "y" {
					fmt.Println("Operation cancelled.")
					return nil
				}
				return executeRemoval(app, p, true)
			}

			choice := promptRemovalChoice()

			switch choice {
			case "1":
				return executeRemoval(app, p, false)
			case "2":
				fmt.Print("Are you sure you want to permanently delete local configuration? [y/N]: ")
				reader := bufio.NewReader(os.Stdin)
				confirmStr, _ := reader.ReadString('\n')
				if strings.ToLower(strings.TrimSpace(confirmStr)) != "y" {
					fmt.Println("Deletion cancelled.")
					return nil
				}
				return executeRemoval(app, p, true)
			default:
				fmt.Println("Operation cancelled.")
				return nil
			}
		},
	}

	cmd.Flags().BoolVar(&deleteDataFlag, "delete-data", false, "Also remove local profile configuration directory")
	cmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "Do not prompt for confirmation")

	return cmd
}

func executeRemoval(app *AppContext, p *profile.Profile, removeData bool) error {
	// First remove shortcut
	_ = app.Shortcut.RemoveShortcut(p.Shortcut)

	// Remove from manager
	if err := app.Manager.DeleteProfile(p.ID, removeData); err != nil {
		fmt.Fprintf(os.Stderr, "✗ Error removing profile: %v\n", err)
		os.Exit(ExitGeneralError)
	}

	fmt.Printf("✓ Profile %q removed successfully.\n", p.Name)
	if removeData {
		fmt.Printf("✓ Local configuration directory deleted: %s\n", p.ConfigDir)
	} else {
		fmt.Printf("✓ Local configuration kept intact: %s\n", p.ConfigDir)
	}

	return nil
}

// PromptRemove provides an interactive confirmation dialog for removing a profile.
func PromptRemove(app *AppContext, p *profile.Profile) error {
	if p == nil {
		return nil
	}

	fmt.Println("\nRemove Profile")
	fmt.Println("--------------")
	fmt.Printf("Profile:       %s\n", p.Name)
	fmt.Printf("Shortcut:      %s\n", p.Shortcut)
	fmt.Printf("Configuration: %s\n\n", p.ConfigDir)

	choice := promptRemovalChoice()
	switch choice {
	case "1":
		return executeRemoval(app, p, false)
	case "2":
		fmt.Print("Are you sure you want to permanently delete local configuration? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		confirmStr, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(confirmStr)) != "y" {
			fmt.Println("Deletion cancelled.")
			return nil
		}
		return executeRemoval(app, p, true)
	default:
		fmt.Println("Operation cancelled.")
		return nil
	}
}

func promptRemovalChoice() string {
	fmt.Println("Choose an action:")
	fmt.Println("  [1] Keep configuration files (remove profile metadata only)")
	fmt.Println("  [2] Remove profile and delete local configuration files")
	fmt.Println("  [3/q] Cancel")
	fmt.Print("\nSelect option (1, 2, or 3): ")

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		reader := bufio.NewReader(os.Stdin)
		s, _ := reader.ReadString('\n')
		return strings.TrimSpace(s)
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		reader := bufio.NewReader(os.Stdin)
		s, _ := reader.ReadString('\n')
		return strings.TrimSpace(s)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	var buf [1]byte
	for {
		n, err := os.Stdin.Read(buf[:])
		if err != nil || n == 0 {
			return "3"
		}
		ch := buf[0]
		switch ch {
		case '1':
			fmt.Println("1")
			return "1"
		case '2':
			fmt.Println("2")
			return "2"
		case '3', 'q', 'Q', 27, 3: // 3, q, Esc, Ctrl+C
			fmt.Println("Cancel")
			return "3"
		}
	}
}

