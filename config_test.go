package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

func setupTestDir(t *testing.T, testName string) string {
	dir, err := os.MkdirTemp("", testName)

	if err != nil {
		t.Error(err)
	}
	return dir
}

func writeToFiles(t *testing.T, dir string, num, base int) {
	for i := 0; i < num; i++ {
		fileName := filepath.Join(dir, fmt.Sprintf("%d.yaml", i))
		fo, err := os.Create(fileName)
		if err != nil {
			t.Error(err)
		}
		fo.Write([]byte(fmt.Sprintf("val-%d: %d\n", i, i+base)))
		fo.Close()
	}

}

func dumpConfig(t *testing.T, v *viper.Viper) {
	settings := v.AllSettings()
	bs, err := yaml.Marshal(settings)
	if err != nil {
		t.Error(err)
	}
	t.Log(string(bs))
}

func TestGlob(t *testing.T) {
	dir := setupTestDir(t, "glob")
	if dir == "" {
		t.Error("setup failed")
	}
	writeToFiles(t, dir, 3, 1)
	vpCh, erCh := NewFromPathsAndGlob([]string{dir}, "*.yaml")
	if vpCh == nil || erCh == nil {
		t.Error("NewFromPathsAndGlob")
	}

	defer os.RemoveAll(dir)
	time.AfterFunc(3*time.Second, func() {
		writeToFiles(t, dir, 3, 7)
	})

	done := make(chan bool, 1)
	time.AfterFunc(5*time.Second, func() {
		done <- true
	})

	changes := 0

	for {
		select {
		case <-done:
			if changes != 4 { // 1 for the initial setup and then one for each modification
				t.Error("changes count error", changes)
			}
			return
		case vp := <-vpCh:
			dumpConfig(t, vp)
			changes++
		case err := <-erCh:
			t.Error(err)
		}
	}
}
