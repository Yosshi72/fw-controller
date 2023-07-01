package fwconfig

import (
	"encoding/json"
	"fmt"
	"sort"
	"io/ioutil"
)

type Configuration struct {
	Netns      string                 `json:"netns"`
	Interfaces map[string]interface{} `json:"interfaces"`
}

func ConfigReader(configFile string) ([]string, string, []string, error) {
	// read configutation
	file, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// json decode
	var data Configuration
	err = json.Unmarshal(file, &data)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to parse the JSON: %v", err)
	}

	// get TrustZone elements
	trustZoneInterfaces, ok := data.Interfaces["trust_zone"].([]interface{})
	if !ok {
		return nil, "", nil, fmt.Errorf("trust_zone is not a valid array")
	}
	var trustZone []string
	for _, value := range trustZoneInterfaces {
		if strValue, ok := value.(string); ok {
			trustZone = append(trustZone, strValue)
		} else {
			return nil, "", nil, fmt.Errorf("trust_zone contains non-string values")
		}
	}

	// get UnTrust Zone element
	untrustZone, ok := data.Interfaces["untrust_zone"].(string)
	if !ok {
		return nil, "", nil, fmt.Errorf("untrust_zone is not a valid string")
	}

	// TODO: get MgmtAddressRange elements
	
	return trustZone, untrustZone, nil, nil
}


func ConfigWriter(containername, configFile, newUntrustIf string, newTrustIf []string) (error) {
	// read configutation
	file, err := ioutil.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	// json decode
	var data Configuration
	err = json.Unmarshal(file, &data)
	if err != nil {
		return fmt.Errorf("failed to parse the JSON: %v", err)
	}
	
	// interfacesの更新
	trustif, untrustif := newTrustIf, newUntrustIf
	err = updateZone(data.Interfaces, trustif, untrustif)
	if err != nil {
		return fmt.Errorf("failed to update zone: %v", err)
	}

	// TODO MgmtAddressRangeの追加

	// 構造体をJSON形式に変換
	newData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to convert to JSON: %v", err)
	}

	// write to configuration
	err = ioutil.WriteFile(configFile, newData, 0644)
	if err != nil {
		return  fmt.Errorf("failed to write to configuration file: %v", err)
	}


	fmt.Println("Success: Update configuration")
	return nil
}

// trust_zoneとuntrust_zoneのupdate
func updateZone(zoneMap map[string]interface{}, trustZone []string, untrustZone string) error {
	// trustzoneのみが指定された場合、trustzoneを更新する
	if len(trustZone) > 0 && untrustZone == "" {
		if _, ok := zoneMap["trust_zone"]; ok {
			zoneMap["trust_zone"] = trustZone
			return nil
		} else {
			return fmt.Errorf("trust_zone key not found in zoneMap")
		}
	}

	// untrustzoneのみが指定された場合、untrustzoneを更新する
	if len(trustZone) == 0 && untrustZone != "" {
		if _, ok := zoneMap["untrust_zone"]; ok {
			zoneMap["untrust_zone"] = untrustZone
			return nil
		} else {
			return fmt.Errorf("untrust_zone key not found in zoneMap")
		}
	}

	// trustzoneとuntrustzoneの両方が指定された場合、両方を更新する
	if len(trustZone) > 0 && untrustZone != "" {
		if _, ok := zoneMap["trust_zone"]; ok {
			zoneMap["trust_zone"] = trustZone
		} else {
			return fmt.Errorf("trust_zone key not found in zoneMap")
		}

		if _, ok := zoneMap["untrust_zone"]; ok {
			zoneMap["untrust_zone"] = untrustZone
		} else {
			return fmt.Errorf("untrust_zone key not found in zoneMap")
		}

		return nil
	}

	// どちらの変更もなかった場合
	if len(trustZone) == 0 && untrustZone == "" {
		return nil
	}

	return fmt.Errorf("either trust_zone or untrust_zone should be specified")
}

// 順序を気にせず，スライスの要素の比較をする
func MatchElements(slice1, slice2 []string) bool {
	if len(slice1) != len(slice2) {
		return false
	}

	// スライスの要素をソートする
	sort.Strings(slice1)
	sort.Strings(slice2)

	for i := 0; i < len(slice1); i++ {
		if slice1[i] != slice2[i] {
			return false
		}
	}
	return true
}