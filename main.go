package main

import (
	"bufio"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"sync"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

func runCmd(command string) (string, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	out, err := cmd.CombinedOutput()
	return string(out), err
}

func openFile(cmd string) (*os.File, error) {
	filename := fmt.Sprintf(".%x.conf", md5.Sum([]byte(cmd)))
	fp, err := os.Open(filename)
	if err == nil {
		return fp, err
	}

	fp, _ = os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0644)
	return fp, fmt.Errorf("file created")
}

func loadStatuses(fp *os.File) (map[string]string, error) {
	var statuses map[string]string

	b, err := ioutil.ReadAll(fp)
	if err != nil {
		return map[string]string{}, err
	}

	err = json.Unmarshal(b, &statuses)
	return statuses, err
}

func stateLog(state, arg string) string {
	return fmt.Sprintf("%s\t%s", state, arg)
}

func main() {
	var app = kingpin.New("stated-cmd", "stated command runner")

	var run = app.Command("run", "run command")
	var cmd = run.Flag("cmd", "command").Required().String()
	var con = run.Flag("concurrency", "concurrency").Short('c').Default("1").Int()

	var fail = app.Command("fail", "failed list")
	var file = fail.Flag("file", "state file").Short('f').Required().String()

	app.Version("0.1.0")

	subcmd := kingpin.MustParse(app.Parse(os.Args[1:]))

	if subcmd == "fail" {
		fp, err := os.Open(*file)
		if err != nil {
			log.Fatal(err)
		}
		defer fp.Close()
		statuses, err := loadStatuses(fp)
		if err != nil {
			log.Fatal(err)
		}

		for arg, state := range statuses {
			if state == "fail" {
				fmt.Println(arg)
			}
		}

		os.Exit(0)
	}

	chSize := *con
	if chSize < 0 {
		chSize = 0
	}

	conChan := make(chan struct{}, chSize)
	var wg sync.WaitGroup
	var mu sync.Mutex

	var filename string
	fp, err := openFile(*cmd)
	filename = fp.Name()
	var isCreated bool
	if err != nil {
		fmt.Println(fmt.Sprintf("create %s", fp.Name()))
		isCreated = true
	} else {
		fmt.Println(fmt.Sprintf("load %s", fp.Name()))
	}

	var statuses map[string]string

	if !isCreated {
		statuses, err = loadStatuses(fp)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		statuses = make(map[string]string)
	}

	var jb []byte
	defer func() {
		fp.Close()

		if len(jb) > 0 {
			fp, err = os.OpenFile(filename, os.O_TRUNC|os.O_WRONLY, 0644)
			_, err = fp.Write(jb)
			if err != nil {
				log.Fatal(err)
			}
			fp.Close()
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		wg.Add(1)
		go func(arg string) {
			conChan <- struct{}{}

			defer func() {
				defer wg.Done()
				<-conChan
			}()

			if val, ok := statuses[arg]; !ok || val == "fail" {
				c := fmt.Sprintf("%s %s", *cmd, arg)
				log.Println(c)
				out, err := runCmd(c)

				if err != nil {
					log.Println(err)
					statuses[arg] = "fail"
				} else {
					statuses[arg] = "success"
				}

				if out != "" {
					log.Println(out)
				}
				mu.Lock()
				defer mu.Unlock()

				jb, err = json.Marshal(statuses)
				if err != nil {
					log.Fatal(err)
				}
			}
		}(scanner.Text())
	}
	wg.Wait()
}
