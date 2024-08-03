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
					fmt.Printf("Unauthorized. Please run `aws sso login --sso-session %s` to refresh your session.\n", session.Name)
					os.Exit(1)
				} else {
					panic(err)
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
