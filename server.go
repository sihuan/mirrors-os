package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"syscall"

	"github.com/sihuan/rsync-os/rsync"
	"github.com/sihuan/rsync-os/storage"
)

const (
	UPDATING = 1
	SUCCESS  = 0
	FAILD    = -1
)


const HTMLHEAD = `<html>
<head><title>mirrors-os</title></head>
<body>
<hr><pre>
`

const HTMLEND = `
</pre><hr></body>
</html>`

type Status int

//module and its storage backend
type MirrorItem struct {
	ServePath  string
	FS         rsync.FS
	StatusChan chan Status
	Status     Status
}

type MirrorServer struct {
	MirrorItems []*MirrorItem
	ServeMux    *http.ServeMux
	ServeAddr   string
}

func newHandel(base string, fs rsync.FS) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fileName, _ := filepath.Rel(base, r.RequestURI)
		url, err := fs.GetURI(fileName)
		switch err {
		case nil:
			http.Redirect(w, r, url, http.StatusFound)
			break
		case storage.NotFound,storage.TargetNotFound:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not Found."))
			break
		case storage.NotFile:
			bytes,_ := ioutil.ReadFile(filepath.Join("Public",r.RequestURI,"index"))
			w.Write(bytes)
			break
		default:
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("There is something wrong."))
			break
		}
	}
}

func (ms *MirrorServer) handelRoot(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(HTMLHEAD))
	for _, item := range ms.MirrorItems {
		s := fmt.Sprintf(`<a href="%s">%s</a>`, item.ServePath[1:], item.ServePath[1:])
		switch item.Status {
		case UPDATING:
			s += "Updating 🟨\n"
			break
		case SUCCESS:
			s += "Success 🟢\n"
			break
		case FAILD:
			s += "Failed 🔺\n"
			break
		}
		w.Write([]byte(s))
	}
	w.Write([]byte(HTMLEND))

}

func NewMirrorServer(mirrors []*MirrorItem, serveAddr string) *MirrorServer {
	ms := new(MirrorServer)
	mux := http.NewServeMux()
	ms.MirrorItems = mirrors
	ms.ServeMux = mux
	ms.ServeAddr = serveAddr
	return ms
}

func (ms *MirrorServer) Start() {
	ms.ServeMux.HandleFunc("/", ms.handelRoot)
	for _, mirror := range ms.MirrorItems {
		//base := "/" + module + "/"
		ms.ServeMux.HandleFunc(mirror.ServePath, newHandel(mirror.ServePath, mirror.FS))
	}

	for i:=0;i<len(ms.MirrorItems);i++{
		ms.GenDirHtml(i)
	}

	for i, mirror := range ms.MirrorItems {
		go func(i int, m *MirrorItem) {
			var s Status
			for {
				s = <-m.StatusChan
				ms.MirrorItems[i].Status = s
				if s == SUCCESS {
					ms.GenDirHtml(i)
				}
			}
		}(i, mirror)
	}
	if err := http.ListenAndServe(ms.ServeAddr, ms.ServeMux); err != nil {
		panic(err)
	}
}

func (ms *MirrorServer) GenDirHtml(i int) {
	mask := syscall.Umask(0)
	defer syscall.Umask(mask)
	fileList, err := ms.MirrorItems[i].FS.List()
	if err != nil {
		fmt.Println("Get filelist err")
		return
	}

	htmlFiles := make(map[string]*os.File)
	var s string
	for _, fileInfo := range fileList {
		fileInfoPath := string(fileInfo.Path)
		if fileInfo.Mode.IsDIR() {
			p := filepath.Join("Public/", ms.MirrorItems[i].ServePath, fileInfoPath)
			err := os.MkdirAll(p, 0755)
			if err != nil {
				fmt.Println("Create folder err")
				return
			}
			f, err := os.Create(filepath.Join(p, "index"))
			if err != nil {
				fmt.Println("Create html file err")
				return
			}
			f.Write([]byte(HTMLHEAD))
			htmlFiles[fileInfoPath] = f
			s = fmt.Sprintf("<a href=\"%s/\">%s/</a>\n", filepath.Base(fileInfoPath), filepath.Base(fileInfoPath))
		} else {
			s = fmt.Sprintf("<a href=\"%s\">%s</a>\n", filepath.Base(fileInfoPath), filepath.Base(fileInfoPath))
		}
		htmlFiles[filepath.Dir(fileInfoPath)].Write([]byte(s))
	}
	for _, file := range htmlFiles {
		file.Write([]byte(HTMLEND))
		file.Close()
	}
}
