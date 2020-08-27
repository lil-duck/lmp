//
// Created by Chenyu Zhao
// Modified by Zhenwen Xu
// Modified by ChenYuZhao on 2020/7/3.
// Modified by Qiangzhixing Cheng
//
package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"lmp/api"
	"lmp/common/influxdb"
	"lmp/config"
	"lmp/daemon"

	"github.com/cihub/seelog"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/facebookgo/pidfile"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/urfave/cli"
	"lmp/internal/BPF"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	debug.SetTraceback("crash")
	fmt.Println(logo)
	app := cli.NewApp()
	app.Name = "lmp"
	app.Usage = "lmp"
	app.Flags = config.Flags
	app.Action = func(c *cli.Context) error {
		logger, _ := seelog.LoggerFromConfigAsBytes([]byte(logtoconsoleconf))
		seelog.ReplaceLogger(logger)
		defer seelog.Flush()
		config.Initialize(c)

		pidfile.SetPidfilePath(os.Args[0] + ".pid")
		pidfile.Write()

		influxStore := influxdb.NewInfluxStore()
		influxStore.Init()

		//bpf scan service
		bpfscan := &daemon.BpfScan{}
		bpfscan.Init()
		bpfscan.Run()

		srv := api.NewServer(c)
		srv.Use(static.Serve("/", static.LocalFile("static", false)))
		srv.StaticFS("/static", http.Dir("static/"))
		srv.NoRoute(func(c *gin.Context) {
			c.File(fmt.Sprintf("%s/index.html", "static"))
		})

		config.InHost = c.String("inhost")
		config.Outhost = c.String("outhost")
		config.Port = c.String("port")
		
		listenAddress := config.InHost + ":" + config.Port
		config.GrafanaIp = config.Outhost + ":" + "3000"
		seelog.Info("Serve on ", listenAddress)

		// The following timer code is used to test the plug-in mechanism, If you like to delete it, delete it
		ticker := time.NewTicker(time.Second * 5) // 运行时长
		go func() {
			for {
				select {
				case <-ticker.C:
					//fmt.Println(bpf.PluginServices)
					for _,plugin := range bpf.PluginServices {
						fmt.Println("[pligins]: " + plugin.Name)
						fmt.Println()
					}
					//fmt.Println(bpf.PluginServices[1].Name)
				}
			}
			ticker.Stop()
		}()

		return gracehttp.Serve(&http.Server{
			Addr:         listenAddress,
			Handler:      srv,
			ReadTimeout:  100 * time.Second,
			WriteTimeout: 100 * time.Second,
		})
	}
	app.Run(os.Args)
}

const (
	logtoconsoleconf = `
	<seelog>
		<outputs>
			<console formatid="out"/>
		</outputs>
		<formats>
		    <format id="out" format="[%Level] %File:%Line %Func %Msg%n"/>
		</formats>
	</seelog>
	`
)


const (
	logo = `
                     ___        ___      ___       __________
                    /LMP|      /LMP\    /LMP|     /LMPLMPLMPL\     
                   /LMP/      |LMPLM|   LMPLM\    |LMPLMPLMPLM|
                   LMP|      /LMPLMPL\  LMPLMP|  /MPL      LMP|
                   LMP|      |LM| \LM| /LM| LMP| |MP       LMP|
                   LMP|      |LM|  LMP LM/  \LM| |MPLMPLMPLMP/
                   LMP|      /LM|  \LMPLM|   \LMPLMPLMPLMPL/      
                  /LMP|_____|LM|    \LMPL/   |LMPLM/
                  |LMPLMPLMPLMP/    |LMPL|    LMPLM|
                  \LMPLMPLMPLM/     \LMPL/     \LMP/
`
)