package fwconfig

import (
	"bufio"
	"fmt"
	// "io/ioutil"
	"os"
	"sort"
	"regexp"
)

type Configuration struct {
	Netns              string                 `json:"netns"`
	Interfaces         map[string]interface{} `json:"interfaces"`
	PermittedInboundNW []string               `json:"inbound_allowed_network"`
}

func RulesReader(filePath string) ([]string, string, []string, error) {
	// read configutation
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", nil, fmt.Errorf("Failed to open file: %v", err)
	}
	defer file.Close()

	// 正規化パターン
	ip6Regex := regexp.MustCompile(`^[\s\t]*ip6 saddr ([\w:\/]+) accept` )
	trustIfRegex := regexp.MustCompile(`^[\s\t]*oifname ([\w]+) jump ZONE_TRUST`)
	untrustIfRegex := regexp.MustCompile(`^[\s\t]*oifname ([\w-]+) jump ZONE_UNTRUST`)

	scanner := bufio.NewScanner(file)
	var ipv6Addresses []string
	var trustIf []string
	var untrustIf string

	// 抽出
	for scanner.Scan() {
		line := scanner.Text()
		matchIP6 := ip6Regex.FindStringSubmatch(line)
		if matchIP6 != nil {
			ipv6Addresses = append(ipv6Addresses, matchIP6[1])
		}
		matchTrustIf := trustIfRegex.FindStringSubmatch(line)
		if matchTrustIf != nil {
			trustIf = append(trustIf, matchTrustIf[1])
		}
		matchUntrustIf := untrustIfRegex.FindStringSubmatch(line)
		if matchUntrustIf != nil {
			untrustIf = matchUntrustIf[1]
		}
	}

	return trustIf, untrustIf, ipv6Addresses, nil
}

// func ConfigWriter(containername, configFile, newUntrustIf string, newTrustIf, newMgmtAddr []string) error {
// 	// read configutation
// 	file, err := ioutil.ReadFile(configFile)
// 	if err != nil {
// 		return fmt.Errorf("failed to read config file: %v", err)
// 	}

// 	// json decode
// 	var data Configuration
// 	err = json.Unmarshal(file, &data)
// 	if err != nil {
// 		return fmt.Errorf("failed to parse the JSON: %v", err)
// 	}

// 	// interfacesの更新
// 	trustif, untrustif := newTrustIf, newUntrustIf
// 	err = updateZone(data.Interfaces, trustif, untrustif)
// 	if err != nil {
// 		return fmt.Errorf("failed to update zone: %v", err)
// 	}

// 	// inbound_allowed_networkの更新
// 	// TODO: エラーハンドリング
// 	data.PermittedInboundNW = newMgmtAddr

// 	// 構造体をJSON形式に変換
// 	newData, err := json.MarshalIndent(data, "", "    ")
// 	if err != nil {
// 		return fmt.Errorf("failed to convert to JSON: %v", err)
// 	}

// 	// write to configuration
// 	err = ioutil.WriteFile(configFile, newData, 0644)
// 	if err != nil {
// 		return fmt.Errorf("failed to write to configuration file: %v", err)
// 	}

// 	fmt.Println("Success: Update configuration")
// 	return nil
// }

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
