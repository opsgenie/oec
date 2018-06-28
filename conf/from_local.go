package conf

import "os"

func readConfigurationFromLocal() (map[string]string, error) {
	homePath, err := getHomePath()

	if err != nil {
		return nil, err
	}

	return parseConfiguration(homePath + string(os.PathSeparator) + ".opsgenie" + string(os.PathSeparator) +
		"marid.conf")
}
