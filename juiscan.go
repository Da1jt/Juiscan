package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

func main() {
	shutdown := false
	log := false
	deepscan := false
	flag.BoolVar(&shutdown, "shutdown", shutdown, "关闭")
	flag.BoolVar(&log, "l", log, "日志写入")
	flag.BoolVar(&deepscan, "d", deepscan, "递归扫描")
	url := flag.String("url", "", "目标 URL")
	slowscan := flag.Bool("s", false, "慢速扫描")
	help := flag.Bool("h", false, "帮助")
	flag.Parse()

	fmt.Println("\n\n                     /$$                                                            \n                    |__/                                                            \n       /$$ /$$   /$$ /$$  /$$$$$$$  /$$$$$$$  /$$$$$$  /$$$$$$$   /$$$$$$   /$$$$$$ \n      |__/| $$  | $$| $$ /$$_____/ /$$_____/ |____  $$| $$__  $$ /$$__  $$ /$$__  $$\n       /$$| $$  | $$| $$|  $$$$$$ | $$        /$$$$$$$| $$  \\ $$| $$$$$$$$| $$  \\__/\n      | $$| $$  | $$| $$ \\____  $$| $$       /$$__  $$| $$  | $$| $$_____/| $$      \n      | $$|  $$$$$$/| $$ /$$$$$$$/|  $$$$$$$|  $$$$$$$| $$  | $$|  $$$$$$$| $$      \n      | $$ \\______/ |__/|_______/  \\_______/ \\_______/|__/  |__/ \\_______/|__/      \n /$$  | $$                                                                          \n|  $$$$$$/                                                                          \n \\______/                                                                           \n\n")

	if *help {
		helper()
		return
	}
	if *url == "" {
		fmt.Println("无效的 URL")
		return
	}

	if *slowscan {
		fmt.Println("[+] 慢速扫描已启用")
	}

	if shutdown {
		println("[+] 扫描完成后关机")
	}

	// 处理其他命令行参数
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "-h" {
			helper()
			return
		}
		if os.Args[i] == "-shutdown" {
			shutdown = true
		}
		if os.Args[i] == "-l" {
			log = true
		}
		if os.Args[i] == "-d" {
			deepscan = true
		}
	}
	conn, err := net.Dial("ip4:icmp", *url)
	if err != nil {
		fmt.Println("地址存活性检测失败：", err)
		os.Exit(1)
		return
	}
	defer conn.Close()
	dirPath := "./dictionary/"

	fileList, err := getFileList(dirPath)
	if err != nil {
		fmt.Println("读取目录失败：", err)
		return
	}
	temp := 0
	tempall := 0
	logfail := 0
	var wg sync.WaitGroup
	for _, file := range fileList {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			temp, tempall, logfail = processFile(file, *url, *slowscan, log, deepscan)
		}(file)
	}
	wg.Wait()
	fmt.Println("\n\n[*] 扫描完成\n")
	fmt.Print("[+] 存在数量：")
	fmt.Print(tempall, " / ", temp, "\n")
	if logfail >= 1 {
		fmt.Printf("[-] 全部或部分日志写入失败，失败数量:%d", logfail)
	} else if logfail == 0 {
		println("\n[+] 日志写入完成")
	}
	fmt.Println("")

	if shutdown == true {
		shutdn := exec.Command("shutdown", "/f")
		shutdnt := shutdn.Run()
		if shutdnt != nil {
			fmt.Println("\n[-] 关机失败\n\n")
		} else {
			fmt.Println("[+] 关机成功")
		}
	}
}

// 获取各个字典名称
func getFileList(dirPath string) ([]string, error) {
	fileList := []string{}

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".txt" {
			fileList = append(fileList, path)
		}
		return nil
	})

	return fileList, err
}

// 处理字典
func processFile(filePath string, url string, slowmode bool, log bool, deepscan bool) (int, int, int) {
	fmt.Println("[+] 处理字典：", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("[-] 打开文件失败：", err)
		return 0, 0, 0
	}
	defer file.Close()

	var fullc int = 0
	var counterc int = 0
	/*deepnumb := 0
	deepnexist := 0*/
	dislogcount := 0
	currentTime := time.Now()
	var finalPath string = "./log/" + url + " " + currentTime.Format("2006-01-01_15-04-05") + ".txt"
	if log {
		os.Create(finalPath)
	}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		path := scanner.Text()
		result := checkPathExists(url, slowmode, path, log, finalPath)
		fullc++
		if result == "httpnormalisable" || result == "httpsnormalisable" {
			dislogcount++
			counterc++
		}
		if result == "httpsnormal" || result == "httpnormal" {
			counterc++
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("[-] 读取文件失败：", err)
	}
	if log == false {
		return fullc, counterc, -1
	}
	return fullc, counterc, dislogcount

}

// 发包
func checkPathExists(url string, slowmode bool, path string, log bool, finalpath string) string {
	if slowmode {
		time.Sleep(1 * time.Second)
	}
	fullURL := url + "/" + path
	response, err := http.Head("https://" + fullURL)
	if err != nil {
		/*fmt.Println("请求发生错误：", err)*/
		return "error"
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		temppathhttps := "https://" + fullURL
		fmt.Print(response.StatusCode, " ", temppathhttps, "\n")
		if log {
			if err := logs(temppathhttps, finalpath); err != nil {
				fmt.Println("Error writing to log:", err)
				return "httpsnormalisable"
			}
		}
		return "httpsnormal"
	}

	responser, err := http.Head("http://" + fullURL + "/")
	if err != nil {
		fmt.Println("Error sending HTTP request:", err)
		return "error"
	}
	defer responser.Body.Close()

	if responser.StatusCode != http.StatusNotFound {
		temppathhttp := "http://" + fullURL
		fmt.Print(responser.StatusCode, " ", temppathhttp, "\n")
		if log {
			if err := logs(temppathhttp, finalpath); err != nil {
				fmt.Println("Error writing to log:", err)
				return "httpnormalisable"
			}
		}
		return "httpnormal"
	}

	return "none"
}

func logs(exist string, finalPath string) error {
	file, err := os.OpenFile(finalPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(exist + "\n")
	if err != nil {
		return err
	}

	return nil
}

// -h
func helper() {
	fmt.Println("<Useage>: \n\n     -url xxx.xxx.xxx\n     -h help\n     -s 慢速扫描\n     -l 存储存在的路径入日志./log     \n     -shutdown 扫描任务完成后关机\n\n<Examp1e>:\n\n     juiscan.exe -url 127.0.0.1 -s\n     juiscan.exe -url 127.0.0.1 -shutdown -l")
}
