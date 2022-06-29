package main

import (
	"context"
	"encoding/json"
	"fmt"
	"logAgent/tailf"
	"time"

	etcd_client "github.com/coreos/etcd/clientv3"
)

const (
	EtcdKey = "/oldboy/backend/logagent/config/10.141.65.188"
)

// type LogConf struct {
// 	Path  string `json:"path"`
// 	Topic string `json:"topic"`
// 	// SendQps int
// }

func SetLogConfToEtcd() {
	cli, err := etcd_client.New(etcd_client.Config{
		Endpoints:   []string{"localhost:2379", "localhost:22379", "localhost:32379"},
		DialTimeout: 5 * time.Second,
	})

	if err != nil {
		fmt.Println("connect failed,err :", err)
		return
	}
	fmt.Println("connect succ")
	defer cli.Close()

	var logConfArr []tailf.CollectConf
	// logConf  配置
	logConfArr = append(logConfArr,
		tailf.CollectConf{
			LogPath: "/home/yunlongchen/nginx/logs/access.log",
			Topic:   "nginx_log",
		},
	)

	// logConf  配置
	logConfArr = append(logConfArr,
		tailf.CollectConf{
			LogPath: "/home/yunlongchen/nginx/logs/error.log",
			Topic:   "nginx_log_err",
		},
	)
	// logConf  配置
	// logConfArr = append(logConfArr,
	// 	tailf.CollectConf{
	// 		LogPath: "/home/yunlongchen/Documents/goproject/src/logAgent/logs/logcollect.log",
	// 		Topic:   "nginxLog",
	// 	},
	// )

	// json 打包数据
	data, err := json.Marshal(logConfArr)
	if err != nil {
		fmt.Println("logConfArr json filed,err:", err)
		return
	}
	baseCtx := context.Background()
	ctx, cancel := context.WithTimeout(baseCtx, time.Second)
	_, err = cli.Put(ctx, EtcdKey, string(data))
	cancel()
	if err != nil {
		fmt.Println("put failed,err:", err)
		return
	}

	ctx, cancel = context.WithTimeout(baseCtx, time.Second)
	resp, err := cli.Get(ctx, EtcdKey)
	cancel()
	if err != nil {
		fmt.Println("get failed,err:", err)
		return
	}

	for _, ev := range resp.Kvs {
		fmt.Printf("%s:%s\n", ev.Key, ev.Value)
	}

}

func main() {
	SetLogConfToEtcd()
}
