package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Default path for monitor and client logfile
const (
	//logInfoFilePath  = "rekor-monitor/cmd/mirroring/logInfo.txt"
	AcceptedChptFile = "accepted_chpt.txt"
	MonitorList      = "monitor_list.json"
)

// Define a struct to represent the monitor_list JSON data.
type monitorList struct {
	Monitors []struct {
		Description string `json:"description"`
		Logfile     string `json:"logfile"`
	} `json:"monitors"`
}

// readLatestCheckpoints reads the latest two checkpoints from the given file.
func readLatestCheckpoints(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var checkpoints []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		checkpoints = append(checkpoints, scanner.Text())
		if len(checkpoints) > 2 {
			checkpoints = checkpoints[len(checkpoints)-2:]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return checkpoints, nil
}

// deleteOldCheckpoints persists the latest 100 checkpoints. This expects that the log file
// is not being concurrently written to.
func deleteOldCheckpoints(filename string) error {
	// read all lines from file
	file, err := os.Open(filename)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	// exit early if there aren't checkpoints to truncate
	if len(lines) <= 20 {
		return nil
	}

	// open file again to overwrite
	file, err = os.OpenFile(filename, os.O_RDWR|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	for i := len(lines) - 20; i < len(lines); i++ {
		if _, err := file.WriteString(fmt.Sprintf("%s\n", lines[i])); err != nil {
			return err
		}
	}

	return nil
}

func initMonitors(logInfoFilePath string) ([]string, error) {
	// Read the contents of the JSON file.
	contents, err := ioutil.ReadFile(logInfoFilePath)
	if err != nil {
		return nil, err
	}

	// Unmarshal the JSON data into a monitorList struct.
	var list monitorList
	if err := json.Unmarshal(contents, &list); err != nil {
		return nil, err
	}

	// Create the monitors slice.
	monitors := make([]string, len(list.Monitors))

	// Populate the monitors slice with the logfile values.
	for i, m := range list.Monitors {
		monitors[i] = m.Logfile
	}

	return monitors, nil
}

func main() {
	interval := flag.Duration("interval", 1*time.Minute, "Length of interval between each periodical check")
	for {
		monitors, err := filepath.Glob("./logInfo*.txt")
		if err != nil {
			log.Fatalf("Finding files with .txt extension: %v", err)
		}
		i := int(math.Round(0.75 * float64(len(monitors))))
		var checkpoints [][]string
		for _, monitor := range monitors {
			chpts, err := readLatestCheckpoints(monitor)
			if err != nil {
				log.Fatalf("Reading checkpoints from %q: %v", monitor, err)
			}
			checkpoints = append(checkpoints, chpts)
		}

		// Count the number of monitors that agree on each treesize and accept their checkpoints.
		counts := make(map[string]int)
		for _, chpts := range checkpoints {
			// Extract the tree_size from all checkpoint.
			for _, chpt := range chpts {
				fields := strings.Split(chpt, "\\n")
				treeSize, err := strconv.Atoi(fields[1])
				if err != nil {
					log.Fatalf("Converting tree size to int: %v", err)
				}

				counts[strconv.Itoa(treeSize)]++
			}
		}

		// Write all accepted checkpoints to the "AcceptedChptFile" file.
		file, err := os.OpenFile(AcceptedChptFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Fatalf("Opening AcceptedChptFile: %v", err)
		}
		defer file.Close()

		// Find the largest tree size that appears at least twice in the list of checkpoints.
		maxTreeSize := 0
		largestTimestamp := int64(0) // Initialize largestTimestamp to the minimum possible value
		var latest_chpt string
		for _, chpts := range checkpoints {
			for _, chpt := range chpts {
				fields := strings.Split(chpt, "\\n")
				treeSize, err := strconv.Atoi(fields[1])
				if err != nil {
					log.Fatalf("Converting tree size to int: %v", err)
				}
				if counts[strconv.Itoa(treeSize)] >= i && treeSize >= maxTreeSize {
					maxTreeSize = treeSize

					// Write only the checkpoints with the largest tree size and the newest timestamp to the file.
					timestamp, err := strconv.ParseInt(strings.TrimSpace(strings.Split(fields[3], ":")[1]), 10, 64)
					if err != nil {
						log.Printf("Parsing timestamp: %v", err)
						continue // Skip this checkpoint
					}

					if treeSize == maxTreeSize && timestamp > largestTimestamp {
						largestTimestamp = timestamp
						latest_chpt = chpt
					}
				}
			}
		}
		fmt.Fprintln(file, latest_chpt)
		if err := deleteOldCheckpoints(AcceptedChptFile); err != nil {
			log.Fatalf("failed to delete old checkpoints: %v", err)
		}
		time.Sleep(*interval)
	}

}
