package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/zhifengle/rss2cloud/p115"
	"github.com/zhifengle/rss2cloud/rsssite"
	"github.com/zhifengle/rss2cloud/server"
)

var (
	pAgent       *p115.Agent
	rssUrl       string
	cookies      string
	rssJsonPath  string
	qrLogin      bool
	disableCache bool
	chunkDelay   int
	chunkSize    int
	clearTaskNum int
	rootCmd      = &cobra.Command{
		Use:   "rss2cloud",
		Short: `Add offline tasks to 115`,
		Run: func(_cmd *cobra.Command, _args []string) {
			initAgent()
			if rssJsonPath != "" {
				rsssite.SetRssJsonPath(rssJsonPath)
			}
			if rssUrl != "" {
				pAgent.AddRssUrlTask(rssUrl)
				return
			}
			if clearTaskNum > 0 {
				err := pAgent.OfflineClear(clearTaskNum - 1)
				if err != nil {
					log.Fatalln(err)
				}
				return
			}
			pAgent.ExecuteAllRssTask()
		},
	}
	// magnet link
	linkUrl   string
	cid       string
	textFile  string
	magnetCmd = &cobra.Command{
		Use:   "magnet",
		Short: `Add magnet tasks to 115`,
		Run: func(_cmd *cobra.Command, _args []string) {
			initAgent()
			magnets := []string{}
			if textFile != "" {
				var err error
				magnets, err = rsssite.GetMagnetsFromText(textFile)
				if err != nil {
					log.Fatalln(err)
				}
			} else if linkUrl != "" {
				magnets = append(magnets, linkUrl)
			}
			if len(magnets) == 0 {
				log.Fatalln("magnets is empty")
			}
			pAgent.AddMagnetTask(magnets, cid)
		},
	}
	// server subcommand
	port int

	serverCmd = &cobra.Command{
		Use:   "server",
		Short: `Start server`,
		Run: func(_cmd *cobra.Command, _args []string) {
			initAgent()
			server.New(pAgent, port).StartServer()
		},
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&rssUrl, "url", "u", "", "rss url")
	rootCmd.Flags().StringVar(&cookies, "cookies", "", "115 cookies")
	rootCmd.Flags().StringVarP(&rssJsonPath, "rss", "r", "", "rss json path")
	rootCmd.Flags().BoolVarP(&qrLogin, "qrcode", "q", false, "login 115 by qrcode")
	magnetCmd.Flags().StringVarP(&linkUrl, "link", "l", "", "magnet link")
	magnetCmd.Flags().StringVar(&cid, "cid", "", "cid")
	magnetCmd.Flags().StringVar(&textFile, "text", "", "text file")
	rootCmd.Flags().BoolVar(&disableCache, "no-cache", false, "skip checking cache in db.sqlite")
	rootCmd.Flags().IntVar(&chunkDelay, "chunk-delay", 0, "chunk delay. default 2")
	rootCmd.Flags().IntVar(&chunkSize, "chunk-size", 0, "chunk size. default 200")
	rootCmd.Flags().IntVar(&clearTaskNum, "clear-task-type", 0, "clear offline task type: 1-6.\n 1: OfflineClearDone\n 2: OfflineClearAll\n 3: OfflineClearFailed\n 4: OfflineClearRunning\n 5: OfflineClearDoneAndDelete\n 6: OfflineClearAllAndDelete")
	rootCmd.AddCommand(magnetCmd)
	// server subcommand
	serverCmd.Flags().IntVarP(&port, "port", "p", 8115, "server port")
	rootCmd.AddCommand(serverCmd)
}

func initAgent() {
	p115.SetOption(p115.Option{DisableCache: disableCache, ChunkDelay: chunkDelay, ChunkSize: chunkSize})
	var err error
	if cookies != "" {
		pAgent, err = p115.NewAgent(cookies)
	} else if qrLogin {
		pAgent, err = p115.NewAgentByQrcode()
	} else {
		pAgent, err = p115.New()
	}
	if err != nil {
		log.Fatalln(err)
	}
}
