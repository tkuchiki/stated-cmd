package main

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
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
	fp, err := os.OpenFile(filename, os.O_RDWR|os.O_APPEND, 0644)
	if err == nil {
		return fp, err
	}

	fp, _ = os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	return fp, fmt.Errorf("file created")
}

func loadStatuses(fp *os.File) map[string]string {
	statuses := make(map[string]string)

	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		state := strings.SplitN(scanner.Text(), "\t", 2)
		statuses[state[1]] = state[0]
	}

	return statuses
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
		statuses := loadStatuses(fp)

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

	fp, err := openFile(*cmd)
	if err != nil {
		fmt.Println(fmt.Sprintf("create %s", fp.Name()))
	} else {
		fmt.Println(fmt.Sprintf("load %s", fp.Name()))
	}
	defer fp.Close()

	statuses := loadStatuses(fp)

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
				var cmdlog string
				if err != nil {
					log.Println(err)
					cmdlog = stateLog("fail", arg)
				} else {
					cmdlog = stateLog("success", arg)
				}

				if out != "" {
					log.Println(out)
				}
				mu.Lock()
				fmt.Fprintln(fp, cmdlog)
				mu.Unlock()
			}
		}(scanner.Text())
	}
	wg.Wait()
}
