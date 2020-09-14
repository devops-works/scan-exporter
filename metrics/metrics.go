package metrics

import (
	"log"
	"os"
)

// WriteLog writes open ports in a log file, with a log format.
// The name of the file is the scan date and time, and the name of the target
func WriteLog(filename, ip, port, protocol string) {
	f, err := os.OpenFile(filename+".log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	logger := log.New(f, "", log.LstdFlags)
	logger.Printf("%s:%s/%s\n", ip, port, protocol)
}
