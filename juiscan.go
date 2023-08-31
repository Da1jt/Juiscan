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
	flag.BoolVar(&shutdown, "shutdown", shutdown, "关闭")
	flag.BoolVar(&log, "l", log, "日志写入")
	url := flag.String("url", "", "目标 URL")
	slowscan := flag.Bool("s", false, "慢速扫描")
	help := flag.Bool("h", false, "帮助")
	flag.Parse()

	fmt.Println("\n\n                     /$$                                                            \n                    |__/                                                            \n       /$$ /$$   /$$ /$$  /$$$$$$$  /$$$$$$$  /$$$$$$  /$$$$$$$   /$$$$$$   /$$$$$$ \n      |__/| $$  | $$| $$ /$$_____/ /$$_____/ |____  $$| $$__  $$ /$$__  $$ /$$__  $$\n       /$$| $$  | $$| $$|  $$$$$$ | $$        /$$$$$$$| $$  \\ $$| $$$$$$$$| $$  \\__/\n      | $$| $$  | $$| $$ \\____  $$| $$       /$$__  $$| $$  | $$| $$_____/| $$      \n      | $$|  $$$$$$/| $$ /$$$$$$$/|  $$$$$$$|  $$$$$$$| $$  | $$|  $$$$$$$| $$      \n      | $$ \\______/ |__/|_______/  \\_______/ \\_______/|__/  |__/ \\_______/|__/      \n /$$  | $$                                                                          \n|  $$$$$$/                                                                          \n \\______/                                                                           \n\n")
	/*fmt.Println("Made by Bad_jui\n")*/

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
		println("[+] 扫描完关机已启用")
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
	logco := 0
	var wg sync.WaitGroup
	for _, file := range fileList {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			temp, tempall, logco = processFile(file, *url, *slowscan, log)
		}(file)
	}
	wg.Wait()
	fmt.Println("\n\n[*] 扫描完成\n")
	fmt.Printf("[+] 存在数量：")
	fmt.Printf("%d / %d", tempall, temp)
	if logco >= 1 {
		fmt.Printf("[-] 全部或部分日志写入失败,失败数量:%s", logco)
	} else if logco == 0 {
		println("\n[+] 日志写入完成")
	}
	fmt.Println("")

	if shutdown == true {
		shutdn := exec.Command("shutdown", "/s", "/t", "0")
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
func processFile(filePath string, url string, slowmode bool, log bool) (int, int, int) {
	fmt.Println("[+] 处理字典：", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("[-] 打开文件失败：", err)
		return 0, 0, 0
	}
	defer file.Close()

	var fullc int = 0
	var counterc int = 0
	dislogcount := 0
	currentTime := time.Now()
	finalPath := "./log/" + currentTime.Format("2006-01-02_15-04-05") + "_" + url + ".txt"
	if log {
		os.Create(finalPath)
	}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		path := scanner.Text()
		result := checkPathExists(url, slowmode, path, log, finalPath, currentTime.String())
		fullc++
		if result == "normal" {
			counterc++
		}
		if result == "normalisable" {
			dislogcount++
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
func checkPathExists(url string, slowmode bool, path string, log bool, finalpath string, ctime string) string {
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

	if response.StatusCode == http.StatusOK {
		temppathhttps := "https://" + fullURL
		if log {
			if logs(url, temppathhttps, ctime, finalpath) != true {
				return "normalisable"
			}
		}
		println(temppathhttps)
		return "normal"
	}

	responser, err := http.Head("http://" + fullURL)
	if err != nil {
		/*fmt.Println("请求发生错误：", err)*/
		return "error"
	}
	defer responser.Body.Close()

	if responser.StatusCode == http.StatusOK {
		temppathhttp := "http://" + fullURL
		if log {
			if logs(url, temppathhttp, ctime, finalpath) != true {
				return "normalisable"
			}
		}
		println(temppathhttp)
		return "notnormal"
	}
	return "none"
}

func logs(path string, exist string, ctime string, finalPath string) bool {
	file, err := os.OpenFile(finalPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		/*println("无法打开日志文件：", err)*/
		return false
	}
	defer file.Close()

	if _, err := file.WriteString(exist + "\n"); err != nil {
		/*println("无法写入日志内容：", err)*/
		return false
	}

	/*println("[+] 日志写入完成：", finalPath)*/
	return true
}

// -h 时
func helper() {
	//fmt.Println("\n\n                     /$$                                                            \n                    |__/                                                            \n       /$$ /$$   /$$ /$$  /$$$$$$$  /$$$$$$$  /$$$$$$  /$$$$$$$   /$$$$$$   /$$$$$$ \n      |__/| $$  | $$| $$ /$$_____/ /$$_____/ |____  $$| $$__  $$ /$$__  $$ /$$__  $$\n       /$$| $$  | $$| $$|  $$$$$$ | $$        /$$$$$$$| $$  \\ $$| $$$$$$$$| $$  \\__/\n      | $$| $$  | $$| $$ \\____  $$| $$       /$$__  $$| $$  | $$| $$_____/| $$      \n      | $$|  $$$$$$/| $$ /$$$$$$$/|  $$$$$$$|  $$$$$$$| $$  | $$|  $$$$$$$| $$      \n      | $$ \\______/ |__/|_______/  \\_______/ \\_______/|__/  |__/ \\_______/|__/      \n /$$  | $$                                                                          \n|  $$$$$$/                                                                          \n \\______/                                                                           \n\n")
	fmt.Println("<Useage>: \n\n     -url xxx.xxx.xxx\n     -h help\n     -s 慢速扫描\n     -l 存储存在的路径入日志./log     \n     -shutdown 扫描任务完成后关机\n\n<Examp1e>:\n\n     juiscan.exe -url 127.0.0.1 -s\n     juiscan.exe -url 127.0.0.1 -shutdown -l")
}
