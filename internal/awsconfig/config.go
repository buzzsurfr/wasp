package awsconfig

import (
	"fmt"
	"strings"

	"gopkg.in/ini.v1"
)

// ConfigFile represents the AWS config file structure
type ConfigFile struct {
	file        string
	iniFile     *ini.File
	Profiles    *Profiles
	Services    *Services
	SSOSessions *SSOSessions
}

type Section interface {
	Type() string
	colWidths() map[string]int
}

func (cf ConfigFile) GetProfile(name string) (*Profile, error) {
	profile := cf.Profiles.m[name]
	if profile == nil {
		return nil, fmt.Errorf("profile %s not found", name)
	}
	return profile, nil
}

func (cf ConfigFile) HasProfile(name string) bool {
	profile, _ := cf.GetProfile(name)
	return profile != nil
}

func (cf ConfigFile) Profile(name string) *Profile {
	_, err := cf.GetProfile(name)
	if err != nil {
		// Create if it doesn't exist
		cf.Profiles.m[name] = NewProfile(name)
	}
	return cf.Profiles.m[name]
}

func (cf ConfigFile) GetService(name string) (*Service, error) {
	service := cf.Services.m[name]
	if service == nil {
		return nil, fmt.Errorf("service %s not found", name)
	}
	return service, nil
}

func (cf ConfigFile) HasService(name string) bool {
	service, _ := cf.GetService(name)
	return service != nil
}

func (cf ConfigFile) Service(name string) *Service {
	_, err := cf.GetService(name)
	if err != nil {
		// Create if it doesn't exist
		cf.Services.m[name] = NewService(name)
	}
	return cf.Services.m[name]
}

func (cf ConfigFile) GetSSOSession(name string) (*SSOSession, error) {
	session := cf.SSOSessions.m[name]
	if session == nil {
		return nil, fmt.Errorf("SSO session %s not found", name)
	}
	return session, nil
}

func (cf ConfigFile) HasSSOSession(name string) bool {
	session, _ := cf.GetSSOSession(name)
	return session != nil
}

func (cf ConfigFile) SSOSession(name string) *SSOSession {
	_, err := cf.GetSSOSession(name)
	if err != nil {
		// Create if it doesn't exist
		cf.SSOSessions.m[name] = NewSSOSession(name)
	}
	return cf.SSOSessions.m[name]
}

func (cf *ConfigFile) Load(source string) error {
	// Load config file
	cf.file = source
	configFile, err := ini.LoadSources(
		ini.LoadOptions{
			AllowShadows: true,
		},
		source,
	)
	if err != nil {
		return err
	}
	cf.iniFile = configFile

	// Parse sections into profiles, services, and SSO sessions
	for _, section := range configFile.Sections() {
		sectionType, sectionName := splitSectionText(section.Name())

		switch sectionType {
		case "unused":
			continue
		case "profile":
			err := cf.Profiles.NewFromSection(sectionName, section)
			if err != nil {
				return err
			}
		case "service":
			err := cf.Services.NewFromSection(sectionName, section)
			if err != nil {
				return err
			}
		case "sso-session":
			err := cf.SSOSessions.NewFromSection(sectionName, section)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (cf *ConfigFile) Update() error {
	// Merge profiles back into the ini file
	for _, profile := range cf.Profiles.m {
		var section *ini.Section
		// Duplicate from default profile if it doesn't exist
		if !cf.iniFile.HasSection("profile " + profile.Name) {
			defaultSection, err := cf.iniFile.GetSection("default")
			if err != nil {
				// No defaults found. This should only happen if the config file is empty of malformed.
				return err
			}
			section, _ = cf.iniFile.NewSection("profile " + profile.Name)
			for key, value := range defaultSection.KeysHash() {
				section.Key(key).SetValue(value)
			}
		} else {
			section, _ = cf.iniFile.GetSection("profile " + profile.Name)
		}
		section.Key("sso_session").SetValue(profile.SSOSession)
		section.Key("sso_account_id").SetValue(profile.AccountID)
		section.Key("sso_role_name").SetValue(profile.RoleName)
	}

	// Unimplemented: not fully implemented since not handling services
	// Merge services back into the ini file
	// for _, service := range cf.Services.m {
	// 	var section *ini.Section
	// 	section, _ := cf.iniFile.Section("service " + service.Name)
	// }

	// Merge SSO sessions back into the ini file
	for _, session := range cf.SSOSessions.m {
		section := cf.iniFile.Section("sso-session " + session.Name)
		section.Key("sso_start_url").SetValue(session.StartURL)
		section.Key("sso_region").SetValue(session.Region)
		section.Key("sso_account_id").SetValue(session.AccountID)
		section.Key("sso_role_name").SetValue(session.RoleName)
		section.Key("sso_registration_scopes").SetValue(strings.Join(session.RegistrationScopes, ","))
	}

	// Write the ini file back to disk
	err := cf.iniFile.SaveTo(cf.file)
	if err != nil {
		return err
	}

	return nil
}

// splitSectionText splits the section name into section type and section name.
// It takes a section string as input and returns the section type and section name as strings.
// If the section name is unsectioned, it returns "unused" as the section type and the default section name.
// If the section name is in the format "sectionType sectionName", it returns the section type and section name accordingly.
func splitSectionText(sectionText string) (sectionType, sectionName string) {
	sectionParts := strings.Split(sectionText, " ")
	if len(sectionParts) < 2 {
		// AWS Config files don't allow unsectioned keys
		if sectionParts[0] == ini.DefaultSection {
			return "unused", ini.DefaultSection
		} else {
			sectionType = "profile"
			sectionName = sectionParts[0] // Should be "default"
		}
	} else {
		sectionType = sectionParts[0]
		sectionName = strings.Join(sectionParts[1:], ",")
	}
	return sectionType, sectionName
}

// LoadConfig loads the AWS config file and parses the SSO sessions and profiles
func NewFromConfig(source string) (*ConfigFile, error) {
	ret := ConfigFile{
		file:        source,
		Profiles:    newProfiles(),
		Services:    newServices(),
		SSOSessions: newSSOSessions(),
	}

	err := ret.Load(source)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}
