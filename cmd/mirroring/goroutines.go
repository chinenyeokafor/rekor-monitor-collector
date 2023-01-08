package main

import (
	"fmt"
	"os/exec"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(2)

	//Run 3 rekor-monitor goroutines concurrently
	for i := 0; i < 3; i++ {
		go func(filename string) {
			defer wg.Done()
			cmd := exec.Command("go", "run", "/Users/Chinenye/Documents/tmp/monitor3/rekor-monitor/cmd/mirroring/main.go", filename)
			err := cmd.Run()
			if err != nil {
				fmt.Println(err)
			}
		}(fmt.Sprintf("logInfo%d.txt", i))
	}

	//Run a client goroutines
	go func() {
		defer wg.Done()
		cmd := exec.Command("go", "run", "/Users/Chinenye/Documents/tmp/monitor3/rekor-monitor/cmd/mirroring/client.go")
		err := cmd.Run()
		if err != nil {
			fmt.Println(err)
		}
	}()

	wg.Wait()
}
