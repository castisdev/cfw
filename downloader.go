package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cfm/tasker"
	"github.com/castisdev/cilog"
)

// Downloader :
type Downloader struct {
	HTTPClient            *http.Client
	BaseDir               string
	MyAddr                string
	DownloaderBin         string
	TaskerAddr            string
	DiskUsageLimitPercent uint
	DownloaderSleepSec    uint
}

// BaseDir 아래에 만들어지는 temp dir 이름
const TempDir string = "NetIOTemp"

// download binary가 실행되고, stderr로 내보내는 메시지 중
// 성공 여부를 판단할 수 있는 문자열
const DownloadSuccessMatchString = "Successfully"

// NewDownloader :
// baseDir:
// myAddr: ip:port
// downloaderBinPath :
// taskerAddr :
// usageLimit :
// sleepSEc : task 검색 후 sleep(초)
func NewDownloader(baseDir string, myAddr string, downloaderBinPath string,
	taskerAddr string, usageLimit uint, sleepSec uint) *Downloader {

	return &Downloader{
		HTTPClient:            &http.Client{Timeout: time.Second * 10},
		BaseDir:               baseDir,
		MyAddr:                myAddr,
		DownloaderBin:         downloaderBinPath,
		TaskerAddr:            taskerAddr,
		DiskUsageLimitPercent: usageLimit,
		DownloaderSleepSec:    sleepSec,
	}
}

// disk usage를 구해서 limit 넘었는지 반환
// disk usage를 구할 수 없는 경우 false 반환
func (dl *Downloader) checkEnoughDiskSpace() bool {
	du, err := common.GetDiskUsage(dl.BaseDir)
	if err != nil {
		cilog.Errorf("check disk space, fail to get disk usage percent, error(%s)",
			err.Error())
		return false
	}
	if du.UsedPercent > dl.DiskUsageLimitPercent {
		cilog.Warningf("check disk space, not enough disk space, used(%d) > limit(%d)",
			du.UsedPercent, dl.DiskUsageLimitPercent)
		return false
	}

	cilog.Debugf("check disk space, enough disk space, used(%d) <= limit(%d)",
		du.UsedPercent, dl.DiskUsageLimitPercent)
	return true
}

// RunForever :
func (dl *Downloader) RunForever() {
	dl.removeTempdir()

	for {
		task := dl.waitTask()
		if err := dl.download(task); err != nil {
			cilog.Errorf(err.Error())
		}

		if err := dl.reportTask(task, tasker.DONE); err != nil {
			cilog.Warningf("after download, fail to report(working->done), error(%s)",
				err.Error())
		}
	}
}

func (dl *Downloader) removeTempdir() {
	tmpDir := filepath.Join(dl.BaseDir, TempDir)
	if err := os.RemoveAll(tmpDir); err != nil {
		cilog.Errorf("fail to remove temp dir(%s), error(%s)", tmpDir, err.Error())
	} else {
		cilog.Infof("remove temp dir(%s)", tmpDir)
	}
}

func (dl *Downloader) download(t *tasker.Task) error {

	tmpDir := filepath.Join(dl.BaseDir, TempDir)
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		if err := os.MkdirAll(tmpDir, os.FileMode(0755)); err != nil {
			return err
		}
	}
	tmpFile := filepath.Join(tmpDir, t.FileName+"."+strconv.FormatInt(t.ID, 10))
	cilog.Infof("[%d] start to download, file(%s), grade(%d), srcIP(%s), copySpeed(%s)",
		t.ID, t.FileName, t.Grade, t.SrcIP, t.CopySpeed)
	cilog.Debugf("cmd (%s %s %s %s %s)",
		dl.DownloaderBin, t.SrcIP, tmpFile, t.FilePath, t.CopySpeed)

	cmd := exec.Command(dl.DownloaderBin, t.SrcIP, tmpFile, t.FilePath, t.CopySpeed)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("[%d] fail to run cmd, error(%s)", t.ID, err.Error())
	}

	matched, err := regexp.MatchString(DownloadSuccessMatchString, string(stderr.Bytes()))
	if !matched {
		os.Remove(tmpFile)
		return fmt.Errorf("[%d] fail to download, srcIP(%s), file(%s), error(%s)",
			t.ID, t.SrcIP, t.FileName, string(stderr.Bytes()))
	}
	fileNamePath := filepath.Join(dl.BaseDir, t.FileName)
	if err := os.Rename(tmpFile, fileNamePath); err != nil {
		return fmt.Errorf("[%d] fail to move file, file(%s), from(%s), to(%s), error(%s)",
			t.ID, t.FileName, tmpDir, dl.BaseDir, err.Error())
	}

	cilog.Infof("[%d] success downloading, filePath(%s), file(%s), grade(%d), srcIP(%s)",
		t.ID, fileNamePath, t.FileName, t.Grade, t.SrcIP)
	return nil
}

// getTask
//
// cfm 의 task list 중에서
//
// task.status가 READY이고,
//
// task.DstAddr가 cfw의 addr인 첫번째 task를 하나 찾아냄
func (dl *Downloader) getTask() (*tasker.Task, bool) {
	url := fmt.Sprintf("http://%s/tasks", dl.TaskerAddr)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		cilog.Errorf("cannot get task list, error(%s)", err.Error())
		return nil, false
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := dl.HTTPClient.Do(req)
	if err != nil {
		cilog.Errorf("cannot get task list, error(%s)", err.Error())
		return nil, false
	}
	taskList := make([]tasker.Task, 0)
	if err := json.NewDecoder(resp.Body).Decode(&taskList); err != nil {
		cilog.Errorf("cannot get task list, error(%s)", err.Error())
		return nil, false
	}
	for _, t := range taskList {
		if t.DstAddr == dl.MyAddr {
			if t.Status == tasker.READY {
				myTask := tasker.NewTaskFrom(t)
				return myTask, true
			}
		}
	}
	return nil, false
}

func (dl *Downloader) waitTask() *tasker.Task {

	for {
		ok := dl.checkEnoughDiskSpace()
		if !ok {
			cilog.Debugf("skip getting task, not enough disk space, used > limit(%d)",
				dl.DiskUsageLimitPercent)
			time.Sleep(time.Second * time.Duration(dl.DownloaderSleepSec))
			continue
		}
		// cilog.Debugf("start to get task, enough disk space, used <= limit(%d)",
		// 	dl.DiskUsageLimitPercent)
		myTask, ok := dl.getTask()
		if !ok {
			time.Sleep(time.Second * time.Duration(dl.DownloaderSleepSec))
			continue
		}

		if err := dl.reportTask(myTask, tasker.WORKING); err != nil {
			cilog.Warningf("[%d] fail to report(ready->working), error(%s)",
				myTask.ID, err.Error())
		}
		cilog.Debugf("get no task")
		time.Sleep(time.Second * time.Duration(dl.DownloaderSleepSec))
	}
}

func (dl *Downloader) reportTask(t *tasker.Task, s tasker.Status) error {

	st := struct {
		Status tasker.Status `json:"status"`
	}{
		Status: s,
	}

	body, err := json.Marshal(&st)
	if err != nil {
		return err
	}

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
