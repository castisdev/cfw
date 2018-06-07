package main

import (
	"testing"
)

func TestFileList(t *testing.T) {
}

func TestDiskUsage(t *testing.T) {

	total := 249779191808
	used := 189656698880

	t.Log(total)
	t.Log(used)
	t.Log((used * 100 / total * 100) / 100)
	t.Error("error")

}

func TestDeleteFile(t *testing.T) {
}
