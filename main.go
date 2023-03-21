package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/juju/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	port  int
	count int64

	logFile = filepath.Join(os.TempDir(), "webhook.log")

	log *zap.Logger

	broken bool
)

// https://willh.gitbook.io/build-web-application-with-golang-zhtw/03.0/03.2

func handleEvent(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		log.Warn("invalid request method", zap.String("method", req.Method))
		return
	}

	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Error("read body failed", zap.Error(err))
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	count++
	log.Info("receive a message", zap.String("message", string(b)), zap.Int64("count", count))

	if broken {
		log.Warn("return 500")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

func readLog(w http.ResponseWriter, req *http.Request) {
	b, err := ioutil.ReadFile(logFile)
	if err != nil {
		log.Error("read log file failed", zap.Error(err))
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
	log.Info("running", zap.String("addr", addr))

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("start failed", zap.Error(err))
	}
}

func encodeTimeLayout(t time.Time, layout string, enc zapcore.PrimitiveArrayEncoder) {
	type appendTimeEncoder interface {
		AppendTimeLayout(time.Time, string)
	}

	if enc, ok := enc.(appendTimeEncoder); ok {
		enc.AppendTimeLayout(t, layout)
		return
	}

	enc.AppendString(t.Format(layout))
}

func initLog() (func(), error) {
	logCfg := zap.NewProductionConfig()
	logCfg.OutputPaths = append(logCfg.OutputPaths, logFile)
	logCfg.EncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		encodeTimeLayout(t, "2006-01-02 15:04:05.000", enc)
	}

	var err error
	log, err = logCfg.Build()
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
		log.Sync()
	}, nil
}

func initFlags() {
	flag.IntVar(&port, "port", 9000, "port")
	flag.BoolVar(&broken, "broken", false, "broken")
	flag.Parse()
}

func main() {
	initFlags()

	f, err := initLog()
	if err != nil {
		panic(err)
	}
	defer f()

	log.Info("log file created", zap.String("file", logFile))

	initRouter()
	run()
}
