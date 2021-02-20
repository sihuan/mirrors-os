// Send @RSYNCD x.x\n
// Send modname\n
// Send arugment with mod list\0	filter list write(0)    \n
// handshake
// batch seed
// Recv file list
//

package main

import (
	"flag"
	"github.com/Si-Huan/rsync-os/rsync"
	"github.com/Si-Huan/rsync-os/storage"
	"github.com/robfig/cron"
	"log"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/spf13/viper"
)

func initTask(task *taskConf) (*MirrorItem, *storage.Teambition,func(), error) {
	addr, module, path, err := rsync.SplitURI(task.Src)

	if err != nil {
		log.Println("Invaild URI")
		return nil, nil,nil, err
	}

	log.Println(module, path)

	ppath := rsync.TrimPrepath(path)

	bucket := filepath.Join(task.Base, module)

	stor, err := storage.NewTeambition(bucket, ppath, task.DBPath, task.Cookie)
	if err != nil {
		return nil, nil,nil, err
	}

	statusChan := make(chan Status)
	mi := &MirrorItem{
		ServePath:  "/" + filepath.Join(module, ppath) + "/",
		FS:         stor,
		StatusChan: statusChan,
		Status:     0,
	}

	sync := func() {
		mi.StatusChan <- UPDATING
		client, err := rsync.SocketClient(stor, addr, module, ppath, nil)
		if err != nil {
			mi.StatusChan <- FAILD
			return
		}
		if err := client.Sync(); err != nil {
			mi.StatusChan <- FAILD
			return
		}
		err = stor.FinishSync()
		if err != nil {
			mi.StatusChan <- FAILD
			return
		}
		mi.StatusChan <- SUCCESS
		return
	}

	return mi, stor,sync, nil
}

type taskConf struct {
	Src    string
	Cookie string
	Base   string
	DBPath string
	Cron   string
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
		mi,stor, sync, err := initTask(task)
		if err != nil {
			panic(arg + " init err")
		}
		mirrorItems = append(mirrorItems, mi)
		stors = append(stors,stor)
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
