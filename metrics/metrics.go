package metrics

import (
	"flag"
	"log"
	"os"
)

// WriteLog writes open ports in a log file, with a log format.
// The name of the file is the scan date and time, and the name of the target
func WriteLog(filename, ip, port, protocol string) {
	// Lookup checks if logpath flag has been set. if not, it takes its default value ("./")
	path := flag.Lookup("logpath").Value.(flag.Getter).Get().(string)
	f, err := os.OpenFile(path+"/"+filename+".log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	logger := log.New(f, "", log.LstdFlags)
	if protocol == "icmp" {
		// if protocol is icmp, we write a special line into the log
		logger.Printf("%s responds to ping\n", ip)
	} else {
		logger.Printf("%s:%s/%s\n", ip, port, protocol)
	}
}
