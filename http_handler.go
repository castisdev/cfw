package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/castisdev/cfm/common"
	"github.com/gorilla/mux"
)

// Heartbeat :
func Heartbeat(w http.ResponseWriter, r *http.Request) {
	api.Infof("[%s] received heartbeat request", r.RemoteAddr)
	defer api.Infof("[%s] responsed heartbeat request", r.RemoteAddr)

	w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
	w.WriteHeader(http.StatusOK)
}

// GetFileList :
func GetFileList(w http.ResponseWriter, r *http.Request) {
	api.Infof("[%s] received getFileList request", r.RemoteAddr)
	defer api.Infof("[%s] responsed getFileList request", r.RemoteAddr)

	w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
	files, err := ioutil.ReadDir(config.BaseDir)
	if err != nil {
		api.Errorf("[%s] failed to read basedir, path(%s), error(%s)",
			r.RemoteAddr, config.BaseDir, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		fmt.Fprintln(w, file.Name())
	}
}

// GetDiskUsage :
func GetDiskUsage(w http.ResponseWriter, r *http.Request) {
	api.Infof("[%s] received getDiskUsage request", r.RemoteAddr)
	defer api.Infof("[%s] responsed getDiskUsage request", r.RemoteAddr)

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	du, err := common.GetDiskUsage(config.BaseDir)
	if err != nil {
		api.Errorf("[%s] failed to get disk usage, path(%s), error(%s)",
			r.RemoteAddr, config.BaseDir, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(&du); err != nil {
		api.Errorf("[%s] failed to get disk usage, path(%s), error(%s)",
			r.RemoteAddr, config.BaseDir, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// DeleteFile :
func DeleteFile(w http.ResponseWriter, r *http.Request) {
	api.Infof("[%s] received deleteFile request", r.RemoteAddr)
	defer api.Infof("[%s] responsed deleteFile request", r.RemoteAddr)

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	vars := mux.Vars(r)
	fileName, exists := vars["fileName"]

	if !exists {
		api.Errorf("[%s] failed to delete file, request has no fileName", r.RemoteAddr)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(config.BaseDir, fileName)

	finfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			api.Warningf("[%s] failed to delete file(%s), not exist, error(%s)",
				r.RemoteAddr, filePath, err.Error())
			w.WriteHeader(http.StatusNotFound)
			return
		} else {
			api.Errorf("[%s] failed to delete file(%s), error(%s)",
				r.RemoteAddr, filePath, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	if finfo.IsDir() {
		api.Errorf("[%s] failed to delete file, it is directory(%s)",
			r.RemoteAddr, filePath)
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	if err := os.Remove(filePath); err != nil {
		api.Warningf("[%s] failed to delete file(%s), error(%s)",
			r.RemoteAddr, filePath, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	api.Infof("[%s] deleted file(%s)", r.RemoteAddr, filePath)

	w.WriteHeader(http.StatusOK)
}
