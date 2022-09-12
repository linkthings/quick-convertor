package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"

	nested "github.com/antonfisher/nested-logrus-formatter"

	"github.com/sirupsen/logrus"
)

const usageTmpl = `
Extract, filter and transform csv file into Microsoft Excel file

Usage: 
  qc -c config file
`

var versionStr = "1.0"
var log = logrus.New()

const (
	LOG_DEBUG = 10
	LOG_INFO  = 5
)

var infof = func(level int, format string, args ...interface{}) {
	if *logLevel >= level {
		log.Printf(format, args...)
	}
}

var fatalf = func(arg string, v ...interface{}) {
	log.Fatalf(arg, v...)
}

var errorf = func(arg string, v ...interface{}) {
	log.Errorf(arg, v...)
}

func showUsage() {
	fmt.Fprintf(os.Stdout, "%s", usageTmpl)
	fmt.Fprintf(os.Stdout, "Flags:\n")
	flag.PrintDefaults()
}

var (
	helpFlag    = flag.Bool("h", false, "Display the help menu")
	versionFlag = flag.Bool("v", false, "Display version information")
	logLevel    = flag.Int("l", 5, "Set the info level for troubleshooting")
	warning     = flag.Bool("w", false, "Display the warning information")
	inputFile   = flag.String("c", "./config.yaml", "Set the configuration file")

	cpuProfile = flag.String("cpuprof", "", "Writes CPU profile to the specified file")
	memProfile = flag.String("memprof", "", "Writes memory profile to the specified file")
)

func main() {
	var err error

	log.SetOutput(os.Stdout)
	//log.SetOutput(ioutil.Discard)
	log.SetLevel(logrus.DebugLevel)
	log.SetFormatter(&nested.Formatter{
		HideKeys: true,
		NoColors: false,
	})

	flag.Parse()

	cpuProfiling := false

	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create CPU profiling file: %v\n", err)
			os.Exit(1)
		}
		if err = pprof.StartCPUProfile(f); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start CPU profiling: %v\n", err)
			os.Exit(1)
		}
		cpuProfiling = true
	}

	// show help if requested
	if *helpFlag || len(*inputFile) == 0 {
		showUsage()
		os.Exit(0)
	}

	// show version if requested
	if *versionFlag {
		fmt.Printf("Version: %s\n", versionStr)
		os.Exit(0)
	}

	ReadConfig(*inputFile)

	err = ProcessCSVFile(config)
	if err != nil {
		fatalf("unable to process CSV: %s", err)
	}

	//save the subFiles
	for _, subFile := range config.Subfiles {
		processSubFiles(subFile)
	}

	if *memProfile != "" {
		f, err := os.Create(*memProfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create memory profiling file: %v\n", err)
			os.Exit(1)
		}

		runtime.GC()
		if err := pprof.WriteHeapProfile(f); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write memory profiling data: %v", err)
			os.Exit(1)
		}
		_ = f.Close()
	}

	if cpuProfiling {
		pprof.StopCPUProfile()
	}
}
