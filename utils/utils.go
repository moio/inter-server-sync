// SPDX-FileCopyrightText: 2023 SUSE LLC
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Return default config paths - etc default, web default, package default
func getDefaultConfigs() []string {
	return []string{"/etc/rhn/rhn.conf",
		"/usr/share/rhn/config-defaults/rhn_web.conf",
		"/usr/share/rhn/config-defaults/rhn.conf"}
}

//ReverseArray reverses the array
func ReverseArray(slice interface{}) {
	size := reflect.ValueOf(slice).Len()
	swap := reflect.Swapper(slice)
	for i, j := 0, size-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
}

// Contains is a helper method to check if a string element exist in the string slice
func Contains(slice []string, elementToFind string) bool {
	for _, element := range slice {
		if strings.ToLower(elementToFind) == strings.ToLower(element) {
			return true
		}
	}
	return false
}

func GetAbsPath(path string) string {
	result := path
	if filepath.IsAbs(path) {
		result, _ = filepath.Abs(path)
	} else {
		homedir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal().Msg("Couldn't determine the home directory")
		}
		if strings.HasPrefix(path, "~") {
			result = strings.Replace(path, "~", homedir, -1)
		}
	}
	return result
}

func FolderExists(path string) error {
	folder, err := os.Open(path)
	defer folder.Close()
	if err != nil {
		return err
	}
	folderInfo, err := folder.Stat()
	if err != nil {
		return err
	}
	if !folderInfo.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}
	return nil
}

func GetCurrentServerVersion(serverConfig string) (string, string) {
	files := []string{serverConfig}
	files = append(files, getDefaultConfigs()...)
	property := []string{"product_name", "web.product_name"}
	product := "SUSE Manager"
	p, err := getProperty(files, property)
	if err == nil {
		product = p
	}

	propertyVersion := []string{"web.version"}
	if product != "SUSE Manager" {
		propertyVersion = []string{"web.version.uyuni"}
		product = "uyuni"
	}
	version, err := getProperty(files, propertyVersion)
	if err != nil {
		log.Fatal().Msgf("No version found for product %s", product)
	}
	return version, product
}

func GetCurrentServerFQDN(serverConfig string) string {
	files := []string{serverConfig}
	files = append(files, getDefaultConfigs()...)
	property := []string{"cobbler.host"}
	p, err := getProperty(files, property)
	if err != nil {
		log.Error().Msgf("FQDN of server not found, images pillar may not be processed correclty")
		return ""
	}
	return p
}

func getProperty(filePaths []string, names []string) (string, error) {
	for _, search := range names {
		for _, path := range filePaths {
			p, err := ScannerFunc(path, search)
			if err == nil {
				return p, nil
			}
		}
	}
	return "", fmt.Errorf("String not found!")
}

func ScannerFunc(path string, search string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal().Err(err).Msgf("Couldn't open file: %s", path)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		linetext := scanner.Text()

		index := strings.Index(linetext, "=")
		if index < 0 {
			continue
		}
		key := strings.Trim(linetext[:index], " ")
		if key == search {
			return strings.Trim(linetext[index+1:], " "), nil
		}
	}
	return "", fmt.Errorf("String not found!")
}

func ValidateDate(date string) (string, bool) {
	if date == "" {
		return "", true
	}

	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02"} {
		t, err := time.Parse(layout, date)
		if err == nil {
			return t.Format(layout), true
		}
	}
	return "", false
}

func ReadFileByLine(path string) []string {

	msg := fmt.Sprintf("error opening file at %s", path)
	file, err := os.Open(path)
	checkError(err, msg)
	defer func(file *os.File) {
		err := file.Close()
		checkError(err, msg)
	}(file)

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	var labels []string
	for scanner.Scan() {
		labels = append(labels, scanner.Text())
	}
	return labels
}

// ExecInteractivePrompt calls a command, expects an interactive prompt to start, passes the given input into it.
func ExecInteractivePrompt(name string, input string) error {
	cmd := exec.Command(name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	buffer := bytes.Buffer{}
	buffer.Write([]byte(input))
	cmd.Stdin = &buffer

	return cmd.Run()
}

func checkError(err error, msg string) {
	if err != nil {
		log.Fatal().Err(err).Msg(msg)
	}
}
