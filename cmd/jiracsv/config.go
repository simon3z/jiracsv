package main

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// SearchProfile represents a search profile
type SearchProfile struct {
	ID         string
	JQL        string
	Components struct {
		Include []string
		Exclude []string
	}
}

// Configuration represents a jira instance with multiple search profiles
type Configuration struct {
	Instance struct {
		URL string
	}
	Profiles []*SearchProfile
}

// ReadConfigFile reads a configuration file from the specified path
func ReadConfigFile(path string) (*Configuration, error) {
	f, err := ioutil.ReadFile(path)

	if err != nil {
		return nil, err
	}

	c := &Configuration{}

	err = yaml.Unmarshal(f, c)

	if err != nil {
		return nil, err
	}

	return c, nil
}

// FindProfile finds the profile with the specified ID
func (c *Configuration) FindProfile(ID string) *SearchProfile {
	for _, p := range c.Profiles {
		if p.ID == ID {
			return p
		}
	}

	return nil
}
