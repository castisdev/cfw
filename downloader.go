package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/castisdev/cfm/tasker"
	"github.com/castisdev/cilog"
)

// Downloader :
type Downloader struct {
	HTTPClient            *http.Client
	BaseDir               string
	MyIP                  string
	DownloaderBin         string
	TaskerAddr            string
	DiskUsageLimitPercent int
}

// NewDownloader :
func NewDownloader(baseDir string, myIP string, downloaderBinPath string, taskerIP string, usageLimit int) *Downloader {

	return &Downloader{
		HTTPClient:            &http.Client{Timeout: time.Second * 10},
		BaseDir:               baseDir,
		MyIP:                  myIP,
		DownloaderBin:         downloaderBinPath,
		TaskerAddr:            taskerIP,
		DiskUsageLimitPercent: usageLimit,
	}
}

// RunForever :
func (dl *Downloader) RunForever() {

	for {

		usedPercent, err := dl.getDiskUsagePercent()
		if err != nil {
			cilog.Errorf("fail to get disk usage percent,error(%s)", err.Error())
			time.Sleep(time.Second * 3)
			continue
		}

		if usedPercent > dl.DiskUsageLimitPercent {
			cilog.Warningf("not enough storage,used(%d),limit(%d)", usedPercent, dl.DiskUsageLimitPercent)
			time.Sleep(time.Second * 3)
			continue
		}
		cilog.Debugf("get disk usage percent(%d),limit(%d)", usedPercent, dl.DiskUsageLimitPercent)

		task := dl.getTask()
		if err := dl.download(task); err != nil {
			cilog.Errorf(err.Error())
		}

		if err := dl.reportTask(task, tasker.DONE); err != nil {
			cilog.Warningf("fail to report(working->done),error(%s)", err.Error())
		}
	}
}

func (dl *Downloader) download(t *tasker.Task) error {

	tmpDir := dl.BaseDir + "/NetIOTemp"
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		if err := os.MkdirAll(tmpDir, os.FileMode(755)); err != nil {
			return err
		}
	}
	tmpFile := tmpDir + "/" + t.FileName + "." + strconv.FormatInt(t.ID, 10)
	cilog.Infof("start to download,file(%s),grade(%d),srcIP(%s),copySpeed(%s)", t.FileName, t.Grade, t.SrcIP, t.CopySpeed)
	cilog.Debugf("cmd (%s %s %s %s %s)", dl.DownloaderBin, t.SrcIP, tmpFile, t.FilePath, t.CopySpeed)

	cmd := exec.Command(dl.DownloaderBin, t.SrcIP, tmpFile, t.FilePath, t.CopySpeed)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("fail to run cmd : error(%s)", err)
	}

	matched, err := regexp.MatchString("Successfully", string(stderr.Bytes()))
	if !matched {
		os.Remove(tmpFile)
		return fmt.Errorf("fail to download,srcIP(%s),file(%s),error(%s)", t.SrcIP, t.FileName, string(stderr.Bytes()))
	}

	cilog.Successf("success to download,file(%s),grade(%d),srcIP(%s)", t.FileName, t.Grade, t.SrcIP)
	if err := os.Rename(tmpFile, dl.BaseDir+"/"+t.FileName); err != nil {
		cilog.Errorf("fail to move file,file(%s),from(%s),to(%s),error(%s)", t.FileName, tmpDir, dl.BaseDir, err.Error())
	}

	return nil
}

func (dl *Downloader) getTask() *tasker.Task {

	for {

		url := fmt.Sprintf("http://%s/tasks", dl.TaskerAddr)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			cilog.Errorf("cannot get task list,error(%s)", err)
			time.Sleep(time.Second * 5)
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := dl.HTTPClient.Do(req)
		if err != nil {
			cilog.Errorf("cannot get task list,error(%s)", err)
			time.Sleep(time.Second * 5)
			continue
		}

		taskList := make(map[int64]*tasker.Task)
		if err := json.NewDecoder(resp.Body).Decode(&taskList); err != nil {
			cilog.Errorf("cannot get task list,error(%s)", err.Error())
			time.Sleep(time.Second * 5)
			continue
		}

		for _, task := range taskList {
			if task.DstIP == dl.MyIP {
				if task.Status == tasker.READY {
					if err := dl.reportTask(task, tasker.WORKING); err != nil {
						cilog.Warningf("fail to report(ready->working),error(%s)", err.Error())
					}
					return task
				}
			}
		}

		cilog.Debugf("no task")
		time.Sleep(time.Second * 5)
	}
}

func (dl *Downloader) reportTask(t *tasker.Task, s tasker.Status) error {

	st := struct {
		Status tasker.Status `json:"status"`
	}{
		Status: s,
	}

	body, err := json.Marshal(&st)

	url := fmt.Sprintf("http://%s/tasks/%d", dl.TaskerAddr, t.ID)
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := dl.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	return fmt.Errorf("%d", resp.StatusCode)
}

func (dl *Downloader) getDiskUsagePercent() (int, error) {

	fs := syscall.Statfs_t{}
	if err := syscall.Statfs(dl.BaseDir, &fs); err != nil {
		return -1, err
	}

	t := int64(fs.Blocks * uint64(fs.Bsize))
	f := int64(fs.Bfree * uint64(fs.Bsize))
	u := t - f
	usedPercent := int((u*100/t*100)/100 + 1)

	return usedPercent, nil
}
