package config

import (
	"errors"
	"math"
	"strconv"
)

func parseConfigGroup(label string, data map[string]interface{}) (map[string]interface{}, error) {
	group, hasGroup := data[label]
	if !hasGroup {
		return nil, appConfigError{"Missing " + label + " config"}
	}
	groupValue, ok := group.(map[string]interface{})
	if !ok {
		return nil, appConfigError{"Invalid " + label + " config"}
	}
	return groupValue, nil
}

func parseString(group, key string, data map[string]interface{}) (string, error) {
	keyValue, hasKey := data[key]
	if !hasKey {
		return "", appConfigError{"Invalid " + group + " config: " + key + " attribute missing"}
	}
	keyStringValue, ok := keyValue.(string)
	if !ok {
		return "", appConfigError{"Invalid " + group + " config: " + key + " attribute not a string"}
	}
	return keyStringValue, nil
}

func parseBool(group, key string, data map[string]interface{}) (bool, error) {
	keyValue, hasKey := data[key]
	if !hasKey {
		return false, appConfigError{"Invalid " + group + " config: " + key + " attribute missing"}
	}
	keyStringValue, ok := keyValue.(bool)
	if !ok {
		return false, appConfigError{"Invalid " + group + " config: " + key + " attribute not a bool"}
	}
	return keyStringValue, nil
}

func parseInt(group, key string, data map[string]interface{}) (int, error) {
	keyValue, hasKey := data[key]
	if !hasKey {
		return 0, appConfigError{"Invalid " + group + " config: " + key + " attribute missing"}
	}
	keyStringValue, ok := keyValue.(float64)
	if !ok {
		return 0, appConfigError{"Invalid " + group + " config: " + key + " attribute not an int"}
	}
	return int(keyStringValue), nil
}

func parseStringArray(group, key string, data map[string]interface{}) ([]string, error) {
	keyValue, hasKey := data[key]
	if !hasKey {
		return nil, appConfigError{"Invalid " + group + " config: " + key + " attribute missing"}
	}
	keyStringValue, ok := keyValue.([]interface{})
	if !ok {
		return nil, appConfigError{"Invalid " + group + " config: " + key + " attribute not a list of strings"}
	}
	results := make([]string, 0, 0)
	for _, value := range keyStringValue {
		valueValue, ok := value.(string)
		if ok {
			results = append(results, valueValue)
		}
	}
	return results, nil
}

func getFloat(unk interface{}) (float64, error) {
	if v_flt, ok := unk.(float64); ok {
		return v_flt, nil
	} else if v_int, ok := unk.(int); ok {
		return float64(v_int), nil
	} else if v_int, ok := unk.(int16); ok {
		return float64(v_int), nil
	} else if v_str, ok := unk.(string); ok {
		v_flt, err := strconv.ParseFloat(v_str, 64)
		if err == nil {
			return v_flt, nil
		}
		return math.NaN(), err
	} else if unk == nil {
		return math.NaN(), errors.New("getFloat: unknown value is nil")
	} else {
		return math.NaN(), errors.New("getFloat: unknown value is of incompatible type")
	}
}

func getStringArray(unknown interface{}) ([]string, error) {
	unknownData, ok := unknown.([]interface{})
	if ok {
		results := make([]string, 0, 0)
		for _, value := range unknownData {
			valueValue, ok := value.(string)
			if ok {
				results = append(results, valueValue)
			}
		}
		return results, nil
	}
	return nil, appConfigError{"Data is not an array."}
}
