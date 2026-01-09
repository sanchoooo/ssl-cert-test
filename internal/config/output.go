package config

import (
	"cmp"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings" // <--- Added import
	"time"
)

func WriteOutputToFile(outputFile string, data []DomainValidity) error {
	slices.SortFunc(data, func(a, b DomainValidity) int {
		return cmp.Compare(a.Domain, b.Domain)
	})
	currentTime := time.Now()
	dateStamp := currentTime.Format("20060102")

	dirPath := filepath.Dir(outputFile)
	dirPath = dirPath + "/" + dateStamp + "/"
	fmt.Println("output path:", dirPath)

	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		return fmt.Errorf("error creating directory: %v", err)
	}
	filePrefix := filepath.Base(outputFile)
	outputFile = dirPath + filePrefix
	successFile := outputFile + "_success_only"
	fmt.Println("outputFile:", outputFile)

	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling JSON: %v", err)
	}

	if err := os.WriteFile(outputFile+".json", output, 0644); err != nil {
		return fmt.Errorf("error writing to file: %v", err)
	}

	file, err := os.Create(outputFile + ".csv")
	if err != nil {
		return fmt.Errorf("error creating csv file: %v", err)
	}
	defer file.Close()
	successfile, err := os.Create(successFile + ".csv")
	if err != nil {
		return fmt.Errorf("error creating csv file: %v", err)
	}
	defer successfile.Close()

	w := csv.NewWriter(file)
	defer w.Flush()
	sw := csv.NewWriter(successfile)
	defer sw.Flush()

	// Update Header with "Cipher Suite" and "FIPS Compliant"
	csvRow := []string{
		"Domain", "IP Address", "Port",
		"TLS Version", "Cipher Suite", "FIPS Compliant",
		"Chain Status", "Issuer", "Sig Algo", "SANs", // <--- New Headers
		"Serial", "Common Name", "Not Before", "Not After", "Days until Expire", "Error",
	}
	w.Write(csvRow)
	sw.Write(csvRow)
	if err := w.Error(); err != nil {
		return fmt.Errorf("error writing csv file: %v", err)
	}

	for _, r := range data {
		var csvRow []string

		fipsStatus := "No"
		if r.FIPSCompliant {
			fipsStatus = "Yes"
		}

		// Join SANs slice into a single string
		sansString := strings.Join(r.SANs, ";")

		csvRow = append(csvRow,
			r.Domain,
			fmt.Sprint(r.IPAddress),
			fmt.Sprint(r.Port),
			r.TLSVersion,
			r.CipherSuite,
			fipsStatus,
			r.ChainStatus,
			r.Issuer,        // <--- New
			r.SignatureAlgo, // <--- New
			sansString,      // <--- New
			r.Serial,
			r.CommonName,
			fmt.Sprint(r.NotBefore),
			fmt.Sprint(r.NotAfter),
			fmt.Sprint(r.DaysUntilExpiry),
			r.Error,
		)

		w.Write(csvRow)
		if r.DaysUntilExpiry != 999999 {
			sw.Write(csvRow)
		}
		if err := w.Error(); err != nil {
			return fmt.Errorf("error writing csv file: %v", err)
		}
	}
	return nil
}
