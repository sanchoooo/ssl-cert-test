package main

import (
	"context"
	"fmt"
	"log/slog" // Ensure you use slog for structured logging
	"os"
	"sync"
	"time"

	"github.com/andre/ssl-cert-test/internal/alerting"
	"github.com/andre/ssl-cert-test/internal/config"
	"github.com/andre/ssl-cert-test/internal/discovery"
	"github.com/andre/ssl-cert-test/internal/scan"
)

func main() {
	// 1. Load Configuration
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	if cfg.Help {
		printHelp()
		return
	}

	// 2. Setup Structured Logger
	setupLogging(cfg.Verbose)
	slog.Info("configuration loaded", "mode", cfg.ConfigType, "timeout", cfg.Timeout)

	// 3. Load Targets
	targets, err := loadTargets(cfg)
	if err != nil {
		slog.Error("failed to load targets", "error", err)
		os.Exit(1)
	}
	slog.Info("targets loaded", "domains", len(targets.Domains), "ports", len(targets.Ports))

	// 4. Run Scan
	results := runScan(cfg, targets)

	// 5. Send Alerts (Refactored)
	processAlerts(cfg, results)

	// 6. Save Output
	saveOutput(cfg, results)
}

func setupLogging(verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	// Use JSON handler for production-grade logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)
}

func loadTargets(cfg *config.AppConfig) (config.Config, error) {
	var targetConf config.Config

	cliPorts, err := config.ParsePorts(cfg.PortString)
	if err != nil {
		return targetConf, fmt.Errorf("error parsing ports: %w", err)
	}

	// Use Provider Factory
	provider, err := config.GetProvider(cfg)
	if err != nil {
		return targetConf, err
	}

	targetConf, err = provider.FetchTargets()
	if err != nil {
		return targetConf, fmt.Errorf("failed to fetch targets: %w", err)
	}

	// Expand CIDRs
	for _, cidr := range targetConf.Cidr {
		slog.Debug("expanding cidr", "cidr", cidr)
		ips, err := config.ConvertCidrToIPList(cidr)
		if err != nil {
			return targetConf, fmt.Errorf("cidr error: %w", err)
		}
		targetConf.Domains = append(targetConf.Domains, ips...)
	}

	// Merge Ports
	targetConf.Ports = config.MergePorts(targetConf.Ports, cliPorts)
	if len(targetConf.Ports) == 0 {
		targetConf.Ports = config.DefaultPorts
	}

	if len(targetConf.Domains) == 0 {
		return targetConf, fmt.Errorf("no domains found to test")
	}

	return targetConf, nil
}

func runScan(cfg *config.AppConfig, targets config.Config) []config.DomainValidity {
	start := time.Now()
	totalWork := len(targets.Domains) * len(targets.Ports)
	resultsChan := make(chan config.DomainValidity, totalWork)
	var wg sync.WaitGroup
	ctx := context.Background()

	for i := 0; i < len(targets.Domains); i += cfg.Split {
		end := i + cfg.Split
		if end > len(targets.Domains) {
			end = len(targets.Domains)
		}

		wg.Add(1)
		go config.ProcessDomains(
			ctx,
			targets.Domains[i:end],
			targets.Ports,
			cfg.Timeout,
			start,
			resultsChan,
			&wg,
		)
	}

	wg.Wait()
	close(resultsChan)

	var results []config.DomainValidity
	for result := range resultsChan {
		results = append(results, result)
	}

	slog.Info("scan completed", "duration", time.Since(start).String(), "results", len(results))
	return results
}

// processAlerts now iterates over the provider list
func processAlerts(cfg *config.AppConfig, results []config.DomainValidity) {
	providers := config.GetAlertProviders(cfg)

	for _, p := range providers {
		slog.Info("sending alert", "provider", p.Name())
		if err := p.Send(results, cfg.AlertDays); err != nil {
			slog.Error("alert failed", "provider", p.Name(), "error", err)
		}
	}
}

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
	fmt.Println(`Usage: ssl_cert_checker [OPTIONS]
Options can be set via flags (e.g. -timeout 10s) or env vars (e.g. SSL_TIMEOUT=10s)`)
}
