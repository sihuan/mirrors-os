// Send @RSYNCD x.x\n
// Send modname\n
// Send arugment with mod list\0	filter list write(0)    \n
// handshake
// batch seed
// Recv file list
//

package main

import (
	"errors"
	"flag"
	"github.com/Si-Huan/rsync-os/rsync"
	"github.com/Si-Huan/rsync-os/storage"
	"github.com/robfig/cron"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/viper"
)

func initTask(task *taskConf,logger *logrus.Logger) (*MirrorItem, *storage.Teambition, func(), error) {
	options := make(map[string][]string)
	options["--exclude"] = task.Exclusion
	addr, module, path, err := rsync.SplitURI(task.Src)
	if err != nil {
		return nil, nil, nil, err
	}
	ppath := rsync.TrimPrepath(path)

	dbppath, err := rel(task.SrcRoot, task.Src)
	if err != nil {
		return nil, nil, nil, err
	}
	if task.Name == "" {
		return nil, nil, nil, errors.New("task name can't be blank")
	}
	bucket := filepath.Join(task.Base, task.Name)
	stor, err := storage.NewTeambition(bucket, dbppath, task.DBPath, task.Cookie)
	if err != nil {
		return nil, nil, nil, err
	}

	statusChan := make(chan Status)
	mi := &MirrorItem{
		ServePath:  "/" + filepath.Join(task.Name, dbppath) + "/",
		FS:         stor,
		StatusChan: statusChan,
		Status:     0,
	}

	sync := func() {
		mi.StatusChan <- UPDATING

		Logger.WithFields(logrus.Fields{
			"task": task.Name,
		}).Info("Sync Start")

		var (
			retry        = 0
			err    error = nil
			client rsync.SendReceiver
		)

		for retry == 0 || (err != nil && retry < 5) {
			client, err = rsync.SocketClient(stor, addr, module, ppath, options,logger)
			if err != nil {
				Logger.WithFields(logrus.Fields{
					"task": task.Name,
					"err": err,
					"retry": retry,
				}).Error("Sync Socket Connect Err")
				retry++
				continue
			}
			err = client.Sync()
			if err != nil {
				if err := stor.FinishSync(); err != nil {
					Logger.WithFields(logrus.Fields{
						"task": task.Name,
						"err": err,
						"at_retry": retry,
					}).Error("Sync Err FinishSync Err")
				}
				Logger.WithFields(logrus.Fields{
					"task": task.Name,
					"err": err,
					"retry": retry,
				}).Error("Sync Err")
			}
			retry++
		}

		if err != nil {
			mi.StatusChan <- FAILD
			Logger.WithFields(logrus.Fields{
				"task": task.Name,
				"err": err,
			}).Error("Sync Faild")
			return
		}

		err = stor.FinishSync()
		if err != nil {
			mi.StatusChan <- FAILD
			Logger.WithFields(logrus.Fields{
				"task": task.Name,
				"err": err,
			}).Error("Sync Success FinishSync Err")
			return
		}
		mi.StatusChan <- SUCCESS
		Logger.WithFields(logrus.Fields{
			"task": task.Name,
			"err": err,
		}).Info("Sync Success")
		return
	}

	return mi, stor, sync, nil
}

type taskConf struct {
	Name    string
	Src     string
	SrcRoot string
	Cookie  string
	Base    string
	DBPath  string
	Cron    string
	Exclusion []string
}

var Logger = logrus.New()

func main() {
	loadConfigIfExists()
	flag.Parse()
	args := flag.Args()

	globalConf := viper.GetStringMapString("global")
	if globalConf["logdir"] == "" {
		globalConf["logdir"] = "./log"
	}

	var mirrorItems []*MirrorItem
	var stors []*storage.Teambition
	var logFiles []*os.File
	c := cron.New()

	if err := createLogDir(globalConf["logdir"]); err != nil {
		panic(err)
	}
	//logFile, err := os.OpenFile(filepath.Join(globalConf["logdir"], "mirrors-os.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	//if err != nil {
	//	panic(err)
	//}
	//defer logFile.Close()
	//Logger.SetOutput(io.MultiWriter(os.Stdout, logFile))
	Logger.SetOutput(os.Stdout)

	for _, arg := range args {
		task := &taskConf{}
		if viper.UnmarshalKey(arg, task) != nil {
			panic(arg + " conf err")
		}
		logFile, err := os.OpenFile(filepath.Join(globalConf["logdir"], task.Name+".log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		logFiles = append(logFiles, logFile)
		logger := logrus.New()
		logger.SetOutput(logFile)
		mi, stor, sync, err := initTask(task,logger)
		if err != nil {
			panic(arg + " init err: " + err.Error())
		}
		mirrorItems = append(mirrorItems, mi)
		stors = append(stors, stor)
		c.AddFunc(task.Cron, sync)
		Logger.WithFields(logrus.Fields{
			"task": task.Name,
			"cron": task.Cron,
			"src": task.Src,
		}).Info("Add Cron")
	}

	mirrorServer := NewMirrorServer(mirrorItems, globalConf["server"])
	go mirrorServer.Start()
	Logger.Warn("Mirrors HTTP Server Start!")
	c.Start()
	Logger.Warn("Cron Task Start!")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
	for _, stor := range stors {
		stor.Close()
	}
	Logger.Warn("Stors  Close!")
	time.Sleep(521 * time.Millisecond)
	for _, file := range logFiles {
		file.Close()
	}
	Logger.Warn("Shutdown.")
}

func rel(srcRoot string, src string) (string, error) {
	if !strings.HasSuffix(srcRoot, "/") {
		srcRoot += "/"
	}
	if !strings.HasSuffix(src, "/") {
		src += "/"
	}

	if !strings.HasPrefix(src, srcRoot) {
		return "", errors.New("wrong src root")
	}
	return strings.TrimPrefix(src, srcRoot), nil
}

func createLogDir(path string) error {
	mask := syscall.Umask(0)
	defer syscall.Umask(mask)
	return os.MkdirAll(path, 0755)
}
