package cli

func getConfigString(arguments map[string]interface{}, key string) string {
	configPath, hasConfigPath := arguments[key]
	if hasConfigPath {
		value, ok := configPath.(string)
		if ok {
			return value
		}
	}
	return ""
}

func getConfigStringArray(arguments map[string]interface{}, key string) []string {
	configPath, hasConfigPath := arguments[key]
	if hasConfigPath {
		value, ok := configPath.([]string)
		if ok {
			return value
		}
	}
	return []string{}
}

func getConfigBool(arguments map[string]interface{}, key string) bool {
	configPath, hasConfigPath := arguments[key]
	if hasConfigPath {
		value, ok := configPath.(bool)
		if ok {
			return value
		}
	}
	return false
}

func getConfigInt(arguments map[string]interface{}, key string) int {
	configPath, hasConfigPath := arguments[key]
	if hasConfigPath {
		value, ok := configPath.(int)
		if ok {
			return value
		}
	}
	return 0
}

func GetCommand(arguments map[string]interface{}) string {
	if getConfigBool(arguments, "render") {
		return "render"
	}
	return "daemon"
}
