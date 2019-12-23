package main

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/castisdev/cfm/tasker"
	"github.com/stretchr/testify/assert"
)

func TestGetTask(t *testing.T) {

	myip := "172.18.0.101"
	// dummy task
	d1ID := int64(111111111111)
	d2ID := int64(222222222222)
	dummyTaskList := map[int64]tasker.Task{

		d1ID: {
			ID:       d1ID,
			Ctime:    222222222,
			Mtime:    333333333,
			Status:   tasker.READY,
			SrcIP:    "172.18.0.101",
			DstIP:    "172.18.0.105",
			FilePath: "/data2/A.mpg",
			FileName: "A.mpg",
			Grade:    1,
		},
		d2ID: {
			ID:       d2ID,
			Ctime:    222222222,
			Mtime:    333333333,
			Status:   tasker.READY,
			SrcIP:    "172.18.0.102",
			DstIP:    myip,
			FilePath: "/data2/B.mpg",
			FileName: "B.mpg",
			Grade:    2,
		},
	}

	// dummy http server
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(&dummyTaskList)
	})

	ds := httptest.NewUnstartedServer(h)
	l, _ := net.Listen("tcp", "127.0.0.1:18883")
	ds.Listener.Close()
	ds.Listener = l
	ds.Start()
	defer ds.Close()

	dl := NewDownloader("/data", myip, "SampleDownloader", "127.0.0.1", 90)

	task := dl.getTask()
	assert.Equal(t, task.ID, d2ID)
	assert.Equal(t, task.Ctime, dummyTaskList[d2ID].Ctime)
	assert.Equal(t, task.Mtime, dummyTaskList[d2ID].Mtime)
	assert.Equal(t, task.Status, dummyTaskList[d2ID].Status)
	assert.Equal(t, task.SrcIP, dummyTaskList[d2ID].SrcIP)
	assert.Equal(t, task.DstIP, dummyTaskList[d2ID].DstIP)
	assert.Equal(t, task.FilePath, dummyTaskList[d2ID].FilePath)
	assert.Equal(t, task.FileName, dummyTaskList[d2ID].FileName)
	assert.Equal(t, task.Grade, dummyTaskList[d2ID].Grade)

}

func TestDownloader_download(t *testing.T) {

	dl1 := NewDownloader("/data", "127.0.0.1", "./SampleDownloader_Fail", "127.0.0.1", 90)
	task := tasker.Task{
		SrcIP:    "127.0.0.1",
		FilePath: "/data3/A.mpg",
		FileName: "A.mpg",
	}
	assert.NotNil(t, dl1.download(&task))

	dl2 := NewDownloader("/data", "127.0.0.1", "./SampleDownloader_Success", "127.0.0.1", 90)
	task = tasker.Task{
		SrcIP:    "127.0.0.1",
		FilePath: "/data3/A.mpg",
		FileName: "A.mpg",
	}
	assert.Nil(t, dl2.download(&task))

}

func TestDownloader_reportTask(t *testing.T) {

	// dummy http server
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ds := httptest.NewUnstartedServer(h)
	l, _ := net.Listen("tcp", "127.0.0.1:18883")
	ds.Listener.Close()
	ds.Listener = l
	ds.Start()
	defer ds.Close()

	dl := NewDownloader("/data", "127.0.0.1", "SampleDownloader", "127.0.0.1:18883", 90)

	task := tasker.Task{
		ID:       11111111111111,
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

func TestDownloader_moveToBaseDir(t *testing.T) {

	//dl := NewDownloader("/data", "127.0.0.1", "SampleNetIODownloader", "127.0.0.1", 90)

	// dst 폴더가 없는 경우 : err:no such file or directory
	// dst 폴더 접근 권한이 없는 경우 : permission denied
	// dst 폴더와 src 폴더의 파티션이 다른 경우 : err:invalid cross-device link
	// dst 폴더에 이미 같은 파일명이 있는 경우 : overwrite
	// if err := dl.moveToBaseDir("/root/a.mpg"); err != nil {
	// 	t.Errorf(err.Error())
	// }
}
