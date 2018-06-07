package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"syscall"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cilog"

	"github.com/gorilla/mux"
)

// GetFileList :
func GetFileList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain;charset=UTF-8")

	files, err := ioutil.ReadDir(config.BaseDir)
	if err != nil {
		cilog.Warningf("fail to get file list,path(%s),error(%s)", config.BaseDir, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	for _, file := range files {
		fmt.Fprintln(w, file.Name())
	}

}

// GetDiskUsage :
func GetDiskUsage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	fs := syscall.Statfs_t{}
	if err := syscall.Statfs(config.BaseDir, &fs); err != nil {
		cilog.Warningf("fail to get disk usage,path(%s),error(%s)", config.BaseDir, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	t := int64(fs.Blocks * uint64(fs.Bsize))
	f := int64(fs.Bfree * uint64(fs.Bsize))
	u := t - f
	p := int((u*100/t*100)/100 + 1)

	du := common.DiskUsage{
		TotalSize:   t,
		UsedSize:    u,
		FreeSize:    f,
		UsedPercent: p,
	}

	if err := json.NewEncoder(w).Encode(&du); err != nil {
		// w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// w.WriteHeader(http.StatusOK)

}

// DeleteFile :
func DeleteFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	vars := mux.Vars(r)
	fileName, exists := vars["fileName"]

	if !exists {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filePath := config.BaseDir + "/" + fileName
	if err := os.Remove(filePath); err != nil {
		cilog.Warningf("fail to delete file,error(%s)", err)
	}

	cilog.Infof("delete file(%s)", filePath)

	w.WriteHeader(http.StatusOK)

}
