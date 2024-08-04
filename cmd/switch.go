/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	awsconfig "github.com/buzzsurfr/wasp/internal/awsconfig"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/spf13/cobra"
)

// switchCmd represents the switch command
var switchCmd = &cobra.Command{
	Use:     "switch",
	Aliases: []string{"sw", "swap"},
	Short:   "Change your current AWS profile",
	Long: `Switch will change your current AWS profile. This will change the
current profile in the AWS_PROFILE and AWS_DEFAULT_PROFILE environment
variables.`,
	Run: func(cmd *cobra.Command, args []string) {

		// Load AWS config file
		cf, err := awsconfig.NewFromConfig(config.DefaultSharedConfigFilename())
		if err != nil {
			panic(err)
		}

		// Create Bubbles table for profiles
		t := cf.Profiles.TableModel(10)
		t.SetColumns(cf.Profiles.TableColumns())
		t.Focus()
		t.SetStyles(tableStyle)

		// Choose a profile
		p := tea.NewProgram(newProfileModel(t, cf.Profiles.TableColumns()))
		m, err := p.Run()
		if err != nil {
			fmt.Println("Error running program:", err)
			os.Exit(1)
		}
		if m, ok := m.(profileModel); ok && cf.HasProfile(m.profileName) {
			fmt.Printf("export AWS_PROFILE=%s\n", m.profileName)
		} else {
			// fmt.Println("Profile not found")
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(switchCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// switchCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// switchCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

type profileModel struct {
	profileName string
	table       table.Model
	columns     []table.Column
	selected    table.Row
	quitting    bool
}

func newProfileModel(t table.Model, columns []table.Column) profileModel {
	return profileModel{
		table:    t,
		columns:  columns,
		quitting: false,
	}
}

func (m profileModel) Init() tea.Cmd {
	return nil
}

func (m profileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			m.selected = m.table.SelectedRow()
			m.profileName = m.selected[0]
			m.quitting = true
			return m, tea.Quit
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m profileModel) View() string {
	if m.quitting {
		return ""
	}
	return m.table.View() + "\n\n\n"
}
