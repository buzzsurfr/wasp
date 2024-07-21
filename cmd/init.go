/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:     "init",
	Aliases: []string{"initialize", "initialise", "create"},
	Short:   "Initialize a new wasp configuration",
	Long: `Initialize (wasp init) will create a new wasp configuration file and start
discovery of AWS SSO sessions.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Read AWS config file
		awsConfig, err := ini.Load(config.DefaultSharedConfigFilename())
		if err != nil {
			panic(err)
		}

		// Parse session names and profile names
		var ssoSessionNames []string
		var profileNames []string

		var rows []table.Row
		colWidths := make(map[string]int)
		colWidths["Name"] = 0
		colWidths["Start URL"] = 0
		colWidths["Region"] = 0

		for _, section := range awsConfig.Sections() {
			str := strings.Split(section.Name(), " ")
			var sectionType, sectionName string
			if len(str) < 2 {
				// AWS Config files don't allow unsectioned keys
				if str[0] == ini.DefaultSection {
					continue
				}
				sectionType = "profile"
				sectionName = str[0] // Should be "default"
			} else {
				sectionType = str[0]
				sectionName = str[1]
			}
			// fmt.Printf("%s: %s\n", sectionType, sectionName)
			switch sectionType {
			case "profile":
				profileNames = append(profileNames, sectionName)
			case "sso-session":
				ssoSessionNames = append(ssoSessionNames, sectionName)
				rows = append(rows, table.Row{sectionName, section.Key("sso_start_url").Value(), section.Key("sso_region").Value()})
				colWidths["Name"] = max(colWidths["Name"], len(sectionName))
				colWidths["Start URL"] = max(colWidths["Start URL"], len(section.Key("sso_start_url").Value()))
				colWidths["Region"] = max(colWidths["Region"], len(section.Key("sso_region").Value()))
			}

			// for _, key := range section.Keys() {
			// 	fmt.Printf("\t%s = %s\n", key.Name(), key.Value())
			// }
		}
		fmt.Printf("SSO Sessions: %v\n", ssoSessionNames)
		fmt.Printf("Profiles: %v\n", profileNames)

		// Create bubbles table columns based on colWidths
		columns := []table.Column{
			{Title: "Name", Width: colWidths["Name"]},
			{Title: "Start URL", Width: colWidths["Start URL"]},
			{Title: "Region", Width: colWidths["Region"]},
		}

		// Create bubbles table
		t := table.New(
			table.WithColumns(showFirstColumnOnly(columns)),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(7),
		)

		s := table.DefaultStyles()
		s.Header = s.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			BorderBottom(true).
			Bold(false)
		s.Selected = s.Selected.
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(false)
		t.SetStyles(s)

		// Choose a sso session
		p := tea.NewProgram(model{
			columns: columns,
			names:   showFirstColumnOnly(columns),
			choices: t,
		})
		m, err := p.Run()
		if err != nil {
			fmt.Println("Error running program:", err)
			os.Exit(1)
		}

		// Assert the final tea.Model to our local model and print the choice.
		if m, ok := m.(model); ok && m.choice != "" {
			fmt.Printf("\n---\nYou chose %s!\n", m.choice)
		}

	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.HiddenBorder()).
	BorderForeground(lipgloss.Color("240"))

type model struct {
	choice  string
	columns []table.Column
	names   []table.Column
	choices table.Model
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case tea.KeyRight.String():
			m.choices.SetColumns(m.columns)
		case tea.KeyLeft.String():
			m.choices.SetColumns(m.names)
		case "enter":
			m.choice = m.choices.SelectedRow()[0]
			return m, tea.Quit
		}
	}
	m.choices, cmd = m.choices.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return "\n\n" + baseStyle.Render(m.choices.View()) + "\n"
}

func (m model) ExpandTable() tea.Msg {
	var t tea.Msg
	m.choices.SetColumns(m.columns)
	return t
}

func showFirstColumnOnly(columns []table.Column) []table.Column {
	ret := []table.Column{}
	for i, col := range columns {
		if i > 0 {
			col.Width = -1
		}
		ret = append(ret, col)
	}
	return ret
}
