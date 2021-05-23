package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-ping/ping"
	"github.com/reiver/go-telnet"
	"net/http"
	"os"
	"strconv"
	"time"
)

var host = ""
var telPort = 0
var macAddr = ""

func sendMsg(c *gin.Context, status int, msg string) {
	var statusString = ""
	if status == 1 {
		statusString = "online"
	} else if status == 2 {
		statusString = "processing"
	} else if status == -1 {
		statusString = "error"
	}
	c.JSON(200, gin.H{
		"message": msg,
		"status":  statusString,
	})
}

func hostStatus() (int, string) {
	pingSuccess, err := pingIp(host)

	if err != nil {
		return -1, "ping host:" + host + " error, " + err.Error()
	}
	var telSuccess = false
	if pingSuccess {
		success, err := telnetHost(host, telPort)
		if err != nil {
			return -1, "telnetHost host:" + host + ":" + strconv.Itoa(telPort) + " error, " + err.Error()
		}
		telSuccess = success
	}
	if pingSuccess && telSuccess {
		return 1, "Win10 is online"
	} else {
		return 0, ""
	}

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

func telnetHost(host string, port int) (bool, error) {
	ch := make(chan bool, 1)
	defer close(ch)

	var telError error

	go func() {
		address := host + ":" + strconv.Itoa(port)
		_, err := telnet.DialTo(address)
		defer func() {
			if e := recover(); e != nil {
				fmt.Println("recover", e)
			}
		}()
		if err == nil {
			fmt.Println("telnet success, host:" + host + ",port:" + strconv.Itoa(port))
			ch <- true
		} else {
			telError = err
		}
	}()

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	select {
	case <-ch:
		return true, nil
	case <-timer.C:
		return false, telError
	}
}

func workingHandler(c *gin.Context) {
	err := WakeCmd(macAddr, "")
	if err != nil {
		sendMsg(c, -1, "send WOL error, mac:"+macAddr+", "+err.Error())
		return
	}

	status, msg := hostStatus()

	if status == 1 {
		sendMsg(c, 1, msg)
		return
	} else if status == -1 {
		sendMsg(c, -1, msg)
	} else {
		sendMsg(c, 2, "sent WOL, waiting Win10 online")
	}
}

func indexHandler(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(indexString))
}

func main() {
	r := gin.Default()

	USER := os.Getenv("HTTP_USER")
	PASSWD := os.Getenv("HTTP_PASSWD")
	HOST_IP := os.Getenv("HOST_IP")
	TEL_PORT := os.Getenv("TEL_PORT")
	HOST_MAC := os.Getenv("HOST_MAC")

	if USER == "" || PASSWD == "" || HOST_IP == "" || TEL_PORT == "" || HOST_MAC == "" {
		panic("please set env: HTTP_USER,HTTP_PASSWD,HOST_IP,TEL_PORT,HOST_MAC")
	}
	host = HOST_IP
	port, err := strconv.Atoi(TEL_PORT)
	if err != nil {
		panic("TEL_PORT must is int")
	}
	telPort = port
	macAddr = HOST_MAC

	authorized := r.Group("/", gin.BasicAuth(gin.Accounts{
		USER: PASSWD,
	}))

	authorized.GET("/", indexHandler)
	authorized.GET("/working", workingHandler)

	r.Run(":8055") // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

const (
	indexString = `
<html>
<head>
    <title>Rstudio-Home</title>
    <script src="https://cdn.jsdelivr.net/npm/vue"></script>
    <script src="https://cdn.staticfile.org/vue-resource/1.5.1/vue-resource.min.js"></script>
    <style>
        .center {
            margin: auto;
            width: 60%;
            border: 3px solid #73AD21;
            padding: 10px;
        }
    </style>
</head>
<body>
<div id="app" class="center">
    <h2>Server Status: {{ status }}</h2>
    <h4>Message: {{ message }}</h4>
    </br></br>
    <h2><a href="https://r.home.fliaping.com:7550/">Go to Rstudio</a></h2>
</div>
<script>
    var app = new Vue({
        el: '#app',
        data: {
            message: 'Hello R!'
        },
        mounted() {
            this.working()
            this.timer = setInterval(this.working, 30000);
        },
        methods: {
            working() {
                console.log('hello,R')
                //发送get请求
                this.$http.get('/working').then(function (res) {
                    console.log(res.body);
					this.status = res.body.status;
                    this.message = res.body.message;
                }, function () {
                    console.log('请求失败处理');
					this.status = 'ERROR'
					this.message = '!!!!!!!!!!!!!!! 请求失败处理, ERROR !!!!!!!!!!!!!!!!!';
                });
            }
        }
    })
</script>
</body>
</html>
`
)
