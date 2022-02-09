package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-ping/ping"
	"github.com/reiver/go-telnet"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var host = ""
var telPort = 0
var macAddr = ""
var lastSendWolTime = time.Unix(0, 0)
var lastCheckHostStatus = -1
var lastCheckMsg = ""

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
		return 2, "sent WOL, waiting Win10 online"
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
	refresh := c.Query("refresh")
	checked := false
	beforeInterval := time.Now().Sub(lastSendWolTime)
	if beforeInterval > time.Second*60 || "true" == strings.ToLower(refresh) {
		err := WakeCmd(macAddr, "")
		lastSendWolTime = time.Now()
		if err != nil {
			sendMsg(c, -1, "send WOL error, mac:"+macAddr+", "+err.Error())
			return
		}
		status, msg := hostStatus()
		lastCheckHostStatus = status
		lastCheckMsg = msg
		checked = true
	}

	msg := lastCheckMsg
	if !checked {
		msg = msg + " (check before " + beforeInterval.String() + ")"
	}
	if lastCheckHostStatus == 1 {
		sendMsg(c, 1, msg)
	} else if lastCheckHostStatus == -1 {
		sendMsg(c, -1, msg)
	} else {
		sendMsg(c, 2, msg)
	}
}

func indexHandler(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(indexString))
}

func clashFilterHandler(c *gin.Context) {
	url := c.Query("url")
	filterPattern := c.Query("pattern")
	var client http.Client
	resp, err := client.Get(url)
	if err != nil {
		log.Println(err.Error())
		c.Data(http.StatusBadRequest, "application/json; charset=utf-8", []byte("{\"msg\":\""+err.Error()+"\"}"))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		bodyString := string(bodyBytes)
		filtered := ClashHandler(bodyString, filterPattern)
		c.Data(http.StatusBadRequest, "text/plain; charset=utf-8", []byte(filtered))
	}
}

func main() {
	r := gin.Default()

	USER := os.Getenv("HTTP_USER")
	PASSWD := os.Getenv("HTTP_PASSWD")
	HOST_IP := os.Getenv("HOST_IP")
	TEL_PORT := os.Getenv("TEL_PORT")
	HOST_MAC := os.Getenv("HOST_MAC")

	HTTP_PORT := os.Getenv("HTTP_PORT")

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

	r.GET("/clashFilter", clashFilterHandler)

	authorized := r.Group("/", gin.BasicAuth(gin.Accounts{
		USER: PASSWD,
	}))

	authorized.GET("/", indexHandler)
	authorized.GET("/working", workingHandler)

	var addr = ":8055"
	if HTTP_PORT != "" {
		addr = ":" + HTTP_PORT
	}
	r.Run(addr) // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

const (
	indexString = `
<html>
<head>
    <title>Rstudio-Home</title>
    <script src="https://cdn.jsdelivr.net/npm/vue@2.6.14"></script>
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
	</br>
	<h3>注意：使用过程请不要关闭本页面</h3>
	</br>
    <h4>Message: {{ message }}</h4>
    </br></br>
    <h2><a href="https://r.home.fliaping.com:7550/" target="_blank">Go to Rstudio</a></h2>
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
