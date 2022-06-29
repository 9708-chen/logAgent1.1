package main

import (
	"context"
	"encoding/json"
	"fmt"
	"logAgent/tailf"
	"strings"
	"time"

	"github.com/astaxie/beego/logs"
	etcd_client "github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/storage/storagepb"
)

// etcd需要持续监听，定义成全局变量
type EtcdClient struct {
	client *etcd_client.Client
	keys   []string //etcd  存储中的key
}

var (
	etcdClient *EtcdClient
)

//拆分成init 和获取etcd中的配置
func initEtcd(addr, key string) (collectConf []tailf.CollectConf, err error) {
	cli, err := etcd_client.New(etcd_client.Config{
		Endpoints:   []string{"localhost:2379", "localhost:22379", "localhost:32379"}, //使用addr 返回dial 10.141.65.188:2379 connection refused
		DialTimeout: 5 * time.Second,
	})

	if err != nil {
		logs.Error("etcd connect failed,err :", err)
		return
	}
	logs.Debug("init etcd succ ")
	etcdClient = &EtcdClient{client: cli}
	// defer cli.Close() //一直要有没，就不关闭了

	// 判断key是否以/结尾,若不是则添加/
	if !strings.HasSuffix(key, "/") {
		key = key + "/"
	}

	// 定义存放配置的实例
	// var collectConf []tailf.CollectConf //只能获取localIPs中的一个ip所对应的key-val
	for _, ip := range LocalIPs {
		// 组合得到完整明确的　属于某个ip地址的etcdKey
		etcdKey := fmt.Sprintf("%s%s", key, ip)
		etcdClient.keys = append(etcdClient.keys, etcdKey)

		//超时设置
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		resp, err := cli.Get(ctx, etcdKey)
		cancel() //取完即销毁
		if err != nil {
			// fmt.Println("get failed,err:", err)
			logs.Error("client get from etcd failed,err:", err)
			continue
		}

		for _, ev := range resp.Kvs {
			// fmt.Printf("%s:%s\n", ev.Key, ev.Value)
			if string(ev.Key) == etcdKey {
				err = json.Unmarshal(ev.Value, &collectConf)
				if err != nil {
					logs.Error("etcd unmarshal failed,err:%v", err)
					continue
				}
				logs.Debug("key:%v, value:log config is %v", etcdKey, collectConf)
			}
		}
	}

	fmt.Println("connect succ")

	//监控etcd中信息的变动情况
	initEtcdWatcher()

	return
}

//监听etcd中ip key 的val是否有更新
func initEtcdWatcher() {
	for _, key := range etcdClient.keys {
		go watchKey(key)
	}
}

func watchKey(key string) {
	logs.Debug("begin watch key:%s", key)
	for { //持续监听，配置是否有变化　watch 主动告知配置有变
		rch := etcdClient.client.Watch(context.Background(), key) //阻塞在此处，有新变化，管道输出信息

		var (
			// 配置变更信息
			collectConf []tailf.CollectConf
			// 配置变更后成功应用的标识
			getConf bool = true
		)

		for wresp := range rch {
			for _, ev := range wresp.Events {
				// key delete 操作
				// 没有对etcd中key被删除后，配置信息没有了进行相应的操作
				if ev.Type == storagepb.DELETE {
					logs.Warn("key[%s]'s config deleted", key)
					continue
				}

				// key put更新操作
				if ev.Type == storagepb.PUT && string(ev.Kv.Key) == key {
					err := json.Unmarshal(ev.Kv.Value, &collectConf)
					if err != nil {
						getConf = false
						logs.Error("watchKey,key[%s],unmarshal[%s],err:%v", key, ev.Kv.Value, err)
						continue
					}
				}
				// fmt.Printf("%s %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
				logs.Debug("get  config from etcd,%s %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
			}

			//???
			if getConf {
				logs.Debug("get config from etcd succ,%v", collectConf)
				// 配置信息正确更新，将配置信息传输到tail模块更新函数中
				tailf.UpdateConfig(collectConf)
			}
		}
	}
}
