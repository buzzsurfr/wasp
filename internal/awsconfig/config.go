package awsconfig

import (
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

func (cf ConfigFile) Profile(name string) *Profile {
	return cf.Profiles.Name(name)
}

func (cf ConfigFile) Service(name string) *Service {
	return cf.Services.Name(name)
}

func (cf ConfigFile) SSOSession(name string) *SSOSession {
	return cf.SSOSessions.Name(name)
}

func (cf ConfigFile) Load(source string) error {
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
