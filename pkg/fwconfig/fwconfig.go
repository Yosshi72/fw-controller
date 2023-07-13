package fwconfig

import (
	"bufio"
	"fmt"
	// "io/ioutil"
	"os"
	"sort"
	"strings"
	"regexp"
)

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

func ConfigWriter(containername, filePath, newUntrustIf string, newTrustIf, newMgmtAddr []string) error {
	templateFilePath := "../../fw/fw-template.rule" // テンプレートファイルのパスを適切に指定してください
	outputFilePath := filePath            // 出力ファイルのパスを適切に指定してください
	ipv6Addresses := newMgmtAddr
	trustIf := newTrustIf
	untrustIf := newUntrustIf

	// テンプレートファイルを開く
	templateFile, err := os.Open(templateFilePath)
	if err != nil {
		return fmt.Errorf("Failed to open template file: %s", err)
	}
	defer templateFile.Close()

	// 出力ファイルを作成または上書きする
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("Failed to create output file: %s", err)
	}
	defer outputFile.Close()

	scanner := bufio.NewScanner(templateFile)
	writer := bufio.NewWriter(outputFile)
	for scanner.Scan() {
		line := scanner.Text()
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return fmt.Errorf("Failed to write to output file: %s", err)
		}
		// replace v6 address
		if strings.Contains(line, "#Allowed_Address_PLACE") {
			for _, addr := range ipv6Addresses {
				newLine := fmt.Sprintf("\t\tip6 saddr %s accept;", addr)
				_, err := writer.WriteString(newLine + "\n")
				if err != nil {
					return fmt.Errorf("Failed to write to output file: %s", err)
				}
			}
		}
		// replace trustIf, unTrustIf
		if strings.Contains(line, "TRUST_IF_NAME") {
			if strings.Contains(line, "UNTRUST_IF_NAME") {
				newLine := strings.Replace(line, "{UNTRUST_IF_NAME}", untrustIf, -1)
				newLine = strings.Replace(newLine, "# ", "", -1)
				_, err := writer.WriteString(newLine + "\n")
				if err != nil {
					return fmt.Errorf("Failed to write to output file: %s", err)
				}
			} else {
				for _, tif := range trustIf {
					newLine := strings.Replace(line, "{TRUST_IF_NAME}", tif, -1)
					newLine = strings.Replace(newLine, "# ", "", -1)
					_, err := writer.WriteString(newLine + "\n")
					if err != nil {
						return fmt.Errorf("Failed to write to output file: %s", err)
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("Failed to read template file: %s", err)
	}

	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("Failed to flush writer: %s", err)
	}
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
