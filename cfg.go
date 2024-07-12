package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func dimColor(hex string, factor float64) string {
	if factor < 0 || factor > 1 {
		return ""
	}

	if hex[0] == '#' {
		hex = hex[1:]
	}

	r, err := strconv.ParseUint(hex[0:2], 16, 8)
	if err != nil {
		return ""
	}
	g, err := strconv.ParseUint(hex[2:4], 16, 8)
	if err != nil {
		return ""
	}
	b, err := strconv.ParseUint(hex[4:6], 16, 8)
	if err != nil {
		return ""
	}

	r = uint64(float64(r) * factor)
	g = uint64(float64(g) * factor)
	b = uint64(float64(b) * factor)

	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

func ReadCfg() (map[string]string, error) {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal("error finding exe path")
	}

	exeDir := filepath.Dir(exePath)
	configPath := filepath.Join(exeDir, "config.txt")

	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line: %s", line)
		}
		config[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return config, nil

}

func GetApiKey(config map[string]string) string {
	return config["APIKEY"]
}

func GetSteamId(config map[string]string) string {
	return config["STEAMID"]
}

func GetSteamPath(config map[string]string) string {
	return config["PATH"]
}

func GetPrimaryColor(config map[string]string) string {
	return config["PRIMARYCOLOR"]
}

func GetSecondaryColor(config map[string]string) string {
	return config["SECONDARYCOLOR"]
}

func GetDimSecondary(config map[string]string) string {
	return dimColor(GetSecondaryColor(config), 0.7)
}
