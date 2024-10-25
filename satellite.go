package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type InfoStructure struct {
	SatelliteId       int    `json:"satid"`
	SatelliteName     string `json:"satname"`
	TransactionsCount int    `json:"transactionscount"`
	PassesCount       int    `json:"passescount"`
}

type TLEStructure struct {
	Info InfoStructure `json:"info"`
	TLE  string        `json:"tle"`
}

type PassStructure struct {
	StartAz         float64 `json:"startAz"`
	StartAzCompass  string  `json:"startAzCompass"`
	StartEl         float64 `json:"startEl"`
	StartUTC        int64   `json:"startUTC"`
	MaxAz           float64 `json:"maxAz"`
	MaxAzCompass    string  `json:"maxAzCompass"`
	MaxEl           float64 `json:"maxEl"`
	MaxUTC          int64   `json:"maxUTC"`
	EndAz           float64 `json:"endAz"`
	EndAzCompass    string  `json:"endAzCompass"`
	EndEl           float64 `json:"endEl"`
	EndUTC          int64   `json:"endUTC"`
	Mag             float64 `json:"mag"`
	Duration        int     `json:"duration"`
	StartVisibility int64   `json:"startVisibility"`
}

type VisualPassesStructure struct {
	Info   InfoStructure   `json:"info"`
	Passes []PassStructure `json:"passes"`
}

var DEBUG bool

func getVisualPasses(satelliteIds []int) ([]VisualPassesStructure, error) {
	// Whether running in disconnect mode with offline data
	DEBUG = true

	var results []VisualPassesStructure

	for _, satelliteId := range satelliteIds {
		raw, err := performVisualPasses(satelliteId)
		if err != nil {
			return results, fmt.Errorf("obtaining visual pass information: %v", err)
		}

		// Unmarshal the JSON data into the struct
		var visualPassesStruct VisualPassesStructure
		err = json.Unmarshal(raw, &visualPassesStruct)
		if err != nil {
			return results, fmt.Errorf("reading visual pass information: %v", err)
		}

		results = append(results, visualPassesStruct)
	}

	return results, nil
}

func performVisualPasses(satelliteId int) ([]byte, error) {
	if DEBUG {
		return performVisualPassesDebug(satelliteId)
	}

	return performVisualPassesLive(satelliteId)
}

func performVisualPassesDebug(satelliteId int) ([]byte, error) {
	// If debug, read from file
	data, err := os.ReadFile(fmt.Sprintf("./examples/visualpasses-%d.json", satelliteId))
	if err != nil {
		return nil, err
	}

	return data, err
}

func performVisualPassesLive(satelliteId int) ([]byte, error) {

	// Base URL of the API
	baseURL := "https://api.n2yo.com/rest/v1/satellite"

	// Read apiKey and anything else relevant from a local file
	env, err := readHeadersFromDotfile(".env")
	if err != nil {
		return nil, fmt.Errorf("reading header information: %v", err)
	}

	// Read locatiton information from a local file
	location, err := readHeadersFromDotfile(".location")
	if err != nil {
		return nil, fmt.Errorf("reading location detail: %v", err)
	}

	// Read preferences from a local file
	preferences, err := readHeadersFromDotfile(".preferences")
	if err != nil {
		return nil, fmt.Errorf("reading preferences: %v", err)
	}

	// Query parameters - will be used to add the apiKey and maybe other details
	queryParams := url.Values{}

	// Put all values read from the dotfile as header entries
	for key, value := range env {
		queryParams.Add(key, value)
	}

	// Construct the full URL
	fullURL := fmt.Sprintf("/visualpasses/%d/%s/%s/%s/%s/%s?%s", satelliteId, location["latitude"], location["longitude"], location["altitude"], preferences["days"], preferences["minimum_visibility"], queryParams.Encode())

	if strings.Contains(fullURL, "//") {
		return nil, fmt.Errorf("some location or preference information is missing: %s", fullURL)
	}

	// Create a new HTTP client
	client := &http.Client{}

	// Create a new GET request
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", baseURL, fullURL), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %v", err)
	}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %v", err)
	}

	return body, nil
}

func printVisualPasses(structure VisualPassesStructure) {
	fmt.Printf("Satellite name     : %s\n", structure.Info.SatelliteName)
	fmt.Printf("Satellite ID       : %d\n", structure.Info.SatelliteId)
	fmt.Printf("Transactions Count : %d\n", structure.Info.TransactionsCount)
	fmt.Printf("Passes Count       : %d\n", structure.Info.PassesCount)

	for i := 0; i < structure.Info.PassesCount; i++ {
		fmt.Printf("Pass %2d:\n", i)
		fmt.Printf("  Magnitude         : %f (brightness)\n", structure.Passes[i].Mag)
		fmt.Printf("  Duration          : %ds (%s)\n", structure.Passes[i].Duration, secondsToDuration(int64(structure.Passes[i].Duration)))
		fmt.Printf("  Start Visibility  : %s\n", utcSecondsToLocalTime(structure.Passes[i].StartVisibility))
		fmt.Println()
		fmt.Printf("  Start             : %s\n", utcSecondsToLocalTime(structure.Passes[i].StartUTC))
		fmt.Printf("  Start Azimuth     : %.2f\u00B0 (%s)\n", structure.Passes[i].StartAz, structure.Passes[i].StartAzCompass)
		fmt.Printf("  Start Elevation   : %.2f\u00B0\n", structure.Passes[i].StartEl)
		fmt.Println()
		fmt.Printf("  Max               : %s\n", utcSecondsToLocalTime(structure.Passes[i].MaxUTC))
		fmt.Printf("  Max Azimuth       : %.2f\u00B0 (%s)\n", structure.Passes[i].MaxAz, structure.Passes[i].MaxAzCompass)
		fmt.Printf("  Max Elevation     : %.2f\u00B0\n", structure.Passes[i].MaxEl)
		fmt.Println()
		fmt.Printf("  End               : %s\n", utcSecondsToLocalTime(structure.Passes[i].EndUTC))
		fmt.Printf("  End Azimuth       : %.2f\u00B0 (%s)\n", structure.Passes[i].EndAz, structure.Passes[i].EndAzCompass)
		fmt.Printf("  End Elevation     : %.2f\u00B0\n", structure.Passes[i].EndEl)
		fmt.Println()
	}
}

// Read a dotfile, formatted as a property file, into a string map
func readHeadersFromDotfile(filename string) (map[string]string, error) {
	headers := make(map[string]string)

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Ignore empty lines and comment lines (starting with #)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			headers[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return headers, nil
}

func secondsToDuration(totalSeconds int64) string {
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%dm %2ds", minutes, seconds)
}

func secondsToTime(seconds int64) time.Time {
	return time.Unix(seconds, 0).UTC()
}

func utcSecondsToLocalTime(utcSeconds int64) string {
	// Convert to time.Time and then to local time
	t := secondsToTime(utcSeconds).Local()

	// fmt.Println(t.Format(time.RFC822))
	// fmt.Println(t.Format(time.RFC822Z))
	// fmt.Println(t.Format(time.RFC850))
	// fmt.Println(t.Format(time.RFC1123))
	// fmt.Println(t.Format(time.RFC1123Z))
	// fmt.Println(t.Format(time.RFC3339))
	// fmt.Println(t.Format(time.RFC3339Nano))

	// Format and print the time
	return t.Format(time.RFC822)
}
