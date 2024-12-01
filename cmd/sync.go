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
	"os/exec"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/ssocreds"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/sso/types"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	awsconfig "github.com/buzzsurfr/wasp/internal/awsconfig"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize AWS profiles based on SSO sessions",
	Long: `Sync will synchronize AWS profiles based on SSO sessions. This will
create or update profiles based on the SSO sessions found in the AWS config file.`,
	Run: func(cmd *cobra.Command, args []string) {

		// Load AWS config file
		cf, err := awsconfig.NewFromConfig(config.DefaultSharedConfigFilename())
		if err != nil {
			panic(err)
		}

		// For each SSO session in config file
		for _, session := range cf.SSOSessions.Map() {

			cfg, err := config.LoadDefaultConfig(context.Background())
			if err != nil {
				panic(err)
			}
			cfg.Region = session.Region

			ssoClient := sso.NewFromConfig(cfg)
			ssoOidcClient := ssooidc.NewFromConfig(cfg)
			tokenPath, err := ssocreds.StandardCachedTokenFilepath(session.Name)
			if err != nil {
				panic(err)
			}

			// Read JSON from tokenPath and parse sso_start_url
			var ssoData SSOData
			tokenFile, err := os.ReadFile(tokenPath)
			if err == nil {
				err = json.Unmarshal(tokenFile, &ssoData)
				if err != nil {
					panic(err)
				}
			} else if err != nil && !errors.Is(err, os.ErrNotExist) {
				panic(err)
			}

			ssoTokenProvider := ssocreds.NewSSOTokenProvider(ssoOidcClient, tokenPath)

			var provider aws.CredentialsProvider
			provider = ssocreds.New(ssoClient, "123456789012", "no-role", session.StartURL, func(options *ssocreds.Options) {
				options.SSOTokenProvider = ssoTokenProvider
			})

			// Wrap the provider with aws.CredentialsCache to cache the credentials until their expire time
			provider = aws.NewCredentialsCache(provider)

			// List associated AWS accounts and roles
			var listAccounts *sso.ListAccountsOutput
			for retries := 0; retries < 2; retries++ {
				listAccounts, err = ssoClient.ListAccounts(context.TODO(), &sso.ListAccountsInput{
					AccessToken: &ssoData.AccessToken,
				})
				if err != nil {
					var aerr *types.UnauthorizedException
					if errors.As(err, &aerr) {
						// fmt.Printf("Unauthorized. Please run `aws sso login --sso-session %s` to refresh your session.\n", ssoSession)
						fmt.Printf("Unauthorized. Attempting to login to %s SSO session.\n", session.Name)
						cmd := exec.Command("aws", "sso", "login", "--sso-session", session.Name)
						if err := cmd.Run(); err != nil {
							fmt.Printf("Unauthorized. Please run `aws sso login --sso-session %s` to refresh your session.\n", session.Name)
							os.Exit(1)
						} else {
							continue
						}
					} else {
						panic(err)
					}
				}
			}

			for _, account := range listAccounts.AccountList {
				// Account Roles
				listAccountRoles, err := ssoClient.ListAccountRoles(context.TODO(), &sso.ListAccountRolesInput{
					AccessToken: &ssoData.AccessToken,
					AccountId:   account.AccountId,
				})
				if err != nil {
					panic(err)
				}
				for _, role := range listAccountRoles.RoleList {
					// Update profile in AWS config file
					profile := cf.Profile(fmt.Sprintf("%s_%s", aws.ToString(account.AccountName), aws.ToString(role.RoleName)))
					profile.SSOSession = session.Name
					profile.AccountID = aws.ToString(account.AccountId)
					profile.RoleName = aws.ToString(role.RoleName)
				}
			}
		}

		err = cf.Update()
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// syncCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// syncCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

type stringMsg string

func (s stringMsg) String() string {
	return string(s)
}

type errorMsg struct {
	msg string
}

func (e errorMsg) Error() string {
	return e.msg
}

func (e errorMsg) String() string {
	return e.msg
}

type awsClient struct {
	SSOSessionName string
	SSOSession     *awsconfig.SSOSession
	SSOClient      *sso.Client
	SSOOidcClient  *ssooidc.Client
	Cfg            aws.Config
	TokenPath      string
	Status         string
	Done           bool
	Err            error
}

func (c awsClient) Login() error {
	return nil
}

type syncModel struct {
	configFile     *awsconfig.ConfigFile
	currentSession string
	clients        map[string]*awsClient
	status         string
	done           bool
	err            error
}

func (m syncModel) LoadConfigFileCmd() tea.Msg {
	cf, err := awsconfig.NewFromConfig(config.DefaultSharedConfigFilename())
	if err != nil {
		return errorMsg{msg: err.Error()}
	}
	m.configFile = cf
	return stringMsg("config file loaded")
}

func (m syncModel) LoadAwsCliClientsCmd() tea.Msg {
	// Build clients if it hasn't already done so
	for name, session := range m.configFile.SSOSessions.Map() {

		// TODO: move to function with argument
		if _, ok := m.clients[name]; ok && m.clients[name].Done && m.clients[name].Err != nil {
			var err error
			m.clients[name].Err = nil
			m.clients[name].SSOSessionName = name
			m.clients[name].SSOSession = session
			m.clients[name].Cfg, err = config.LoadDefaultConfig(context.Background())
			if err != nil {
				m.clients[name].Err = err
			}
			m.clients[name].Cfg.Region = session.Region
			m.clients[name].SSOClient = sso.NewFromConfig(m.clients[name].Cfg)
			m.clients[name].SSOOidcClient = ssooidc.NewFromConfig(m.clients[name].Cfg)
			m.clients[name].TokenPath, err = ssocreds.StandardCachedTokenFilepath(name)
			if err != nil {
				m.clients[name].Err = err
			}
			m.clients[name].Status = "Initialized"
			m.clients[name].Done = true
		}
	}
	return stringMsg("AWS CLI clients loaded")
}

func newSSOLoginModel(names []string) syncModel {
	clients := make(map[string]*awsClient)
	for _, name := range names {
		clients[name] = &awsClient{
			SSOSessionName: name,
			Done:           false,
			Err:            nil,
		}
	}
	return syncModel{
		currentSession: names[0],
		status:         "Initializing",
		clients:        clients,
		done:           false,
	}
}

func (m syncModel) Init() tea.Cmd {
	return tea.Tick(0*time.Second, func(time.Time) tea.Msg {
		return "start"
	})
}

func (m syncModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		}
	case stringMsg:
		switch msg.String() {
		case "start":
			m.status = "Loading Config File"
			return m, m.LoadConfigFileCmd
		case "config file loaded":
			m.status = "Config file loaded"
			return m, m.LoadAwsCliClientsCmd
		case "AWS CLI clients loaded":
			m.status = "AWS CLI clients loaded"

		}
	}

	return m, cmd
}

func (m syncModel) View() string {
	return m.status + "\n"
}
