/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	awsconfig "github.com/buzzsurfr/wasp/internal/awsconfig"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

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
		baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.HiddenBorder()).
			BorderForeground(lipgloss.Color("240"))

		tableStyle = table.DefaultStyles()
		tableStyle.Header = tableStyle.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			BorderBottom(true).
			Bold(false)
		tableStyle.Selected = tableStyle.Selected.
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(false)

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
		p := tea.NewProgram(newProfileModel(t, cf.Profiles.TableColumns()), tea.WithOutput(os.Stderr))
		m, err := p.Run()
		if err != nil {
			fmt.Println("Error running program:", err)
			os.Exit(1)
		}
		if m, ok := m.(profileModel); ok && cf.HasProfile(m.profileName) {
			fmt.Printf("export AWS_PROFILE=%s\n", m.profileName)
		} else {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(switchCmd)
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
	case tea.KeyPressMsg:
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

func (m profileModel) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}
	return tea.NewView(m.table.View() + "\n\n\n")
}
