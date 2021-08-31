package main

import (
	"regexp"
	"strings"
)

func contains(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func unmarshalledYAMLToMarshallableJSON(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[k.(string)] = unmarshalledYAMLToMarshallableJSON(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = unmarshalledYAMLToMarshallableJSON(v)
		}
	}
	return i
}

func getProjectGroupFromGitURL(gitURL string, projectName string) string {
	pattern := regexp.MustCompile(`((git@[\w\.]+)):(?P<projectPath>[\w\.@\:/\-~]+)(\.git)(/)?`)
	if !pattern.MatchString(gitURL) {
		return ""
	}

	template := "$projectPath"
	projectPath := pattern.ReplaceAllString(gitURL, template)
	projectGroup := strings.TrimSuffix(strings.ToLower(projectPath), strings.ToLower(projectName))
	projectGroup = strings.TrimSuffix(projectGroup, "/")

	return projectGroup
}

func findDefaultBranch(branches []Branch) (Branch, bool) {
	for _, branch := range branches {
		if branch.Default {
			return branch, true
		}
	}

	return Branch{}, false
}
