package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/zhifengle/rss2cloud/p115"
	"github.com/zhifengle/rss2cloud/rsssite"
)

var (
	pAgent      *p115.Agent
	rssUrl      string
	cookies     string
	rssJsonPath string
	qrLogin     bool
	rootCmd     = &cobra.Command{
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
	rootCmd.Flags().BoolVarP(&qrLogin, "qrcode", "q", false, "login 115 by qrcode. Windows only")
	magnetCmd.Flags().StringVarP(&linkUrl, "link", "l", "", "magnet link")
	magnetCmd.Flags().StringVar(&cid, "cid", "", "cid")
	magnetCmd.Flags().StringVar(&textFile, "text", "", "text file")
	rootCmd.AddCommand(magnetCmd)
}

func initAgent() {
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
