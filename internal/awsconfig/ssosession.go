package awsconfig

import (
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"gopkg.in/ini.v1"
)

type SSOSession struct {
	Name               string   `ini:"-"`
	StartURL           string   `ini:"sso_start_url"`
	Region             string   `ini:"sso_region"`
	AccountID          string   `ini:"sso_account_id,omitempty"`
	RoleName           string   `ini:"sso_role_name,omitempty"`
	RegistrationScopes []string `ini:"sso_registration_scopes,omitempty"`
}

func NewSSOSession(name string) *SSOSession {
	return &SSOSession{
		Name: name,
	}
}

func (s *SSOSession) colWidths() map[string]int {
	return map[string]int{
		"name":                    len(s.Name),
		"sso_start_url":           len(s.StartURL),
		"sso_region":              len(s.Region),
		"sso_account_id":          len(s.AccountID),
		"sso_role_name":           len(s.RoleName),
		"sso_registration_scopes": len(strings.Join(s.RegistrationScopes, ",")),
	}
}

func (s *SSOSession) Type() string {
	return "sso-session"
}

type SSOSessions struct {
	m         map[string]*SSOSession
	colWidths map[string]int
}

func newSSOSessions() *SSOSessions {
	return &SSOSessions{
		m:         make(map[string]*SSOSession),
		colWidths: make(map[string]int),
	}
}

func (s *SSOSessions) NewFromSection(name string, section *ini.Section) error {
	session := NewSSOSession(name)
	err := section.MapTo(session)
	if err != nil {
		return err
	}
	s.m[name] = session

	// Update column widths for SSO sessions
	for k, v := range session.colWidths() {
		if s.colWidths[k] < v {
			s.colWidths[k] = v
		}
	}

	return nil
}

func (s *SSOSessions) Name(name string) *SSOSession {
	return s.m[name]
}

func (s *SSOSessions) List() []*SSOSession {
	var list []*SSOSession
	for _, v := range s.m {
		list = append(list, v)
	}
	return list
}

func (s *SSOSessions) Map() map[string]*SSOSession {
	return s.m
}

func (s *SSOSessions) TableModel(maxRows int) table.Model {
	var rows []table.Row
	for _, session := range s.m {
		rows = append(rows, table.Row{
			session.Name,
			session.StartURL,
			session.Region,
		})
	}

	return table.New(
		table.WithColumns(s.TableColumns()),
		table.WithRows(rows),
		table.WithHeight(min(len(rows), maxRows)),
	)
}

func (s *SSOSessions) TableColumns() []table.Column {
	return []table.Column{
		{Title: "Name", Width: s.colWidths["name"]},
		{Title: "Start URL", Width: s.colWidths["sso_start_url"]},
		{Title: "Region", Width: s.colWidths["sso_region"]},
	}
}
