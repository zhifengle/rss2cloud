package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/zhifengle/rss2cloud/config"
	"github.com/zhifengle/rss2cloud/p115"
	"github.com/zhifengle/rss2cloud/rsssite"
	"github.com/zhifengle/rss2cloud/server"
)

var (
	pAgent        *p115.Agent
	loadedConfig  *config.Config // Store loaded config for access across commands
	rssUrl        string
	cookies       string
	rssJsonPath   string
	qrLogin       bool
	disableCache  bool
	chunkDelay    int
	chunkSize     int
	cooldownMinMs int
	cooldownMaxMs int
	clearTaskNum  int
	rootCmd       = &cobra.Command{
		Use:   "rss2cloud",
		Short: `Add offline tasks to 115`,
		Run: func(_cmd *cobra.Command, _args []string) {
			initAgent(_cmd)
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
	savepath  string
	textFile  string
	magnetCmd = &cobra.Command{
		Use:   "magnet",
		Short: `Add magnet tasks to 115`,
		Run: func(_cmd *cobra.Command, _args []string) {
			initAgent(_cmd)
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
			pAgent.AddMagnetTask(magnets, cid, savepath)
		},
	}
	// server subcommand
	port int

	serverCmd = &cobra.Command{
		Use:   "server",
		Short: `Start server`,
		Run: func(_cmd *cobra.Command, _args []string) {
			initAgent(_cmd)
			// Use port from loaded config (which respects CLI > TOML > Default priority)
			serverPort := loadedConfig.Server.Port
			server.New(pAgent, serverPort).StartServer()
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
	rootCmd.PersistentFlags().StringVarP(&rssUrl, "url", "u", "", "rss url")
	rootCmd.PersistentFlags().StringVar(&cookies, "cookies", "", "115 cookies")
	rootCmd.PersistentFlags().StringVarP(&rssJsonPath, "rss", "r", "", "rss json path")
	rootCmd.PersistentFlags().BoolVarP(&qrLogin, "qrcode", "q", false, "login 115 by qrcode")
	magnetCmd.Flags().StringVarP(&linkUrl, "link", "l", "", "magnet link")
	magnetCmd.Flags().StringVar(&cid, "cid", "", "cid")
	magnetCmd.Flags().StringVar(&savepath, "savepath", "", "save path")
	magnetCmd.Flags().StringVar(&textFile, "text", "", "text file")
	rootCmd.PersistentFlags().BoolVar(&disableCache, "no-cache", false, "skip checking cache in db.sqlite")
	rootCmd.PersistentFlags().IntVar(&chunkDelay, "chunk-delay", 0, "chunk delay. default 2")
	rootCmd.PersistentFlags().IntVar(&chunkSize, "chunk-size", 0, "chunk size. default 200")
	rootCmd.PersistentFlags().IntVar(&cooldownMinMs, "cooldown-min-ms", 1000, "minimum cooldown between 115 API calls in milliseconds. default 1000")
	rootCmd.PersistentFlags().IntVar(&cooldownMaxMs, "cooldown-max-ms", 1100, "maximum cooldown between 115 API calls in milliseconds. default 1100")
	rootCmd.Flags().IntVar(&clearTaskNum, "clear-task-type", 0, "clear offline task type: 1-6.\n 1: OfflineClearDone\n 2: OfflineClearAll\n 3: OfflineClearFailed\n 4: OfflineClearRunning\n 5: OfflineClearDoneAndDelete\n 6: OfflineClearAllAndDelete")
	rootCmd.AddCommand(magnetCmd)
	// server subcommand
	serverCmd.Flags().IntVarP(&port, "port", "p", 8115, "server port")
	rootCmd.AddCommand(serverCmd)
}

func buildCLIParams(cmd *cobra.Command) config.CLIParams {
	cliParams := config.CLIParams{
		Cookies: cookies,
		RSSPath: rssJsonPath,
	}

	if commandFlagChanged(cmd, "no-cache") || (cmd == nil && disableCache) {
		cliParams.DisableCache = disableCache
		cliParams.DisableCacheSet = true
	}
	if commandFlagChanged(cmd, "chunk-delay") || (cmd == nil && chunkDelay != 0) {
		cliParams.ChunkDelay = chunkDelay
		cliParams.ChunkDelaySet = true
	}
	if commandFlagChanged(cmd, "chunk-size") || (cmd == nil && chunkSize != 0) {
		cliParams.ChunkSize = chunkSize
		cliParams.ChunkSizeSet = true
	}
	if commandFlagChanged(cmd, "cooldown-min-ms") || (cmd == nil && cooldownMinMs != 1000) {
		cliParams.CooldownMinMs = cooldownMinMs
		cliParams.CooldownMinMsSet = true
	}
	if commandFlagChanged(cmd, "cooldown-max-ms") || (cmd == nil && cooldownMaxMs != 1100) {
		cliParams.CooldownMaxMs = cooldownMaxMs
		cliParams.CooldownMaxMsSet = true
	}
	if cmd != nil && cmd.Flags().Changed("port") {
		cliParams.Port = port
		cliParams.PortSet = true
	}
	return cliParams
}

func commandFlagChanged(cmd *cobra.Command, name string) bool {
	if cmd == nil {
		return false
	}
	return cmd.Flags().Changed(name) || cmd.InheritedFlags().Changed(name) || cmd.PersistentFlags().Changed(name)
}

func initAgent(cmd *cobra.Command) {
	cliParams := buildCLIParams(cmd)

	cfg, _, err := config.LoadWithOptions(cliParams, config.LoadOptions{Auth: true})
	if err != nil {
		log.Fatalln(err)
	}

	loadedConfig = cfg

	p115.SetOption(p115.Option{
		DisableCache:  cfg.P115.DisableCache,
		ChunkDelay:    cfg.P115.ChunkDelay,
		ChunkSize:     cfg.P115.ChunkSize,
		CooldownMinMs: cfg.P115.CooldownMinMs,
		CooldownMaxMs: cfg.P115.CooldownMaxMs,
	})

	var agentErr error
	if cfg.Auth.Cookies != "" {
		pAgent, agentErr = p115.NewAgent(cfg.Auth.Cookies)
	} else if qrLogin {
		pAgent, agentErr = p115.NewAgentByQrcode()
	} else {
		pAgent, agentErr = p115.New()
	}
	if agentErr != nil {
		log.Fatalln(agentErr)
	}
}
