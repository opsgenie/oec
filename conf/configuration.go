package conf

import (
	"github.com/pkg/errors"
	"os"
)

var Configuration map[string]interface{}
var RunbookActionMapping map[string]interface{}
var readConfigurationFromGitFunction = readConfigurationFromGit
var readConfigurationFromLocalFunction = readConfigurationFromLocal
var ParseJson = parseJson

func ReadConfFile() error {
	confSource := os.Getenv("MARIDCONFSOURCE")

	if confSource == "git" {
		privateKeyFilePath := os.Getenv("MARIDCONFREPOPRIVATEKEYPATH")
		gitUrl := os.Getenv("MARIDCONFREPOGITURL")
		maridConfPath := os.Getenv("MARIDCONFGITFILEPATH")

		gitConf, err := readConfigurationFromGitFunction(gitUrl, maridConfPath, privateKeyFilePath)

		if err == nil {
			copied, err := cloneMap(gitConf)

			if err != nil {
				return err
			}

			Configuration = copied

			if actionMappings, ok := copied["actionMappings"].(map[string]interface{}); ok {
				RunbookActionMapping = actionMappings
			} else {
				return errors.New("Action mappings configuration is not found in the configuration file.")
			}

			return nil
		} else {
			return err
		}
	} else if confSource == "local" {
		maridConfPath := os.Getenv("MARIDCONFLOCALFILEPATH")

		if len(maridConfPath) <= 0 {
			homePath, err := getHomePath()

			if err != nil {
				return err
			}

			maridConfPath = homePath + string(os.PathSeparator) + ".opsgenie" + string(os.PathSeparator) +
				"maridConfig.json"
		}

		localConf, err := readConfigurationFromLocalFunction(maridConfPath)

		if err == nil {
			copied, err := cloneMap(localConf)

			if err != nil {
				return err
			}

			Configuration = copied

			if actionMappings, ok := copied["actionMappings"].(map[string]interface{}); ok {
				RunbookActionMapping = actionMappings
			} else {
				return errors.New("Action mappings configuration is not found in the configuration file.")
			}

			return nil
		} else {
			return err
		}
	} else {
		return errors.New("Unknown configuration source [" + confSource + "].")
	}
}
