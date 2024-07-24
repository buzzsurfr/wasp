/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/ssocreds"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/sso/types"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	awsconfig "github.com/buzzsurfr/wasp/internal/awsconfig"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:     "init",
	Aliases: []string{"initialize", "initialise", "create"},
	Short:   "Initialize a new wasp configuration",
	Long: `Initialize (wasp init) will create a new wasp configuration file and start
discovery of AWS SSO sessions.`,
	Run: func(cmd *cobra.Command, args []string) {

		// Load AWS config file
		cf, err := awsconfig.NewFromConfig(config.DefaultSharedConfigFilename())
		if err != nil {
			panic(err)
		}

		// Create Bubbles table for SSO sessions
		t := cf.SSOSessions.TableModel(10)
		t.SetColumns(showFirstColumnOnly(cf.SSOSessions.TableColumns()))
		t.Focus()
		t.SetStyles(tableStyle)

		// Choose a sso session
		p := tea.NewProgram(newModel(t, cf.SSOSessions.TableColumns()))
		m, err := p.Run()
		if err != nil {
			fmt.Println("Error running program:", err)
			os.Exit(1)
		}

		// Assert the final tea.Model to our local model and print the choice.
		var ssoSession, ssoRegion string
		if m, ok := m.(model); ok && m.choice != "" {
			// fmt.Printf("\n---\nYou chose %s!\n", m.choice)
			ssoSession = m.choice
			ssoRegion = m.region
		} else {
			os.Exit(1)
		}

		cfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			panic(err)
		}
		cfg.Region = ssoRegion

		ssoClient := sso.NewFromConfig(cfg)
		ssoOidcClient := ssooidc.NewFromConfig(cfg)
		tokenPath, err := ssocreds.StandardCachedTokenFilepath(ssoSession)
		if err != nil {
			panic(err)
		}

		// Read JSON from tokenPath and parse sso_start_url
		tokenFile, err := os.ReadFile(tokenPath)
		if err != nil {
			panic(err)
		}

		var ssoData SSOData
		err = json.Unmarshal(tokenFile, &ssoData)
		if err != nil {
			panic(err)
		}

		ssoTokenProvider := ssocreds.NewSSOTokenProvider(ssoOidcClient, tokenPath)

		var provider aws.CredentialsProvider
		provider = ssocreds.New(ssoClient, "123456789012", "no-role", ssoData.StartURL, func(options *ssocreds.Options) {
			options.SSOTokenProvider = ssoTokenProvider
		})

		// Wrap the provider with aws.CredentialsCache to cache the credentials until their expire time
		provider = aws.NewCredentialsCache(provider)

		// List associated AWS accounts and roles
		listAccounts, err := ssoClient.ListAccounts(context.TODO(), &sso.ListAccountsInput{
			AccessToken: &ssoData.AccessToken,
		})
		if err != nil {
			var aerr *types.UnauthorizedException
			if errors.As(err, &aerr) {
				fmt.Printf("Unauthorized. Please run `aws sso login --sso-session %s` to refresh your session.\n", ssoSession)
				os.Exit(1)
			} else {
				panic(err)
			}
		}

		var accountRows []table.Row
		accountColWidths := make(map[string]int)
		accountColWidths["Name"] = 0
		accountColWidths["Email Address"] = 0
		accountColWidths["ID"] = 0
		accountColWidths["Role"] = 0

		for _, account := range listAccounts.AccountList {
			// Create account table rows
			accountColWidths["Name"] = max(accountColWidths["Name"], len(aws.ToString(account.AccountName)))
			accountColWidths["Email Address"] = max(accountColWidths["Email Address"], len(aws.ToString(account.EmailAddress)))
			accountColWidths["ID"] = max(accountColWidths["ID"], len(aws.ToString(account.AccountId)))

			// Account Roles
			listAccountRoles, err := ssoClient.ListAccountRoles(context.TODO(), &sso.ListAccountRolesInput{
				AccessToken: &ssoData.AccessToken,
				AccountId:   account.AccountId,
			})
			if err != nil {
				panic(err)
			}
			for _, role := range listAccountRoles.RoleList {
				accountRows = append(accountRows, table.Row{aws.ToString(account.AccountName), aws.ToString(account.EmailAddress), aws.ToString(account.AccountId), aws.ToString(role.RoleName)})
				accountColWidths["Role"] = max(accountColWidths["Role"], len(aws.ToString(role.RoleName)))
				// fmt.Printf("  %s\n", *role.RoleName)
			}
		}
		// Create bubbles table ssoSessionColumns based on colWidths
		accountColumns := []table.Column{
			{Title: "Name", Width: accountColWidths["Name"]},
			{Title: "Email Address", Width: accountColWidths["Email Address"]},
			{Title: "ID", Width: accountColWidths["ID"]},
			{Title: "Role", Width: accountColWidths["Role"]},
		}

		// Create bubbles table
		at := table.New(
			table.WithColumns(accountColumns),
			table.WithRows(accountRows),
			table.WithFocused(true),
			table.WithHeight(min(len(accountRows), 10)),
		)

		at.SetStyles(tableStyle)

		// Choose an account
		ap := tea.NewProgram(newAccountsModel(at, accountColumns))
		am, err := ap.Run()
		if err != nil {
			fmt.Println("Error running program:", err)
			os.Exit(1)
		}

		// Assert the final tea.Model to our local model and print the choice.
		var profile *awsconfig.Profile
		if am, ok := am.(accountsModel); ok && am.accountName != "" {
			profile_name := fmt.Sprintf("%s_%s", am.accountName, am.roleName)
			profile = cf.Profile(profile_name)
			profile.Name = profile_name
			profile.SSOSession = ssoSession
			profile.AccountID = am.accountId
			profile.RoleName = am.roleName
		} else {
			os.Exit(1)
		}
		fmt.Printf("[profile %s]\nsso_session = %s\nsso_account_id = %s\nsso_role_name = %s\n", profile.Name, profile.SSOSession, profile.AccountID, profile.RoleName)

		err = cf.Update()
		if err != nil {
			panic(err)
		}
	},
}

var baseStyle lipgloss.Style // baseStyle is the default style for any text
var tableStyle table.Styles  // tableStyle is the default style for any table

func init() {
	rootCmd.AddCommand(initCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

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

}

type model struct {
	choice  string
	region  string
	columns []table.Column
	names   []table.Column
	choices table.Model
	help    help.Model
	keyMap  keyMap
}

type keyMap struct {
	LineUp       key.Binding
	LineDown     key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	GotoTop      key.Binding
	GotoBottom   key.Binding
	Expand       key.Binding
	Collapse     key.Binding
}

// ShortHelp implements the KeyMap interface.
func (km keyMap) ShortHelp() []key.Binding {
	return []key.Binding{km.LineUp, km.LineDown, km.Expand, km.Collapse}
}

// FullHelp implements the KeyMap interface.
func (km keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{km.LineUp, km.LineDown, km.GotoTop, km.GotoBottom},
		{km.PageUp, km.PageDown, km.HalfPageUp, km.HalfPageDown},
	}
}

func newModel(choices table.Model, columns []table.Column) model {
	return model{
		columns: columns,
		names:   showFirstColumnOnly(columns),
		choices: choices,
		help:    help.New(),
		keyMap: keyMap{
			LineUp: key.NewBinding(
				key.WithKeys("up", "k"),
				key.WithHelp("↑/k", "up"),
			),
			LineDown: key.NewBinding(
				key.WithKeys("down", "j"),
				key.WithHelp("↓/j", "down"),
			),
			PageUp: key.NewBinding(
				key.WithKeys("b", "pgup"),
				key.WithHelp("b/pgup", "page up"),
			),
			PageDown: key.NewBinding(
				key.WithKeys("f", "pgdown", " "),
				key.WithHelp("f/pgdn", "page down"),
			),
			HalfPageUp: key.NewBinding(
				key.WithKeys("u", "ctrl+u"),
				key.WithHelp("u", "½ page up"),
			),
			HalfPageDown: key.NewBinding(
				key.WithKeys("d", "ctrl+d"),
				key.WithHelp("d", "½ page down"),
			),
			GotoTop: key.NewBinding(
				key.WithKeys("home", "g"),
				key.WithHelp("g/home", "go to start"),
			),
			GotoBottom: key.NewBinding(
				key.WithKeys("end", "G"),
				key.WithHelp("G/end", "go to end"),
			),
			Expand: key.NewBinding(
				key.WithKeys("right", "l"),
				key.WithHelp("→/l", "expand"),
			),
			Collapse: key.NewBinding(
				key.WithKeys("left", "h"),
				key.WithHelp("←/h", "collapse"),
			),
		},
	}
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
		case tea.KeyRight.String(), "l":
			m.choices.SetColumns(m.columns)
		case tea.KeyLeft.String(), "h":
			m.choices.SetColumns(m.names)
		case "enter":
			m.choice = m.choices.SelectedRow()[0]
			m.region = m.choices.SelectedRow()[2]
			return m, tea.Quit
		}
	}
	m.choices, cmd = m.choices.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return "\nSelect a SSO session:\n" + baseStyle.Render(m.choices.View()) + "\n" + m.help.View(m.keyMap)
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

type SSOData struct {
	AccessToken      string `json:"accessToken"`
	ExpiresAt        string `json:"expiresAt"`
	Region           string `json:"region"`
	StartURL         string `json:"startUrl"`
	SSORegion        string `json:"ssoRegion"`
	AccountID        string `json:"accountId"`
	RoleName         string `json:"roleName"`
	IdentityProvider string `json:"identityProvider"`
}

type accountsModel struct {
	accountName  string
	accountId    string
	emailAddress string
	roleName     string
	columns      []table.Column
	names        []table.Column
	choices      table.Model
	help         help.Model
	keyMap       keyMap
}

func newAccountsModel(choices table.Model, columns []table.Column) accountsModel {
	return accountsModel{
		columns: columns,
		names:   showFirstColumnOnly(columns),
		choices: choices,
		help:    help.New(),
		keyMap: keyMap{
			LineUp: key.NewBinding(
				key.WithKeys("up", "k"),
				key.WithHelp("↑/k", "up"),
			),
			LineDown: key.NewBinding(
				key.WithKeys("down", "j"),
				key.WithHelp("↓/j", "down"),
			),
			PageUp: key.NewBinding(
				key.WithKeys("b", "pgup"),
				key.WithHelp("b/pgup", "page up"),
			),
			PageDown: key.NewBinding(
				key.WithKeys("f", "pgdown", " "),
				key.WithHelp("f/pgdn", "page down"),
			),
			HalfPageUp: key.NewBinding(
				key.WithKeys("u", "ctrl+u"),
				key.WithHelp("u", "½ page up"),
			),
			HalfPageDown: key.NewBinding(
				key.WithKeys("d", "ctrl+d"),
				key.WithHelp("d", "½ page down"),
			),
			GotoTop: key.NewBinding(
				key.WithKeys("home", "g"),
				key.WithHelp("g/home", "go to start"),
			),
			GotoBottom: key.NewBinding(
				key.WithKeys("end", "G"),
				key.WithHelp("G/end", "go to end"),
			),
			Expand: key.NewBinding(
				key.WithKeys("right", "l"),
				key.WithHelp("→/l", "expand"),
			),
			Collapse: key.NewBinding(
				key.WithKeys("left", "h"),
				key.WithHelp("←/h", "collapse"),
			),
		},
	}
}

func (m accountsModel) Init() tea.Cmd {
	return nil
}

func (m accountsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case tea.KeyRight.String(), "l":
			m.choices.SetColumns(m.columns)
		case tea.KeyLeft.String(), "h":
			m.choices.SetColumns(m.names)
		case "enter":
			m.accountName = m.choices.SelectedRow()[0]
			m.emailAddress = m.choices.SelectedRow()[1]
			m.accountId = m.choices.SelectedRow()[2]
			m.roleName = m.choices.SelectedRow()[3]

			return m, tea.Quit
		}
	}
	m.choices, cmd = m.choices.Update(msg)
	return m, cmd
}

func (m accountsModel) View() string {
	return "\nAWS accounts in session:\n" + baseStyle.Render(m.choices.View()) + "\n" + m.help.View(m.keyMap)
}
