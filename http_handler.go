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
	w.Header().Set("Content-Type", "text/plain;charset=UTF-8")

	api.Debugf("[%s] receive heartbeat request", r.RemoteAddr)
	w.WriteHeader(http.StatusOK)
}

// GetFileList :
func GetFileList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain;charset=UTF-8")

	files, err := ioutil.ReadDir(config.BaseDir)
	if err != nil {
		api.Errorf("[%s] fail to get file list, path(%s), error(%s)",
			r.RemoteAddr, config.BaseDir, err)
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

	api.Debugf("[%s] receive get file list request", r.RemoteAddr)
}

// GetDiskUsage :
func GetDiskUsage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	du, err := common.GetDiskUsage(config.BaseDir)
	if err != nil {
		api.Errorf("[%s] fail to get disk usage, path(%s), error(%s)",
			r.RemoteAddr, config.BaseDir, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(&du); err != nil {
		api.Errorf("[%s] fail to get disk usage, path(%s), error(%s)",
			r.RemoteAddr, config.BaseDir, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	api.Debugf("[%s] receive get disk usage request", r.RemoteAddr)

	// https://github.com/golang/go/issues/18761
	// 버그? : 어떻게 고치는 지는 모르겠음
	//w.WriteHeader(http.StatusOK)
}

// DeleteFile :
func DeleteFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	vars := mux.Vars(r)
	fileName, exists := vars["fileName"]

	if !exists {
		api.Errorf("[%s] fail to delete file, request has no fileName", r.RemoteAddr)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(config.BaseDir, fileName)

	finfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			api.Warningf("[%s] fail to delete file(%s), not exist, error(%s)",
				r.RemoteAddr, filePath, err)
			w.WriteHeader(http.StatusNoContent)
			return
		} else {
			api.Errorf("[%s] fail to delete file(%s), error(%s)", r.RemoteAddr, filePath, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	if finfo.IsDir() {
		api.Errorf("[%s] fail to delete file, it is directory(%s)", r.RemoteAddr, filePath)
		w.WriteHeader(http.StatusConflict)
		return
	}

	if err := os.Remove(filePath); err != nil {
		api.Warningf("[%s] fail to delete file(%s), error(%s)", r.RemoteAddr, filePath, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	api.Infof("[%s] receive delete file request, delete file(%s)", r.RemoteAddr, filePath)
	w.WriteHeader(http.StatusOK)
}
