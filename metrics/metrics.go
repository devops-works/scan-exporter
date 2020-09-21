package metrics

import (
	"flag"
	"log"
	"os"
	"time"
)

// Exploit exploits each ports received by reporter.
// It should write in a log, but also expose prometheus metrics.
func Exploit(time time.Time, name, ip, port, protocol string) {
	logName := time.Format("2006-01-02_15:04:05")
	writeLog(logName+"_"+name, ip, port, protocol)

	// write something like expose() which will expose metrics to prometheus
}

// writeLog writes open ports in a log file, with a log format.
// The name of the file is the scan date and time, and the name of the target.
func writeLog(filename, ip, port, protocol string) {
	// Lookup checks if logpath flag has been set. if not, it takes its default value ("./")
	path := flag.Lookup("logpath").Value.(flag.Getter).Get().(string)
	f, err := os.OpenFile(path+"/"+filename+".log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	logger := log.New(f, "", log.LstdFlags)
	if protocol == "icmp" {
		// if protocol is icmp, we write a different line into the log
		logger.Printf("%s/%s\n", ip, protocol)
	} else {
		logger.Printf("%s:%s/%s\n", ip, port, protocol)
	}
}
