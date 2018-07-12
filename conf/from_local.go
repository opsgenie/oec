package conf

func readConfigurationFromLocal(confPath string) (map[string]interface{}, error) {
	return parseConfiguration(confPath)
}
