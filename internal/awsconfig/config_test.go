package awsconfig

import (
	"testing"

	"gopkg.in/ini.v1"
)

func TestSplitSectionText(t *testing.T) {
	tests := []struct {
		section      string
		expectedType string
		expectedName string
	}{
		{
			section:      "sectionType sectionName",
			expectedType: "sectionType",
			expectedName: "sectionName",
		},
		{
			section:      "unsectioned",
			expectedType: "unused",
			expectedName: "default",
		},
		{
			section:      ini.DefaultSection,
			expectedType: "unused",
			expectedName: ini.DefaultSection,
		},
		// Add more test cases here
	}

	for _, test := range tests {
		sectionType, sectionName := splitSectionText(test.section)
		if sectionType != test.expectedType {
			t.Errorf("Expected section type %s, but got %s", test.expectedType, sectionType)
		}
		if sectionName != test.expectedName {
			t.Errorf("Expected section name %s, but got %s", test.expectedName, sectionName)
		}
	}
}
