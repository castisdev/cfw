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
var api common.MLogger

// App constant
const (
	AppName      = "cfw"
	AppVersion   = "1.0.0"
	AppPreRelVer = "QR2"
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
		fmt.Println(AppName + " " + AppVersion + "." + AppPreRelVer)
		os.Exit(0)
	}

	var err error
	config, err = ReadConfig("cfw.yml")
	if err != nil {
		panic(err)
	}

	ValidationConfig(*config)

	logLevel, _ := cilog.LevelFromString(config.LogLevel)

	mLogWriter := common.MLogWriter{
		LogWriter: cilog.NewLogWriter(config.LogDir, AppName, 10*1024*1024),
		Dir:       config.LogDir,
		App:       AppName,
		MaxSize:   (10 * 1024 * 1024)}

	api = common.MLogger{
		Logger: cilog.StdLogger(),
		Mod:    "api"}

	cilog.Set(mLogWriter,
		AppName, AppVersion, logLevel)

	myAddr := config.ListenAddr
	cilog.Infof("process start")
	dl := NewDownloader(config.BaseDir, myAddr, config.DownloaderBin,
		config.CFMAddr, config.StorageUsageLimitPercent, config.DownloaderSleepSec)
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
