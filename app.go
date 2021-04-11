package main

import (
	"fmt"
	"github.com/go-ping/ping"
	"net/http"
	"time"
)

func wakeUp(w http.ResponseWriter, r *http.Request) {
	host := "192.168.217.242"
	macAddr := "48:d7:05:bd:c6:e3"
	pingSuccess, err := pingIp(host)
	if err != nil {
		html := buildHtml("ping host:"+host+" error, "+err.Error(), "")
		fmt.Fprintf(w, html)
		return
	}
	if pingSuccess {
		html := buildHtml("Win10 is online", "1; url=https://win.home.fliaping.com:7550/vnc.html")
		fmt.Fprintf(w, html)
		return
	} else {
		err := WakeCmd(macAddr, "")
		if err != nil {
			fmt.Fprintf(w, "send WOL error, mac:"+macAddr+", "+err.Error())
			return
		}
		html := buildHtml("sent WOL, waiting Win10 online", "8")
		fmt.Fprintf(w, html)
		return
	}

}

func buildHtml(content string, refreshMeta string) string {
	before := `<!DOCTYPE html>
				<html lang="en">
				<head>
					<meta charset="UTF-8">`
	middle := `<title>WeakUp</title>
				</head>
				<body>
				<h3>`
	after := `</h3>
				</body>
				</html>`

	if refreshMeta != "" {
		refreshMeta = "<meta http-equiv=\"refresh\" content=\"" + refreshMeta + "\">"
	}

	return before + refreshMeta + middle + content + after
}

func pingIp(host string) (bool, error) {
	pinger, err := ping.NewPinger(host)
	if err != nil {
		return false, err
	}
	pinger.Count = 3
	pinger.SetPrivileged(true)
	pinger.Timeout = 3 * time.Second
	err = pinger.Run() // Blocks until finished.
	if err != nil {
		return false, err
	}
	stats := pinger.Statistics() // get send/receive/duplicate/rtt stats
	fmt.Println(stats)
	var success = stats.PacketsRecv > 0
	return success, nil
}

func main() {
	//err := WakeCmd("48:d7:05:bd:c6:e3", "")
	//if err != nil {
	//	panic(err)
	//}
	http.HandleFunc("/", wakeUp)           // 设置访问的路由
	err := http.ListenAndServe(":80", nil) // 设置监听的端口
	fmt.Println("server started, port 80")
	if err != nil {
		fmt.Println("ListenAndServe: ", err)
	}
}
