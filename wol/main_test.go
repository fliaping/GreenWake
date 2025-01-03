package main

import (
	"fmt"
	"regexp"
	"testing"
)

func TestXxx(*testing.T) {
	//ClashHandler("",".*(流量|备用|临时|耗尽).*")

	match, _ := regexp.MatchString(".*(流量|备用|临时|耗尽|\\[2\\]|\\[5\\]|\\[1.5\\]).*", "- ❷V2.US幻象 [综合] [2]")
	if match {
		fmt.Println("mtch")
	}
}
