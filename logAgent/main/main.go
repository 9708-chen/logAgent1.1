package main

import (
	"fmt"

	"logAgent/kafka"
	"logAgent/tailf"

	"github.com/astaxie/beego/logs"
)

func main() {
	// 1.初始化配置
	filename := "../config/logcollect.conf"
	err := loadConf("ini", filename)
	if err != nil {
		fmt.Println("loadConf failed,", err)
		panic("load config failed")
		// return
	}

	//2.初始化日志
	err = initLogger()
	if err != nil {
		fmt.Println("init logger  failed,", err)
		panic("init logger failed")
		// return
	}

	// 打印一个日志初始化成功标志
	logs.Debug("init success")
	// 打印配置信息
	logs.Debug("load config success ,config:%v", appConfig)

	// 3.初始化etcd,并获取配置信息
	collectConf, err := initEtcd(appConfig.etcdAddr, appConfig.etcdKey)
	if err != nil {
		logs.Error("init etcd failed, err:%v", err)
	}

	// 3.初始化tailf
	err = tailf.InitTail(collectConf, appConfig.ChanSize)
	if err != nil {
		logs.Error("init tailf failed,", err)
	}

	// 初始化kafka
	err = kafka.InitKafka(appConfig.kafkaAddr)
	if err != nil {
		logs.Error("init kafka failed,", err)
		return
	}

	logs.Debug("init all successed")

	// go func() {
	// 	for i := 0; i < 10; i++ {
	// 		logs.Debug("test for logger %d ", i+10)
	// 		time.Sleep(time.Millisecond * 10)
	// 	}
	// }()

	//logicMain,实际业务
	err = serverRun()
	if err != nil {
		logs.Error("serverRun failed err:%v", err)
		return
	}

	logs.Info("server Run exited")

}
