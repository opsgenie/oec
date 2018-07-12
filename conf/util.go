package conf

import (
	"os/user"
	"encoding/json"
	"path/filepath"
	"github.com/pkg/errors"
	"io/ioutil"
	"gopkg.in/yaml.v2"
)

func parseConfiguration(path string) (map[string]interface{}, error) {
	extension := filepath.Ext(path)

	if extension == ".json" {
		file, err := ioutil.ReadFile(path)

		if err != nil {
			return nil, err
		}

		return parseJsonConfiguration(file)
	} else if extension == ".yml" || extension == ".yaml" {
		file, err := ioutil.ReadFile(path)

		if err != nil {
			return nil, err
		}

		return parseYamlConfiguration(file)
	} else {
		return nil, errors.New("Unknown configuration file extension [" + extension + "]. Only json and yml" +
			" types are allowed.")
	}
}

func parseJsonConfiguration(content []byte) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal(content, &result)

	return result, err
}

func parseYamlConfiguration(content []byte) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := yaml.Unmarshal(content, &result)

	return result, err
}

func getHomePath() (string, error) {
	currentUser, err := user.Current()

	if err != nil {
		return "", err
	}

	return currentUser.HomeDir, nil
}

func cloneMap(original map[string]interface{}) (map[string]interface{}, error) {
	if original == nil {
		return nil, nil
	}

	originalJson, err := json.Marshal(original)

	if err != nil {
		return nil, err
	}

	copiedMap := make(map[string]interface{})

	err = json.Unmarshal(originalJson, &copiedMap)

	if err != nil {
		return nil, err
	}

	return copiedMap, nil
}
