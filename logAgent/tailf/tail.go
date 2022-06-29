package tailf

import (
	"sync"
	"time"

	"github.com/astaxie/beego/logs"
	"github.com/hpcloud/tail"
)

// 收集日志的config信息
type CollectConf struct {
	LogPath string `json:"logpath"`
	Topic   string `json:"topic"`
}

// 跟踪文件的打印的信息的实例
// 包含*tail.Tail结构体，与该跟踪文件的配置信息
type TailObj struct {
	// 跟踪文件实例
	Tail *tail.Tail
	// 所跟踪文件的地址,主题等信息
	Config CollectConf
	// 跟踪实例的状态信息（正常运行，停止运行/删除任务）
	Status int
	// 用于结束tailf任务的通信Channel(销毁对应开启的goroutine)
	ExitChan chan bool
}

//用于表示tailObj 任务运行状态的常量
const (
	// 正常运行 //默认正常运行
	StatusNormal = 0
	// 该tailObjc 待取消
	StatusDelete = 2
)

// 管理多个跟踪文件
type TailObjMgr struct {
	//存放全部的TailObj
	TailObjs []*TailObj
	// 用于协程间通信，传输tailObj收集的数据
	MsgChan chan *TextMsg
	// 互斥锁，防止多协程对TailObjMgr修改造成资源冲突
	Lock sync.Mutex
}

// 使用一个channel　将打 印的信息发送给main包，再传给kafka
// 两个字段　消息　写入哪个topic
type TextMsg struct {
	Msg   string
	Topic string
}

//不能被外包使用
var (
	tailObjMgr *TailObjMgr
)

func InitTail(conf []CollectConf, chanSize int) (err error) {
	// 先初始化一个管理tailf实例的对象
	tailObjMgr = &TailObjMgr{
		MsgChan: make(chan *TextMsg, chanSize),
	}
	if len(conf) == 0 {
		// err = fmt.Errorf("invalid config for log collect ,conf=%v", conf)
		logs.Error("invalid config for log collect ,conf=%v", conf)
		return
	}

	for _, v := range conf {
		// 创建tailf实例
		createNewTailf(v)
	}
	logs.Debug("init tailf success")

	return
}

// 创建新的tailf实例
func createNewTailf(oneConf CollectConf) (err error) {
	tailObj := &TailObj{
		Config:   oneConf,
		ExitChan: make(chan bool, 1),
	}
	tails, err := tail.TailFile(oneConf.LogPath, tail.Config{
		ReOpen: true,
		Follow: true,
		// Location:  &tail.SeekInfo{Offset: 0, Whence: 2}, //定位读取位置 最后
		MustExist: false, //要求文件必须存在或者暴露
		Poll:      true,
	})

	if err != nil {
		// fmt.Printf("tail.TailFile %v, err:%v\n", v, err)
		logs.Error("tail.TailFile failed conf:%v, err:%v\n", oneConf, err)
		return
	}
	tailObj.Tail = tails

	tailObjMgr.TailObjs = append(tailObjMgr.TailObjs, tailObj)

	//配置好tail环境后，实现读取功能
	go readFormTail(tailObj)
	return
}

//还需要优雅终止：使用tailObj.ExitChann 控制销毁
// 使用ctx?
func readFormTail(tailObj *TailObj) {
	// var msg *tail.Line
	flag := true
	for flag {
		select {
		case line, ok := <-tailObj.Tail.Lines:
			if !ok {
				logs.Warn("tail file close reopen, filename:%s\n", tailObj.Tail.Filename)
				time.Sleep(100 * time.Millisecond)
				continue
			}

			textMsg := &TextMsg{
				Msg:   line.Text,
				Topic: tailObj.Config.Topic,
			}

			tailObjMgr.MsgChan <- textMsg
		case v := <-tailObj.ExitChan:
			if !v {
				logs.Warn("tailObj: %v will exited ", tailObj)
				time.Sleep(time.Second * 2)
				return
			}

		}

		//优雅退出
		//配置一个退出的信号，在环境变量中配置，指针形式
		// 当接收到退出信号将flag置为true
	}
}

//使用函数方式读取信息
func GetOneLine() (msg *TextMsg) {
	msg = <-tailObjMgr.MsgChan
	return
}

// 更新配置信息
func UpdateConfig(confs []CollectConf) (err error) {
	// 先对tailObjMgr 上锁
	tailObjMgr.Lock.Lock()
	//解锁
	defer tailObjMgr.Lock.Unlock()

	// PUT
	//查询现有的配置信息中是否有更新后的配置信息，
	// 达到只启动未有的配置信息的tailf
	for _, oneConf := range confs {
		// 设置是否存在oneConf 配置的tailf
		var isRunning bool = false

		for _, tailObj := range tailObjMgr.TailObjs {
			if oneConf == tailObj.Config {
				isRunning = true
				break
			}
		}

		if isRunning {
			continue
		}
		// 创建新的tailf实例
		createNewTailf(oneConf)
	}

	// DELETE
	// 销毁更新配置中没有的tailObj任务

	//用于更新tailObjMgr.TailObjs，
	var tailObjs []*TailObj

	// 比较现有运行的tailObjcs中的config有没有不在更新后的confsz中的，有则销毁
	for _, tailObj := range tailObjMgr.TailObjs {
		// 先将tailObj 状态设置为删除状态
		tailObj.Status = StatusDelete
		for _, oneConf := range confs {
			if tailObj.Config == oneConf {
				// 有相应的tailObj.Config信息，将实例状态置回正常运行默认值
				tailObj.Status = StatusNormal
				break
			}
		}

		//销毁tailObj对应的goroutine，以及将该tailObj 从tailObjMgr.TailObjs中删除
		if tailObj.Status == StatusDelete {
			// 向ExitChan管道输入值，控制goroutine退出
			tailObj.ExitChan <- false
			continue
		}

		// 剔除待删除的tailObj 实例
		tailObjs = append(tailObjs, tailObj)
	}
	//采用整体替换的方式，实现删除tailObjMgr.TailObjs中对应的tailObj的效果
	tailObjMgr.TailObjs = tailObjs

	return
}
