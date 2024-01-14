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
	"strconv"
	"strings"
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
	ports := flag.String("port", "", "端口扫描范围")
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

	port_scan := "no"
	Gpstart, Gpend, G_port_single := 0, 0, 0
	if *ports != "" {
		if strings.Contains(*ports, "-") {
			parts := strings.Split(*ports, "-")
			pstart := parts[0]
			pend := parts[1]
			pstart_int, err := strconv.Atoi(pstart)
			if err != nil {
				fmt.Println("[-] 端口范围输入错误", err)
				return
			}
			pend_int, err := strconv.Atoi(pend)
			if err != nil {
				fmt.Println("[-] 端口范围输入错误", err)
				return
			}
			if is_valid_port(pstart_int) && is_valid_port(pend_int) {
				fmt.Printf("[+] 端口扫描已启用 %d-%d", pstart, pend)
				port_scan = "range"
				Gpstart, Gpend = pstart_int, pend_int

			} else {
				fmt.Println("[-] 端口范围输入错误", err)
				return
			}
		} else {
			port_single, err := strconv.Atoi(*ports)
			if err != nil {
				fmt.Println("端口范围输入错误", err)
				return
			}
			if is_valid_port(port_single) {
				fmt.Printf("[+] 端口扫描已启用 %d", port_single)
				port_scan = "single"
			}
		}
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
	temp, tempall, logfail, logpath := 0, 0, 0, ""
	var wg sync.WaitGroup
	for _, file := range fileList {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			temp, tempall, logfail, logpath = processFile(file, *url, *slowscan, log, deepscan)
		}(file)
	}
	ip_a, alive_port, TotalAlive := []string{}, [][]int{}, 0
	if port_scan != "no" {
		if port_scan == "range" {
			portlist := []int{Gpstart, Gpend}
			ip_a, alive_port, TotalAlive = port_scanner(portlist, port_scan, *url)
		} else {
			portlist := []int{G_port_single}
			ip_a, alive_port, TotalAlive = port_scanner(portlist, port_scan, *url)
		}
		for i := 0; i < len(alive_port); i++ {
			for j := 0; j < len(alive_port[i]); j++ {
				Err := logs(strconv.Itoa(alive_port[i][j]), logpath)
				if Err != nil {
					logfail++
				}
			}
		}
	}

	wg.Wait()
	fmt.Println("\n\n[*] 扫描完成\n")
	fmt.Print("[+] 存在数量路径：")
	fmt.Print(tempall, " / ", temp, "\n")
	fmt.Print("[+] 存活端口数量：")
	all_port := len(ip_a) * (Gpend - Gpstart)
	fmt.Print(TotalAlive, " / ", all_port, "\n")
	if logfail >= 1 {
		fmt.Printf("[-] 全部或部分日志写入失败，失败数量:%d", logfail)
	} else if logfail == 0 {
		println("\n[+] 日志写入完成")
	}
	fmt.Println()

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

func is_valid_port(port int) bool {
	if port >= 1 && port <= 65535 {
		return true
	}
	return false
}
func port_scanner(portlist []int, method string, ip string) ([]string, [][]int, int) {
	var alive_port [][]int
	TotalAlive := 0
	ip_a, err := url2ip(ip)
	if err != nil {
		fmt.Println("[-] url解析错误：", err)
		return nil, nil, 0
	}
	for j := 0; j < len(ip_a); j++ {
		if method == "single" {
			alive_port[j][0] = sub_portscan(portlist[0], ip_a[j])
			if alive_port[j][0] != 0 {
				TotalAlive++
				fmt.Printf("[+] %s:%d 端口开放\n", ip_a[j], alive_port[j][0])
			}
		} else if method == "range" {
			counter := 0
			for i := portlist[0]; i <= portlist[1]; i++ {
				res := sub_portscan(i, ip_a[j])
				if res != 0 {
					alive_port[j][counter] = res
					fmt.Printf("[+] %s:%d 端口开放\n", ip_a[j], alive_port[j][counter])
					counter++
					TotalAlive++
				}
			}
		}
	}
	return ip_a, alive_port, TotalAlive
}

func sub_portscan(port int, ip string) int {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, strconv.Itoa(port)), time.Second)
	if err == nil {
		defer conn.Close()
		return port
	}
	return 0
}

// 处理字典
func processFile(filePath string, url string, slowmode bool, log bool, deepscan bool) (int, int, int, string) {
	fmt.Println("[+] 处理字典：", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("[-] 打开文件失败：", err)
		return 0, 0, 0, "0"
	}
	defer file.Close()

	var fullc = 0
	var counterc = 0
	/*deepnumb := 0
	deepnexist := 0*/
	dislogcount := 0
	currentTime := time.Now()
	var finalPath = "./log/" + url + " " + currentTime.Format("2006-01-01_15-04-05") + ".txt"
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
			/*if deepscan {
				if result == "normal" {
					counterc++
					temppath := url + "/" + path
					if deepscan {
						for scanner.Scan() {
							path := scanner.Text()
							println(temppath)
							deepresult := deepcheck(url+temppath, slowmode, path, log, finalPath, currentTime.String())
							fullc++
							if deepresult == "normal" {
								counterc++
								deepnumb++
							}
							if deepresult == "normalisable" {
								dislogcount++
								counterc++
								deepnumb++
							}
							if deepresult == "none" {
								deepnexist++
								counterc++
							}
						}
						println(deepnumb)
					}
				}
			}*/
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("[-] 读取文件失败：", err)
	}
	if log == false {
		return fullc, counterc, -1, finalPath
	}
	return fullc, counterc, dislogcount, finalPath
}
func url2ip(url string) ([]string, error) {
	addrs, err := net.LookupIP(url)
	if err != nil {
		return nil, err
	}
	var ips []string
	for _, ip := range addrs {
		ips = append(ips, ip.String())
	}
	return ips, nil
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
		fmt.Print("[+] ", time.Now().Format("15:04:05"), "  ", response.StatusCode, "  ", temppathhttps, "\n")
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
		fmt.Print("[+] ", time.Now().Format("15:04:05"), "  ", response.StatusCode, "  ", temppathhttp, "\n")
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
	fmt.Println("<Useage>: \n\n     -url xxx.xxx.xxx\n     -h help\n     -s 慢速扫描\n     -p 目标端口扫描，可以以1~100的格式指定扫描范围或单个扫描     \n     -l 存储存在的路径入日志./log     \n     -shutdown 扫描任务完成后关机\n\n<Examp1e>:\n\n     juiscan.exe -url 127.0.0.1 -s\n     juiscan.exe -url 127.0.0.1 -shutdown -l")
}
