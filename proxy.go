package main

import (
	"log"
	"github.com/parnurzeal/gorequest"
	"regexp"
	"strconv"
	"time"
	"os"
	"sync"
	"errors"
	"strings"
	"encoding/json"
	"github.com/kr/beanstalk"
	"net"
	"runtime"
)


type Proxy struct {
	ip	string
	port	int
	typ	string
}


func (proxy *Proxy) Valid() (remote_ip string, time_ms float32, err error) {
	conn, err := net.DialTimeout("tcp", proxy.ip + ":" + strconv.Itoa(proxy.port), 2 * time.Second)
	if err != nil {
		err = errors.New("Valid faild: tcp timeout")
		return
	}
	conn.Close()
	request := gorequest.New()
	request.Proxy(proxy.typ + "://" + proxy.ip + ":" + strconv.Itoa(proxy.port))
	request.Timeout(5 * time.Second)
	t1 := time.Now().UnixNano()
	resp, body, errs := request.Get("http://ip.cn").End()
	t2 := time.Now().UnixNano()
	time_ms = float32((t2 - t1) / 1000.0 / 1000.0)
	if len(errs) == 0 && resp.StatusCode == 200 && strings.Count(body, "15005128") > 0 {
		// get remote ip address from resp
		reg := regexp.MustCompile("\\d+\\.\\d+\\.\\d+\\.\\d+")
		ips := reg.FindAllString(body, 1)
		if len(ips) == 1 {
			remote_ip = ips[0]
			err = nil
			return
		}
	}
	err = errors.New("Valid faild")
	return
}

type Task struct {
	Ip	string	`json:"ip"`
	Port	int	`json:"port"`
}


var ChTask		chan *Task
var conn		chan bool
const MaxConn	=	6000
var mutex sync.Mutex

func goValid(ip string, port int) {
	proxy := &Proxy{ip, port, "http"}
	remote_ip, timems , err := proxy.Valid()
	if err == nil {
                if remote_ip != ip && port == 80 {
			//cdn!
			conn <- false
			return
		}
		mutex.Lock()
		f,_ := os.OpenFile("result.txt",os.O_CREATE|os.O_APPEND|os.O_RDWR,0660)
		f.Write([]byte(ip + "|" + strconv.Itoa(proxy.port) + "|" + strconv.Itoa(int(timems)) + "|" + remote_ip + "\n"));
		f.Close()
		mutex.Unlock()
	}
	conn <- true
	return
}


func Work() {
	var curr_conn = 0
	for{
		if curr_conn < MaxConn {
			select {
				case t := <- ChTask:
					log.Printf("Work() get task %#v", t)
					curr_conn++
					go goValid(t.Ip, t.Port)
				//default:
			}
		} else {
			select {
				case <- conn:
					//log.Printf("Work() 1 connect release")
					curr_conn--
				//default:
			}
		}
	}
}


func GetTask() {
	c, err := beanstalk.Dial("tcp", "127.0.0.1:11300")
	if err != nil {
		log.Printf("beanstalk connect failed")
		return
	}
	defer c.Close()

	for{
		id, body, err := c.Reserve(1000*time.Millisecond)
		if err != nil {
			if err.Error() != "reserve-with-timeout: timeout" {
				log.Printf("GetTask get err may error", err)
			}
			c.Delete(id)
			continue
		}
		var t Task
		err = json.Unmarshal(body, &t)
		if err != nil {
			log.Printf("GetTask get unknow format data from beanstalk.")
			c.Delete(id)
			continue
		}
		ChTask <- &t
		c.Delete(id)
	}
}


func main() {
	ChTask = make(chan *Task)
	conn = make(chan bool)
	go Work()
	GetTask()
}

