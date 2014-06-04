package main

import (
	"flag"
	"log"
	"os"
)

var debug, trace bool

func main() {
	monitor, handler := parseCommandLineArgs()

	changeChannel := make(chan string)

	quit := monitor.StartDirectoryMonitor(changeChannel)
	handler.StartChangeHandler(changeChannel)

	<-quit
}

func parseCommandLineArgs() (*DirectoryMonitor, *ChangeHandler) {
	monitor := NewDirectoryMonitor()
	handler := NewChangeHandler()
	flag.IntVar(&monitor.Interval, "interval", 5, "Interval in seconds")
	flag.StringVar(&monitor.Dir, "dir", "./", "Directory to monitor")
	flag.BoolVar(&monitor.NoTraverse, "no-traverse", false, "Flag for turning off monitoring entire directory tree")
	flag.StringVar(&monitor.IncludesPattern, "includes", "*", "File name pattern (no dir info) to match to include in scan (comma separated)")
	flag.StringVar(&monitor.ExcludesPattern, "excludes", "", "File name or directory name pattern to match to exclude in scan (comma separated)")

	flag.StringVar(&handler.Dir, "cd", "", "Directory to run command from")
	flag.StringVar(&handler.Command, "command", "", "Command to run upon finding a change in the monitored directory tree")

	flag.BoolVar(&debug, "v", false, "Turn on debug mode")
	flag.BoolVar(&trace, "vv", false, "Turn on trace mode")

	flag.Parse()

	if handler.Command == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if debug {
		log.Println("DirectoryMonitor:  ", monitor)
		log.Println("ChangeHandler:     ", handler)
	}

	return monitor, handler
}
