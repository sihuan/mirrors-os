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
	"fmt"
	"github.com/Si-Huan/rsync-os/rsync"
	"github.com/Si-Huan/rsync-os/storage"
	"github.com/robfig/cron"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

func initTask(task *taskConf) (*MirrorItem, *storage.Teambition, func(), error) {
	addr, module, path, err := rsync.SplitURI(task.Src)
	if err != nil {
		log.Println("Invaild URI")
		return nil, nil, nil, err
	}
	ppath := rsync.TrimPrepath(path)
	log.Println(module, ppath)

	dbppath, err := rel(task.SrcRoot, task.Src)
	if err != nil {
		log.Println("Invaild Root")
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

		var (
			retry        = 0
			err    error = nil
			client rsync.SendReceiver
		)

		for retry == 0 || (err != nil && retry < 5) {
			client, err = rsync.SocketClient(stor, addr, module, ppath, nil)
			if err != nil {
				fmt.Println(task.Name, "Sync Socket Connect Err: ", err, "Retry:", retry)
				retry++
				continue
			}
			err = client.Sync()
			if err != nil {
				if err := stor.FinishSync(); err != nil {
					fmt.Println(task.Name, "Sync Err FinishSync Err: ", err)
				}
				fmt.Println(task.Name, "Sync Err: ", err, "Retry:", retry)
			}
			retry++
		}

		if err != nil {
			fmt.Println(task.Name, "Sync F,ERR: ", err)
			mi.StatusChan <- FAILD
			return
		}

		err = stor.FinishSync()
		if err != nil {
			fmt.Println("Sync Success FinishSync Err: ", err)
			mi.StatusChan <- FAILD
			return
		}
		mi.StatusChan <- SUCCESS
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
}

func main() {
	loadConfigIfExists()
	flag.Parse()
	args := flag.Args()

	var mirrorItems []*MirrorItem
	var stors []*storage.Teambition

	c := cron.New()

	for _, arg := range args {
		task := &taskConf{}
		if viper.UnmarshalKey(arg, task) != nil {
			panic(arg + " conf err")
		}
		mi, stor, sync, err := initTask(task)
		if err != nil {
			panic(arg + " init err")
		}
		mirrorItems = append(mirrorItems, mi)
		stors = append(stors, stor)
		c.AddFunc(task.Cron, sync)
	}
	globalConf := viper.GetStringMapString("global")
	mirrorServer := NewMirrorServer(mirrorItems, globalConf["server"])
	go mirrorServer.Start()
	c.Start()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
	for _, stor := range stors {
		stor.Close()
	}
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
