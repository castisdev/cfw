package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cilog"
	"github.com/stretchr/testify/assert"
)

func cfw(cfmaddr string) *httptest.Server {
	api = common.MLogger{
		Logger: cilog.StdLogger(),
		Mod:    "api"}
	router := NewRouter()
	s := &http.Server{
		Addr:         cfmaddr,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	cfw := httptest.NewUnstartedServer(router)
	l, _ := net.Listen("tcp", cfmaddr)
	cfw.Listener.Close()
	cfw.Listener = l
	cfw.Config = s

	return cfw
}

func request(method, url string, body []byte) (int, []byte, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: time.Second * 10}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	responsebody, _ := ioutil.ReadAll(resp.Body)

	return resp.StatusCode, responsebody, nil
}

func createfile(dir string, filename string) {
	fp := filepath.Join(dir, filename)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
			log.Fatal(err)
		}
	}
	f, err := os.OpenFile(fp, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
}

func deletefile(dir string, filename string) {
	dir = filepath.Clean(dir)
	if dir == "." || dir == ".." {
		log.Fatal(errors.New("do not delete current or parent folder"))
	}
	fp := filepath.Join(dir, filename)
	err := os.RemoveAll(fp)
	if err != nil {
		log.Fatal(err)
	}
}

func TestHearbeat(t *testing.T) {
	cfwaddr := "127.0.0.1:18883"
	cfw := cfw(cfwaddr)
	cfw.Start()
	defer cfw.Close()

	url := fmt.Sprintf("http://%s/hb", cfwaddr)
	rc, _, err := request(http.MethodHead, url, nil)
	if err != nil {
		t.Errorf("fail to heartbeat, error(%s)", err.Error())
		return
	}
	if rc != http.StatusOK {
		t.Errorf("fail to heartbeat, response(%s)", http.StatusText(rc))
		return
	}

}

func TestGetFileList(t *testing.T) {
	cfwaddr := "127.0.0.1:18883"
	cfw := cfw(cfwaddr)
	cfw.Start()
	defer cfw.Close()

	// main packager의 global variable
	config = &Config{}
	config.BaseDir = "testfolder"

	f1 := "test1.mpg"
	createfile(config.BaseDir, f1)
	f2 := "test2.mpg"
	createfile(config.BaseDir, f2)
	f3 := "test3.mpg"
	createfile(config.BaseDir, f3)
	defer deletefile(config.BaseDir, "")

	url := fmt.Sprintf("http://%s/files", cfwaddr)
	rc, body, err := request(http.MethodGet, url, nil)
	if err != nil {
		t.Errorf("fail to get file list, error(%s)", err.Error())
		return
	}
	if rc != http.StatusOK {
		t.Errorf("fail to get file list, response(%s)", http.StatusText(rc))
		return
	}

	filelist := make([]string, 0)
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		filelist = append(filelist, scanner.Text())
	}

	// testfolder 안의 파일 개수
	// test1.mpg
	// test2.mpg
	// test3.mpg
	t.Log(filelist)
	assert.Equal(t, 3, len(filelist))
	assert.Contains(t, filelist, "test1.mpg")
	assert.Contains(t, filelist, "test2.mpg")
	assert.Contains(t, filelist, "test3.mpg")
}

func TestGetDiskUsage(t *testing.T) {
	cfwaddr := "127.0.0.1:18883"
	cfw := cfw(cfwaddr)
	cfw.Start()
	defer cfw.Close()

	// main packager의 global variable
	config = &Config{}
	config.BaseDir = "."

	url := fmt.Sprintf("http://%s/df", cfwaddr)
	rc, body, err := request(http.MethodGet, url, nil)
	if err != nil {
		t.Errorf("fail to get disk usage, error(%s)", err.Error())
		return
	}
	if rc != http.StatusOK {
		t.Errorf("fail to get disk usage, response(%s)", http.StatusText(rc))
		return
	}
	du := common.DiskUsage{}
	err = json.Unmarshal(body, &du)
	if err != nil {
		t.Errorf("fail to get disk usage, error(%s)", err.Error())
		return
	}
	t.Log(du)
}

func TestDeleteFile(t *testing.T) {
	cfwaddr := "127.0.0.1:18883"
	cfw := cfw(cfwaddr)
	cfw.Start()
	defer cfw.Close()

	// main packager의 global variable
	config = &Config{}
	config.BaseDir = "testfolder"

	fn := "test.mpg"
	wrongfn := "wrongfile.mpg"
	createfile(config.BaseDir, fn)
	defer deletefile(config.BaseDir, "")

	// 없는 파일 삭제 요청
	url := fmt.Sprintf("http://%s/files/%s", cfwaddr, wrongfn)
	rc, _, err := request(http.MethodDelete, url, nil)
	if err != nil {
		t.Errorf("fail to delete file, error(%s)", err.Error())
		return
	}
	assert.Equal(t, rc, http.StatusNoContent)

	//있는 파일 삭제 요청
	url = fmt.Sprintf("http://%s/files/%s", cfwaddr, fn)
	rc, _, err = request(http.MethodDelete, url, nil)
	if err != nil {
		t.Errorf("fail to delete file, error(%s)", err.Error())
		return
	}
	assert.Equal(t, rc, http.StatusOK)
}
