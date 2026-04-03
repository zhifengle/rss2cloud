package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/zhifengle/rss2cloud/cloudfs"
	"github.com/zhifengle/rss2cloud/p115"
)

// fs-level shared flags
var (
	fsCwd              string
	fsRootID           string
	fsPageSize         int
	fsOpRateLimitMinMs int
	fsOpRateLimitMaxMs int
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
	fsCmd.PersistentFlags().BoolVar(&fsJSON, "json", false, "output as JSON")

	fsCmd.AddCommand(fsPwdCmd)
	fsCmd.AddCommand(fsLsCmd)
	fsCmd.AddCommand(fsStatCmd)
	fsCmd.AddCommand(fsMkdirCmd)
	fsCmd.AddCommand(fsRenameCmd)
	fsCmd.AddCommand(fsMvCmd)
	fsCmd.AddCommand(fsRmCmd)
	fsCmd.AddCommand(fsCpCmd)

	rootCmd.AddCommand(fsCmd)
}

// initFsSession initialises a p115.Agent and returns a cloudfs.Session.
// It reuses the top-level cookies/qrLogin flags already defined in rss2cloud.go.
func initFsSession(ctx context.Context) *cloudfs.Session {
	p115.SetOption(p115.Option{
		CooldownMinMs: cooldownMinMs,
		CooldownMaxMs: cooldownMaxMs,
	})

	var agent *p115.Agent
	var err error
	if cookies != "" {
		agent, err = p115.NewAgent(cookies)
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

	if fsCwd != "" {
		if _, err := session.Cd(ctx, fsCwd); err != nil {
			log.Fatalf("error: --cwd %q: %v\n", fsCwd, err)
		}
	}
	return session
}

// newOperationLimiter is re-exported here for cmd layer use.
// The actual implementation lives in p115/filesystem.go.
func newOperationLimiter(minMs, maxMs int) cloudfs.Limiter {
	if minMs <= 0 && maxMs <= 0 {
		return nil
	}
	return p115.NewOperationLimiter(minMs, maxMs)
}
