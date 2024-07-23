package awsconfig

import "gopkg.in/ini.v1"

// Unimplemented past name
type Service struct {
	Name string
}

func newService(name string) *Service {
	return &Service{
		Name: name,
	}
}

func (s *Service) colWidths() map[string]int {
	return map[string]int{
		"name": len(s.Name),
	}
}

func (s *Service) Type() string {
	return "service"
}

type Services struct {
	m         map[string]*Service
	colWidths map[string]int
}

func newServices() *Services {
	return &Services{
		m:         make(map[string]*Service),
		colWidths: make(map[string]int),
	}
}

func (s *Services) NewFromSection(name string, section *ini.Section) error {
	service := newService(name)
	err := section.MapTo(service)
	if err != nil {
		return err
	}
	s.m[name] = service

	// Update column widths for services
	for k, v := range service.colWidths() {
		if s.colWidths[k] < v {
			s.colWidths[k] = v
		}
	}

	return nil
}

func (s *Services) Name(name string) *Service {
	return s.m[name]
}

func (s *Services) List() []*Service {
	var list []*Service
	for _, v := range s.m {
		list = append(list, v)
	}
	return list
}

func (s *Services) Map() map[string]*Service {
	return s.m
}
