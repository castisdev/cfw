package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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
	HTTPClient                 *http.Client
	BaseDir                    string
	MyAddr                     string
	DownloaderBin              string
	TaskerAddr                 string
	DiskUsageLimitPercent      uint
	DownloaderSleepSec         uint
	DownloadSuccessMatchString string
}

// BaseDir 아래에 만들어지는 temp dir 이름
const TempDir string = "NetIOTemp"

// NewDownloader :
//
// baseDir: 파일 download direcotry
//
// myAddr: ip:port
//
// downloaderBinPath : download 하기 위한 실행파일이름
//
// taskerAddr : cfm 주소 (ip:port)
//
// usageLimit : download 하기위해 cfw 의 disk limit 사용량(0 <= percent <= 100)
//
// sleepSec : task 검색 후 없을 때 sleep 하는 시간(초)
//
// DownloadSuccessMatchString : download binary가 실행되고, stderr로 내보내는 메시지 중
// 성공 여부를 판단할 수 있는 문자열
func NewDownloader(baseDir string, myAddr string, downloaderBinPath string,
	taskerAddr string, usageLimit uint, sleepSec uint,
	downloadSuccessMatchString string) *Downloader {
	if usageLimit > 100 {
		usageLimit = 100
	}
	baseDir = filepath.Clean(baseDir)
	downloaderBinPath = filepath.Clean(downloaderBinPath)

	return &Downloader{
		HTTPClient:                 &http.Client{Timeout: time.Second * 10},
		BaseDir:                    baseDir,
		MyAddr:                     myAddr,
		DownloaderBin:              downloaderBinPath,
		TaskerAddr:                 taskerAddr,
		DiskUsageLimitPercent:      usageLimit,
		DownloaderSleepSec:         sleepSec,
		DownloadSuccessMatchString: downloadSuccessMatchString,
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
			cilog.Warningf("[%d] after download, fail to report(working->done), error(%s)",
				task.ID, err.Error())
		} else {
			cilog.Infof("[%d] report(working->done)", task.ID)
		}
	}
}

func (dl *Downloader) removeTempdir() {
	tmpDir := filepath.Join(dl.BaseDir, TempDir)
	tmpDir = filepath.Clean(tmpDir)
	if err := os.RemoveAll(tmpDir); err != nil {
		cilog.Errorf("fail to remove temp dir(%s), error(%s)", tmpDir, err.Error())
	} else {
		cilog.Infof("remove temp dir(%s)", tmpDir)
	}
}

func (dl *Downloader) download(t *tasker.Task) error {
	tmpDir := filepath.Join(dl.BaseDir, TempDir)
	tmpDir = filepath.Clean(tmpDir)
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		if err := os.MkdirAll(tmpDir, os.FileMode(0755)); err != nil {
			return err
		}
	}
	tmpFile := filepath.Join(tmpDir, t.FileName+"."+strconv.FormatInt(t.ID, 10))
	tmpFile = filepath.Clean(tmpFile)
	targetFileNamePath := filepath.Join(dl.BaseDir, t.FileName)

	dltaskdesc := fmt.Sprintf("targetFilePath(%s), srcIP(%s), srcFilePath(%s), grade(%d), bps(%s)",
		targetFileNamePath, t.SrcIP, t.FilePath, t.Grade, t.CopySpeed)
	cilog.Infof("[%d] start to download, %s", t.ID, dltaskdesc)

	cmddesc := fmt.Sprintf("cmd(%s), srcIP(%s), targetFilePath(%s), srcFilePath(%s), bps(%s)",
		dl.DownloaderBin, t.SrcIP, tmpFile, t.FilePath, t.CopySpeed)
	cilog.Infof("[%d] %s", t.ID, cmddesc)

	cmd := exec.Command(dl.DownloaderBin, t.SrcIP, tmpFile, t.FilePath, t.CopySpeed)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("[%d] fail to download, %s, fail to run %s, error(%s)",
			t.ID, dltaskdesc, cmddesc, err.Error())
	}

	matched, err := regexp.MatchString(dl.DownloadSuccessMatchString, string(stderr.Bytes()))
	if !matched {
		os.Remove(tmpFile)
		return fmt.Errorf("[%d] fail to download, %s, fail to match(%s) in stderr(%s)",
			t.ID, dltaskdesc, dl.DownloadSuccessMatchString, string(stderr.Bytes()))
	}

	if err := os.Rename(tmpFile, targetFileNamePath); err != nil {
		return fmt.Errorf("[%d] fail to download, %s, fail to move file(%s) to(%s), error(%s)",
			t.ID, dltaskdesc, tmpFile, targetFileNamePath, err.Error())
	}

	cilog.Infof("[%d] download, %s", t.ID, dltaskdesc)
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
		cilog.Errorf("fail to get task list, error(%s)", err.Error())
		return nil, false
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := dl.HTTPClient.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		cilog.Errorf("fail to get task list, error(%s)", err.Error())
		return nil, false
	}
	taskList := make([]tasker.Task, 0)

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&taskList); err != nil {
		cilog.Errorf("fail to get task list, error(%s)", err.Error())
		return nil, false
	}
	if dec.More() {
		// there's more data in the stream, so discard whatever is left
		io.Copy(ioutil.Discard, resp.Body)
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
			//cilog.Debugf("get no task")
			time.Sleep(time.Second * time.Duration(dl.DownloaderSleepSec))
			continue
		}

		cilog.Infof("[%d] find a task(%s)", myTask.ID, *myTask)
		if err := dl.reportTask(myTask, tasker.WORKING); err != nil {
			cilog.Warningf("[%d] fail to report(ready->working), error(%s)",
				myTask.ID, err.Error())
		} else {
			cilog.Infof("[%d] report(ready->working)", myTask.ID)
		}
		return myTask
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
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}
	io.Copy(ioutil.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%d", resp.StatusCode)
	}

	return nil
}
