package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/juju/errors"
	"go.uber.org/zap"
)

var (
	port  int
	count int64

	logFile = filepath.Join(os.TempDir(), "webhook.log")

	logger *zap.Logger
)

// https://willh.gitbook.io/build-web-application-with-golang-zhtw/03.0/03.2

func handleEvent(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		logger.Warn("invalid request method", zap.String("method", req.Method))
		return
	}

	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.Error("read body failed", zap.Error(err))
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	count++
	logger.Info("receive a message", zap.String("message", string(b)), zap.Int64("count", count))

	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

func readLog(w http.ResponseWriter, req *http.Request) {
	b, err := ioutil.ReadFile(logFile)
	if err != nil {
		logger.Error("read log file failed", zap.Error(err))
		http.Error(w, "can't read log file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

func initRouter() {
	http.HandleFunc("/", handleEvent)
	http.HandleFunc("/log", readLog)
}

func run() {
	addr := fmt.Sprintf(":%d", port)
	logger.Info("running", zap.String("addr", addr))

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		logger.Fatal("start failed", zap.Error(err))
	}
}

func initLog() (func(), error) {
	logCfg := zap.NewProductionConfig()
	logCfg.OutputPaths = append(logCfg.OutputPaths, logFile)

	var err error
	logger, err = logCfg.Build()
	if err != nil {
		return nil, errors.Trace(err)
	}

	// log.Printf("log file: %s", logFile)
	// f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	// if err != nil {
	// 	panic(err)
	// }
	// log.SetOutput(f)

	return func() {
		logger.Sync()
	}, nil
}

func initFlags() {
	flag.IntVar(&port, "port", 9000, "port")
	flag.Parse()
}

func main() {
	initFlags()

	f, err := initLog()
	if err != nil {
		panic(err)
	}
	defer f()

	logger.Info("log file created", zap.String("file", logFile))

	initRouter()
	run()
}
