package main

import (
	"encoding/json"

	"github.com/ghodss/yaml"
)

// MarshalJSON implements Marshaler interface from encoding/json.
func (p *Project) MarshalJSON() ([]byte, error) {
	p.ParsedBuildDefinition = p.parseBuildDefinition()
	type antiInfiniteLoop Project
	return json.Marshal((*antiInfiniteLoop)(p))
}

func (p *Project) parseBuildDefinition() interface{} {
	if p.BuildDefinition != "" {
		var t interface{}
		err := yaml.Unmarshal([]byte(p.BuildDefinition), &t)
		if err != nil {
			log.Error().
				WithError(err).
				WithUint("project", p.ProjectID).
				Message("Failed to parse build-definition.")
			return nil
		}
		return unmarshalledYAMLToMarshallableJSON(t)
	}
	return nil
}

// MarshalJSON implements Marshaler interface from encoding/json.
func (b *Build) MarshalJSON() ([]byte, error) {
	b.Status = b.StatusID.String()
	type antiInfiniteLoop Build
	return json.Marshal((*antiInfiniteLoop)(b))
}
