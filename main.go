package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cilog"
)

var config *Config
var myIP string

// App constant
const (
	AppName      = "cfw"
	AppVersion   = "1.0.0"
	AppPreRelVer = "-qr1"
)

func main() {

	debug.SetTraceback("crash")

	printSimpleVer := flag.Bool("v", false, "print version")
	printVer := flag.Bool("version", false, "print version includes pre-release version")
	flag.Parse()

	if err := common.EnableCoreDump(); err != nil {
		log.Fatal(err)
	}

	if *printSimpleVer {
		fmt.Println(AppName + " " + AppVersion)
		os.Exit(0)
	}

	if *printVer {
		fmt.Println(AppName + " " + AppVersion + AppPreRelVer)
		os.Exit(0)
	}

	var err error
	config, err = ReadConfig("cfw.yml")
	if err != nil {
		panic(err)
	}

	ValidationConfig(*config)

	logLevel, _ := cilog.LevelFromString(config.LogLevel)

	cilog.Set(cilog.NewLogWriter(config.LogDir, AppName, 10*1024*1024), AppName, AppVersion, logLevel)

	myIP, err = common.GetIPv4ByInterfaceName(config.IFName)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	cilog.Infof("process start")
	dl := NewDownloader(config.BaseDir, myIP, config.DownloaderBin, config.CFMAddr, config.StorageUsageLimitPercent)
	go dl.RunForever()

	router := NewRouter()

	s := &http.Server{
		Addr:         config.ListenAddr,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	s.ListenAndServe()

}
