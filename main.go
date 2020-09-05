package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	DEFAULT_LOOP_MS = 100
)

var (
	g_flag_exit bool = false
)

func enum_files_in_dir(dir_path string, suffix string) ([]string, error) {
	paths_buffer := make([]string, 0, 100)
	files, err := ioutil.ReadDir(dir_path)
	if err != nil {
		log.Fatal(err)
		return paths_buffer, err
	}
	upper_suffix := strings.ToUpper(suffix)
	for _, file_info := range files {

		if file_info.IsDir() == true {
			sub_path, err := enum_files_in_dir(dir_path, suffix)
			if err != nil {
				for _, full_path := range sub_path {
					paths_buffer = append(paths_buffer, full_path)
				}
			} else {
				return paths_buffer, err
			}
		} else {
			if upper_suffix != "*" {
				upper_filename := strings.ToUpper(file_info.Name())
				if strings.HasSuffix(upper_filename, upper_suffix) == false {
					continue
				}
			}
			full_path := filepath.Join(dir_path, file_info.Name())
			abs_full_path, err := filepath.Abs(full_path)
			if err != nil {
				log.Fatal(err)
			}
			paths_buffer = append(paths_buffer, abs_full_path)
		}
	}
	return paths_buffer, nil
}

func enum_config_in_dir(dir_path string) ([]string, error) {
	return enum_files_in_dir(dir_path, ".conf")
}

func load_config_in_dir(dir_path string) []*CronItem {
	file_paths, err := enum_config_in_dir(dir_path)
	if err != nil {
		log.Fatal(err)
	}

	jobs := make([]*CronItem, 0, 10)
	for _, file_path := range file_paths {
		log.Printf("load config file : %s", file_path)
		raw_data, err := ioutil.ReadFile(file_path)
		if err != nil {
			log.Fatal(err)
		}
		job_item := NewCronItem()
		err = job_item.LoadCronItemFromJson(raw_data)
		if err != nil {
			log.Fatal(err)
		}
		jobs = append(jobs, job_item)
		log.Printf("load config file : %s successed", file_path)
	}
	return jobs
}

func channel_println(log_ch chan string) {
	var buffer string
	for {
		buffer = <-log_ch
		if buffer == "CMD:EXIT" {
			break
		}
		log.Printf(buffer)
	}
}

func core_loop(cron_jobs []*CronItem) {
	var exec_signal sync.WaitGroup = sync.WaitGroup{}

	log_ch := make(chan string)
	defer close(log_ch)
	go channel_println(log_ch)

	for {
		if g_flag_exit == true {
			break
		}
		current_time := time.Now()
		for _, job_object := range cron_jobs {
			if job_object.IsNeedExecute(current_time) == true {
				exec_signal.Add(1)
				go job_object.GoExecuteTask(&exec_signal, log_ch)
			}
		}
		time.Sleep(time.Millisecond * DEFAULT_LOOP_MS)
	}
	exec_signal.Wait()
	log_ch <- "CMD:EXIT"
	log.Printf("Exit loop")
}

func signal_exit(ch chan os.Signal) {
	for sig := range ch {
		switch sig {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM:
			log.Println("Set Exit signal")
			g_flag_exit = true
			break
		default:
			break
		}
	}
}

func register_signal() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go signal_exit(ch)
}

func init() {
	const (
		gocroncli_version = "0.0.2"
	)
	log.Printf(
		"gocroncli version : %s",
		gocroncli_version)
	log.Printf("Build date : 20200905-00")
	log.Printf(
		"Go runtime version : %s",
		runtime.Version())
}

func main() {
	ptr_flag_help := flag.Bool("help", false, "display help.")
	ptr_flag_config_dir := flag.String("config_dir", "./config", "set configuration dir `dir`")
	flag.Parse()

	if *ptr_flag_help {
		flag.Usage()
		os.Exit(0)
	}

	cron_jobs := load_config_in_dir(*ptr_flag_config_dir)
	register_signal()
	core_loop(cron_jobs)
}
