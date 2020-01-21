package main

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cfm/tasker"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func cfm(cfmaddr string) *httptest.Server {
	router := mux.NewRouter().StrictSlash(true)
	router.Methods("PATCH").Path("/tasks/{taskId}").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var s struct {
				Status tasker.Status `json:"status"`
			}
			if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			defer r.Body.Close()
			w.WriteHeader(http.StatusOK)
			return
		})

	router.Methods("GET").Path("/tasks").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			destip := "127.0.0.1"
			destaddr := "127.0.0.1:8889"
			// dummy task
			d1ID := int64(111111111111)
			d2ID := int64(222222222222)
			dummyTaskList := []tasker.Task{
				{
					ID:       d1ID,
					Ctime:    222222222,
					Mtime:    333333333,
					Status:   tasker.READY,
					SrcIP:    "172.18.0.101",
					DstIP:    "172.18.0.105",
					FilePath: "/data2/A.mpg",
					FileName: "A.mpg",
					Grade:    1,
					SrcAddr:  "172.18.0.101:9888",
					DstAddr:  "172.18.0.105:7889",
				},
				{
					ID:       d2ID,
					Ctime:    222222222,
					Mtime:    333333333,
					Status:   tasker.READY,
					SrcIP:    "172.18.0.102",
					DstIP:    destip,
					FilePath: "/data2/B.mpg",
					FileName: "B.mpg",
					Grade:    2,
					SrcAddr:  "172.18.0.102:9888",
					DstAddr:  destaddr,
				},
			}
			json.NewEncoder(w).Encode(&dummyTaskList)
		})
	s := &http.Server{
		Addr:         cfmaddr,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	cfm := httptest.NewUnstartedServer(router)
	l, _ := net.Listen("tcp", cfmaddr)
	cfm.Listener.Close()
	cfm.Listener = l
	cfm.Config = s

	return cfm
}

func TestWaitTask(t *testing.T) {
	myaddr := "127.0.0.1:8889"
	cfmaddr := "127.0.0.1:18883"
	cfm := cfm(cfmaddr)
	cfm.Start()
	defer cfm.Close()

	expectedtaskID := int64(222222222222)
	destip := "127.0.0.1"
	destaddr := "127.0.0.1:8889"
	successmatch := "Successfully"
	expectedtask := tasker.Task{
		ID:       expectedtaskID,
		Ctime:    222222222,
		Mtime:    333333333,
		Status:   tasker.READY,
		SrcIP:    "172.18.0.102",
		DstIP:    destip,
		FilePath: "/data2/B.mpg",
		FileName: "B.mpg",
		Grade:    2,
		SrcAddr:  "172.18.0.102:9888",
		DstAddr:  destaddr,
	}

	dl := NewDownloader(".", myaddr, "SampleDownloader", cfmaddr, 100, 1,
		successmatch)

	task := dl.waitTask()
	t.Logf("found task:%s", task)

	assert.Equal(t, task.ID, expectedtask.ID)
	assert.Equal(t, task.Ctime, expectedtask.Ctime)
	assert.Equal(t, task.Mtime, expectedtask.Mtime)
	assert.Equal(t, task.Status, expectedtask.Status)
	assert.Equal(t, task.SrcIP, expectedtask.SrcIP)
	assert.Equal(t, task.DstIP, expectedtask.DstIP)
	assert.Equal(t, task.FilePath, expectedtask.FilePath)
	assert.Equal(t, task.FileName, expectedtask.FileName)
	assert.Equal(t, task.Grade, expectedtask.Grade)
}

func TestGetTask(t *testing.T) {
	myaddr := "127.0.0.1:8889"
	cfmaddr := "127.0.0.1:18883"
	cfm := cfm(cfmaddr)
	cfm.Start()
	defer cfm.Close()

	expectedtaskID := int64(222222222222)
	destip := "127.0.0.1"
	destaddr := "127.0.0.1:8889"
	expectedtask := tasker.Task{
		ID:       expectedtaskID,
		Ctime:    222222222,
		Mtime:    333333333,
		Status:   tasker.READY,
		SrcIP:    "172.18.0.102",
		DstIP:    destip,
		FilePath: "/data2/B.mpg",
		FileName: "B.mpg",
		Grade:    2,
		SrcAddr:  "172.18.0.102:9888",
		DstAddr:  destaddr,
	}

	successmatch := "Successfully"
	dl := NewDownloader("/data", myaddr, "SampleDownloader", cfmaddr, 90, 1,
		successmatch)
	task, ok := dl.getTask()
	if ok {
		t.Logf("found task:%s", task)
	}

	assert.Equal(t, ok, true)
	assert.Equal(t, task.ID, expectedtask.ID)
	assert.Equal(t, task.Ctime, expectedtask.Ctime)
	assert.Equal(t, task.Mtime, expectedtask.Mtime)
	assert.Equal(t, task.Status, expectedtask.Status)
	assert.Equal(t, task.SrcIP, expectedtask.SrcIP)
	assert.Equal(t, task.DstIP, expectedtask.DstIP)
	assert.Equal(t, task.FilePath, expectedtask.FilePath)
	assert.Equal(t, task.FileName, expectedtask.FileName)
	assert.Equal(t, task.Grade, expectedtask.Grade)

	cfw2addr := "127.0.0.3:8889"
	cfw2 := NewDownloader("/data", cfw2addr, "SampleDownloader", cfmaddr, 90, 1,
		successmatch)
	task2, ok := cfw2.getTask()
	if ok {
		t.Logf("found task2:%s", task2)
	}
	assert.Equal(t, ok, false)
}

func TestDownloader_download(t *testing.T) {
	myaddr := "127.0.0.1:8889"
	cfmaddr := "127.0.0.1:18883"
	testfilename := "A.mpg"
	successmatch := "Successfully"

	defer os.RemoveAll(TempDir)
	defer os.RemoveAll(testfilename)

	dl1 := NewDownloader(".", myaddr,
		"script//SampleDownloader_Fail", cfmaddr, 90, 1, successmatch)
	task := tasker.Task{
		SrcIP:     "127.0.0.1",
		FilePath:  "/data3/FAIL1.mpg",
		FileName:  testfilename,
		CopySpeed: "10000000",
	}
	err := dl1.download(&task)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "fail to download")
	t.Logf(err.Error())

	dl2 := NewDownloader(".", myaddr,
		"script/SampleDownloader_Success", cfmaddr, 90, 1, successmatch)
	task = tasker.Task{
		SrcIP:     "127.0.0.1",
		FilePath:  "/data3/FAIL2.mpg",
		FileName:  testfilename,
		CopySpeed: "20000000",
	}
	err2 := dl2.download(&task)
	assert.NotNil(t, err2)
	assert.Contains(t, err2.Error(), "fail to move file")
	t.Logf(err2.Error())

	dl3 := NewDownloader(".", myaddr,
		"script/pseudo_downloader.sh", cfmaddr, 90, 1, successmatch)
	task = tasker.Task{
		SrcIP:     "127.0.0.1",
		FilePath:  "/data3/SUCCESS1.mpg",
		FileName:  testfilename,
		CopySpeed: "30000000",
	}
	err3 := dl3.download(&task)
	assert.Nil(t, err3)

	successmatch = "hello"
	dl4 := NewDownloader(".", myaddr,
		"script/pseudo_downloader.sh", cfmaddr, 90, 1, successmatch)
	task = tasker.Task{
		SrcIP:     "127.0.0.1",
		FilePath:  "/data3/FAIL3.mpg",
		FileName:  testfilename,
		CopySpeed: "30000000",
	}
	err4 := dl4.download(&task)
	assert.NotNil(t, err4)
	t.Logf(err4.Error())

	successmatch = "successful"
	dl5 := NewDownloader(".", myaddr,
		"script/sample_scp_downloader.sh", cfmaddr, 90, 1, successmatch)
	task = tasker.Task{
		SrcIP:     "127.0.0.1",
		FilePath:  "/data1/SCP.mpg",
		FileName:  testfilename,
		CopySpeed: "30000000",
	}
	err5 := dl5.download(&task)
	assert.NotNil(t, err5)
	t.Logf(err5.Error())
}

func TestDownloader_reportTaskStatusDone(t *testing.T) {
	requestTaskId := int64(11111111111111)
	cfmaddr := "127.0.0.1:18883"
	myaddr := "127.0.0.1:8889"

	// dummy http server
	router := mux.NewRouter().StrictSlash(true)
	router.Methods("PATCH").Path("/tasks/{taskId}").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			taskID := vars["taskId"]
			ID, err := strconv.ParseInt(taskID, 10, 64)
			t.Logf("taskId: %d:", ID)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if requestTaskId == ID {
				var s struct {
					Status tasker.Status `json:"status"`
				}
				if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				defer r.Body.Close()

				t.Logf("staus: %s", s.Status)
				if s.Status == tasker.DONE {
					w.WriteHeader(http.StatusOK)
					return
				} else {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)
		})
	s := &http.Server{
		Addr:         cfmaddr,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	successmatch := "Successfully"
	ds := httptest.NewUnstartedServer(router)
	l, _ := net.Listen("tcp", cfmaddr)
	ds.Listener.Close()
	ds.Listener = l
	ds.Config = s
	ds.Start()
	defer ds.Close()

	dl := NewDownloader(".", myaddr,
		"SampleDownloader", cfmaddr, 90, 1, successmatch)

	task := tasker.Task{
		ID:       requestTaskId,
		Ctime:    222222222,
		Mtime:    333333333,
		Status:   tasker.READY,
		SrcIP:    "172.18.0.102",
		DstIP:    "127.0.0.1",
		FilePath: "/data2/B.mpg",
		FileName: "B.mpg",
		Grade:    2,
	}
	assert.Nil(t, dl.reportTask(&task, tasker.DONE))
}

func TestCheckEnoughDiskSpace(t *testing.T) {
	cfmaddr := "127.0.0.1:18883"
	myaddr := "127.0.0.1:8889"
	sleepsec := uint(1)
	basedir := "."
	successmatch := "Successfully"
	du, err := common.GetDiskUsage(basedir)
	if err != nil {
		assert.Error(t, err)
	}
	var limitpercent uint
	var dl *Downloader
	var ok bool

	limitpercent = 100
	dl = NewDownloader(basedir, myaddr,
		"SampleDownloader", cfmaddr, limitpercent, sleepsec, successmatch)
	ok = dl.checkEnoughDiskSpace()
	assert.Equal(t, true, ok)

	limitpercent = du.UsedPercent
	dl = NewDownloader(basedir, myaddr,
		"SampleDownloader", cfmaddr, limitpercent, sleepsec, successmatch)
	ok = dl.checkEnoughDiskSpace()
	assert.Equal(t, true, ok)

	limitpercent = du.UsedPercent + uint(10)
	dl = NewDownloader(basedir, myaddr,
		"SampleDownloader", cfmaddr, limitpercent, sleepsec, successmatch)
	ok = dl.checkEnoughDiskSpace()
	assert.Equal(t, true, ok)

	limitpercent = 0
	dl = NewDownloader(basedir, myaddr,
		"SampleDownloader", cfmaddr, limitpercent, sleepsec, successmatch)
	ok = dl.checkEnoughDiskSpace()
	assert.Equal(t, false, ok)

	limitpercent = du.UsedPercent - uint(10)
	dl = NewDownloader(basedir, myaddr,
		"SampleDownloader", cfmaddr, limitpercent, sleepsec, successmatch)
	ok = dl.checkEnoughDiskSpace()
	assert.Equal(t, false, ok)
}
