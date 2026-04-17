package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/zhifengle/rss2cloud/cloudfs"
	"github.com/zhifengle/rss2cloud/config"
	"github.com/zhifengle/rss2cloud/p115"
)

// fs-level shared flags
var (
	fsCwd              string
	fsRootID           string
	fsPageSize         int
	fsOpRateLimitMinMs int
	fsOpRateLimitMaxMs int
	fsListCacheTTL     time.Duration
	fsJSON             bool
)

var fsCmd = &cobra.Command{
	Use:   "fs",
	Short: "Cloud filesystem operations",
}

func init() {
	fsCmd.PersistentFlags().StringVar(&fsCwd, "cwd", "", "starting working directory (default /)")
	fsCmd.PersistentFlags().StringVar(&fsRootID, "root-id", "", "override provider logical root ID")
	fsCmd.PersistentFlags().IntVar(&fsPageSize, "page-size", 0, "list page size hint")
	fsCmd.PersistentFlags().IntVar(&fsOpRateLimitMinMs, "op-rate-limit-min-ms", 0, "operation rate limit minimum cooldown (ms)")
	fsCmd.PersistentFlags().IntVar(&fsOpRateLimitMaxMs, "op-rate-limit-max-ms", 0, "operation rate limit maximum cooldown (ms)")
	fsCmd.PersistentFlags().DurationVar(&fsListCacheTTL, "list-cache-ttl", cloudfs.DefaultListCacheTTL, "directory list cache TTL (0 disables caching)")
	fsCmd.PersistentFlags().BoolVar(&fsJSON, "json", false, "output as JSON")

	fsCmd.AddCommand(fsPwdCmd)
	fsCmd.AddCommand(fsLsCmd)
	fsCmd.AddCommand(fsStatCmd)
	fsCmd.AddCommand(fsMkdirCmd)
	fsCmd.AddCommand(fsRenameCmd)
	fsCmd.AddCommand(fsMvCmd)
	fsCmd.AddCommand(fsRmCmd)
	fsCmd.AddCommand(fsCpCmd)
	fsCmd.AddCommand(fsFlattenCmd)
	fsCmd.AddCommand(fsSearchMvCmd)

	rootCmd.AddCommand(fsCmd)
}

// initFsSession initialises a p115.Agent and returns a cloudfs.Session.
// It reuses the top-level cookies/qrLogin flags already defined in rss2cloud.go.
func initFsSession(ctx context.Context) *cloudfs.Session {
	cliParams := buildCLIParams(nil)
	cfg, _, err := config.LoadWithOptions(cliParams, config.LoadOptions{Auth: true})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: load config: %v\n", err)
		os.Exit(1)
	}

	p115.SetOption(p115.Option{
		DisableCache:  cfg.P115.DisableCache,
		ChunkDelay:    cfg.P115.ChunkDelay,
		ChunkSize:     cfg.P115.ChunkSize,
		CooldownMinMs: cfg.P115.CooldownMinMs,
		CooldownMaxMs: cfg.P115.CooldownMaxMs,
	})

	var agent *p115.Agent
	if cfg.Auth.Cookies != "" {
		agent, err = p115.NewAgent(cfg.Auth.Cookies)
	} else if qrLogin {
		agent, err = p115.NewAgentByQrcode()
	} else {
		agent, err = p115.New()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: login failed: %v\n", err)
		os.Exit(1)
	}

	var opLimiter cloudfs.Limiter
	if fsOpRateLimitMinMs > 0 || fsOpRateLimitMaxMs > 0 {
		opLimiter = p115.NewOperationLimiter(fsOpRateLimitMinMs, fsOpRateLimitMaxMs)
	}
	opt := p115.FileSystemOption{
		RootID:           fsRootID,
		PageSize:         fsPageSize,
		OperationLimiter: opLimiter,
	}
	driver := agent.FileSystemWithOption(opt)

	session, err := cloudfs.NewSession(ctx, driver)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: init session: %v\n", err)
		os.Exit(1)
	}
	configureSessionListCacheTTL(session, fsListCacheTTL)

	if fsCwd != "" {
		if _, err := session.Cd(ctx, fsCwd); err != nil {
			log.Fatalf("error: --cwd %q: %v\n", fsCwd, err)
		}
	}
	return session
}

func configureSessionListCacheTTL(session *cloudfs.Session, ttl time.Duration) {
	if session == nil {
		return
	}
	session.SetListCacheTTL(ttl)
}

// newOperationLimiter is re-exported here for cmd layer use.
// The actual implementation lives in p115/filesystem.go.
func newOperationLimiter(minMs, maxMs int) cloudfs.Limiter {
	if minMs <= 0 && maxMs <= 0 {
		return nil
	}
	return p115.NewOperationLimiter(minMs, maxMs)
}
