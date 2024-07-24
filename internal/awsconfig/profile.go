package awsconfig

import (
	"github.com/charmbracelet/bubbles/table"
	"gopkg.in/ini.v1"
)

type Profile struct {
	Name        string      `ini:"-"`
	Session     *SSOSession `ini:"-"`
	SSOSession  string      `ini:"sso_session"`
	AccountName string      `ini:"-"`
	AccountID   string      `ini:"sso_account_id"`
	RoleName    string      `ini:"sso_role_name"`
}

func NewProfile(name string) *Profile {
	return &Profile{
		Name:    name,
		Session: nil,
	}
}

func (p *Profile) colWidths() map[string]int {
	return map[string]int{
		"account_name": len(p.AccountName),
		"sso_session":  len(p.SSOSession),
		"account_id":   len(p.AccountID),
		"role_name":    len(p.RoleName),
	}
}

func (p *Profile) Type() string {
	return "profile"
}

type Profiles struct {
	m         map[string]*Profile
	colWidths map[string]int
}

func newProfiles() *Profiles {
	return &Profiles{
		m:         make(map[string]*Profile),
		colWidths: make(map[string]int),
	}
}

func (p *Profiles) NewFromSection(name string, section *ini.Section) error {
	profile := NewProfile(name)
	err := section.MapTo(profile)
	if err != nil {
		return err
	}
	p.m[name] = profile

	// Update column widths for profiles
	for k, v := range profile.colWidths() {
		if p.colWidths[k] < v {
			p.colWidths[k] = v
		}
	}

	return nil
}

func (p *Profiles) Name(name string) *Profile {
	return p.m[name]
}

func (p *Profiles) List() []*Profile {
	var list []*Profile
	for _, v := range p.m {
		list = append(list, v)
	}
	return list
}

func (p *Profiles) Map() map[string]*Profile {
	return p.m
}

func (p *Profiles) TableModel(maxRows int) table.Model {
	var rows []table.Row
	for _, profile := range p.m {
		rows = append(rows, table.Row{
			profile.AccountName,
			profile.SSOSession,
			profile.AccountID,
			profile.RoleName,
		})
	}

	return table.New(
		table.WithColumns(p.TableColumns()),
		table.WithRows(rows),
		table.WithHeight(min(len(rows), maxRows)),
	)
}

func (p *Profiles) TableColumns() []table.Column {
	return []table.Column{
		{Title: "Account Name", Width: p.colWidths["account_name"]},
		{Title: "SSO Session", Width: p.colWidths["sso_session"]},
		{Title: "Account ID", Width: p.colWidths["account_id"]},
		{Title: "Role Name", Width: p.colWidths["role_name"]},
	}
}
