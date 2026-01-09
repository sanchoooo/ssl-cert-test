package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/andre/ssl-cert-test/config"
)

func main() {
	// 1. Load Configuration
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		// Cannot use slog yet if config failed
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	if cfg.Help {
		printHelp()
		return
	}

	// 2. Setup Structured Logger (Recommendation #2)
	logLevel := slog.LevelInfo
	if cfg.Verbose {
		logLevel = slog.LevelDebug
	}

	// Use JSON handler for production-grade structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	slog.Info("configuration loaded", "mode", cfg.ConfigType, "timeout", cfg.Timeout, "split", cfg.Split)

	// 3. Load Targets
	targets, err := loadTargets(cfg)
	if err != nil {
		slog.Error("failed to load targets", "error", err)
		os.Exit(1)
	}
	slog.Info("targets loaded", "count", len(targets.Domains), "ports", len(targets.Ports))

	// 4. Run Scan (Recommendation #3: Context)
	results := runScan(cfg, targets)

	// 5. Send Alerts
	processAlerts(cfg, results)

	// 6. Save Output
	saveOutput(cfg, results)
}

// loadTargets handles the logic for discovering domains using the Provider pattern
func loadTargets(cfg *config.AppConfig) (config.Config, error) {
	var targetConf config.Config

	// A. Parse CLI Ports
	cliPorts, err := config.ParsePorts(cfg.PortString)
	if err != nil {
		return targetConf, fmt.Errorf("error parsing ports: %w", err)
	}

	// B. Get the Correct Provider (Factory Pattern)
	provider, err := config.GetProvider(cfg)
	if err != nil {
		return targetConf, err
	}

	// C. Fetch Targets (Polymorphic Call)
	targetConf, err = provider.FetchTargets()
	if err != nil {
		return targetConf, fmt.Errorf("failed to fetch targets: %w", err)
	}

	// D. Process CIDRs (if any)
	for _, cidr := range targetConf.Cidr {
		slog.Debug("expanding cidr", "cidr", cidr)
		ips, err := config.ConvertCidrToIPList(cidr)
		if err != nil {
			return targetConf, fmt.Errorf("cidr error: %w", err)
		}
		targetConf.Domains = append(targetConf.Domains, ips...)
	}

	// E. Merge Ports
	targetConf.Ports = config.MergePorts(targetConf.Ports, cliPorts)
	if len(targetConf.Ports) == 0 {
		targetConf.Ports = config.DefaultPorts
	}

	if len(targetConf.Domains) == 0 {
		return targetConf, fmt.Errorf("no domains found to test")
	}

	return targetConf, nil
}

// runScan manages the worker pool and collects results
func runScan(cfg *config.AppConfig, targets config.Config) []config.DomainValidity {
	start := time.Now()
	totalWork := len(targets.Domains) * len(targets.Ports)
	resultsChan := make(chan config.DomainValidity, totalWork)
	var wg sync.WaitGroup

	// Create a Root Context (useful for timeouts or graceful shutdown)
	ctx := context.Background()

	// Batch processing
	for i := 0; i < len(targets.Domains); i += cfg.Split {
		end := i + cfg.Split
		if end > len(targets.Domains) {
			end = len(targets.Domains)
		}

		wg.Add(1)
		// We pass the context to ProcessDomains
		go config.ProcessDomains(
			ctx,
			targets.Domains[i:end],
			targets.Ports,
			cfg.Timeout,
			start, // Pass start time for 'NotAfter' calculation
			resultsChan,
			&wg,
		)
	}

	wg.Wait()
	close(resultsChan)

	// Collect results
	results := make([]config.DomainValidity, 0, totalWork)
	for result := range resultsChan {
		results = append(results, result)
	}

	slog.Info("scan completed", "duration", time.Since(start).String(), "results", len(results))
	return results
}

// processAlerts handles all notification channels
func processAlerts(cfg *config.AppConfig, results []config.DomainValidity) {
	if cfg.PagerDutyKey != "" {
		if err := config.SendAlert(cfg.PagerDutyKey, cfg.AlertDays, results); err != nil {
			slog.Error("pagerduty alert failed", "error", err)
		}
	}

	if cfg.SlackWebhook != "" {
		if err := config.SendSlackAlert(cfg.SlackWebhook, cfg.AlertDays, results); err != nil {
			slog.Error("slack alert failed", "error", err)
		}
	}

	if cfg.TeamsWebhook != "" {
		if err := config.SendTeamsAlert(cfg.TeamsWebhook, cfg.AlertDays, results); err != nil {
			slog.Error("teams alert failed", "error", err)
		}
	}
}

// saveOutput handles writing results to disk
func saveOutput(cfg *config.AppConfig, results []config.DomainValidity) {
	if cfg.Output == "" {
		return
	}

	if err := config.WriteOutputToFile(cfg.Output, results); err != nil {
		slog.Error("failed to write output", "error", err)
		os.Exit(1)
	}
	slog.Info("results saved", "path", cfg.Output)
}

func printHelp() {
	helpText := `Usage: ssl_cert_checker [OPTIONS]

Options can be set via flags (e.g., -timeout 10s) or env vars (e.g., SSL_TIMEOUT=10s).

Supported Modes (-type):
  - zone:       AWS Route53 Hosted Zone
  - cloudflare: Cloudflare DNS Zone
  - azure:      Azure DNS Zone
  - gitlab:     JSON file hosted in GitLab
  - config:     Local JSON config file

Example:
  ssl_cert_checker -type cloudflare -cloudflaretoken "..." -cloudflarezoneid "..."
`
	fmt.Println(helpText)
}
