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
		if err := os.MkdirAll(tmpDir, os.FileMode(755)); err != nil {
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
		return fmt.Errorf("[%d] fail to run cmd, error(%s)", t.ID, err)
	}

	matched, err := regexp.MatchString(DownloadSuccessMatchString, string(stderr.Bytes()))
	if !matched {
		os.Remove(tmpFile)
		return fmt.Errorf("[%d] fail to download, srcIP(%s), file(%s), error(%s)",
			t.ID, t.SrcIP, t.FileName, string(stderr.Bytes()))
	}

	cilog.Successf("[%d] success downloading, file(%s), grade(%d), srcIP(%s)",
		t.ID, t.FileName, t.Grade, t.SrcIP)
	if err := os.Rename(tmpFile, filepath.Join(dl.BaseDir, t.FileName)); err != nil {
		cilog.Errorf("[%d] fail to move file, file(%s), from(%s), to(%s), error(%s)",
			t.ID, t.FileName, tmpDir, dl.BaseDir, err.Error())
	}

	return nil
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
		cilog.Debugf("start to get task, enough disk space, used <= limit(%d)",
			dl.DiskUsageLimitPercent)

		url := fmt.Sprintf("http://%s/tasks", dl.TaskerAddr)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			cilog.Errorf("cannot get task list, error(%s)", err)
			time.Sleep(time.Second * time.Duration(dl.DownloaderSleepSec))
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := dl.HTTPClient.Do(req)
		if err != nil {
			cilog.Errorf("cannot get task list, error(%s)", err)
			time.Sleep(time.Second * time.Duration(dl.DownloaderSleepSec))
			continue
		}

		taskList := make(map[int64]*tasker.Task)
		if err := json.NewDecoder(resp.Body).Decode(&taskList); err != nil {
			cilog.Errorf("cannot get task list, error(%s)", err.Error())
			time.Sleep(time.Second * time.Duration(dl.DownloaderSleepSec))
			continue
		}

		for _, task := range taskList {
			if task.DstAddr == dl.MyAddr {
				if task.Status == tasker.READY {
					if err := dl.reportTask(task, tasker.WORKING); err != nil {
						cilog.Warningf("[%d] fail to report(ready->working), error(%s)",
							task.ID, err.Error())
					}
					return task
				}
			}
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
