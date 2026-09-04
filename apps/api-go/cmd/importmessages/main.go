// Command importmessages backfills chat_messages from a JSON message export
// (the "items" shape returned by the app's own message export/search paths).
// It shares its mapping, validation, and dedup rules with the admin panel
// import through internal/usecase/messageimport, and is append-only: existing
// rows are looked up by kick_message_id and are never updated. Nothing is
// written unless -dry-run=false is combined with the exact -confirm phrase.
package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	clickhouseinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/clickhouse"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/messageimport"
)

func main() {
	input := flag.String("input", "", "path to the JSON export file (required)")
	dryRun := flag.Bool("dry-run", true, "analyze and report only; no ClickHouse writes (default true)")
	limit := flag.Int("limit", 0, "only process the first N records from the export (0 = all)")
	confirm := flag.String("confirm", "", "must equal "+messageimport.ConfirmationText+" to allow a real write when -dry-run=false")
	verifyCSV := flag.String("verify-csv", "", "optional: path to a CSV export of the same dataset, cross-checked by kick_message_id only")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	if strings.TrimSpace(*input) == "" {
		logger.Error("missing required -input flag")
		os.Exit(1)
	}

	payload, err := os.ReadFile(*input)
	if err != nil {
		logger.Error("failed to read input file", "path", *input, "error", err)
		os.Exit(1)
	}

	file, err := messageimport.ParseExport(payload)
	if err != nil {
		logger.Error("failed to parse input JSON", "path", *input, "error", err)
		os.Exit(1)
	}
	logger.Info(
		"loaded export file",
		"path", *input,
		"declared_count", file.Count,
		"items_in_file", len(file.Items),
		"truncated", file.Truncated,
	)

	if *verifyCSV != "" {
		if err := verifyAgainstCSV(*verifyCSV, file.Items, logger); err != nil {
			logger.Error("csv verification failed", "error", err)
			os.Exit(1)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	conn, err := clickhouseinfra.Open(ctx, cfg)
	if err != nil {
		logger.Error("failed to open clickhouse connection", "error", err)
		os.Exit(1)
	}
	// The CLI is the unbounded path: an operator running it on the host is
	// trusted with a full export, unlike the size-capped admin upload.
	service := messageimport.NewService(clickhouseinfra.NewMessageRepository(conn), 0)

	preview, err := service.Preview(ctx, payload, *limit)
	if err != nil {
		logger.Error("failed to analyze export", "error", err)
		os.Exit(1)
	}
	printPreview(logger, preview)

	if *dryRun {
		logger.Info("dry-run finished; no rows were written")
		logger.Info(
			"before running with -dry-run=false, back up the clickhouse_data volume " +
				"(see docs/operations/backup_restore.md); this import only INSERTs rows for " +
				"kick_message_ids that do not already exist, so a backup lets you fully undo the run",
		)
		return
	}

	result, err := service.Confirm(ctx, payload, *limit, *confirm)
	if err != nil {
		logger.Error(
			"import refused or failed",
			"error", err,
			"required_confirm", messageimport.ConfirmationText,
		)
		os.Exit(1)
	}
	logger.Info(
		"import complete",
		"written", result.Written,
		"skipped_already_existed", result.AlreadyExists,
		"skipped_duplicate_in_file", result.DuplicateInFile,
		"skipped_invalid", result.Invalid,
	)
}

func printPreview(logger *slog.Logger, preview domain.MessageImportPreview) {
	logger.Info(
		"import plan summary",
		"records_read", preview.RecordsRead,
		"would_insert", preview.ToInsert,
		"already_in_clickhouse", preview.AlreadyExists,
		"duplicate_within_file", preview.DuplicateInFile,
		"invalid_records", preview.Invalid,
		"can_execute", preview.CanExecute,
	)
	if preview.Reason != "" {
		logger.Info("import cannot execute", "reason", preview.Reason)
	}
	if len(preview.SampleToInsertIDs) > 0 {
		logger.Info("sample kick_message_ids that would be inserted", "sample", preview.SampleToInsertIDs)
	}
	for _, reason := range preview.InvalidReasons {
		logger.Warn("invalid record reason", "reason", reason.Reason, "count", reason.Count, "example", reason.Example)
	}
}

func verifyAgainstCSV(path string, jsonItems []messageimport.ExportItem, logger *slog.Logger) error {
	handle, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open csv: %w", err)
	}
	defer handle.Close()

	reader := csv.NewReader(handle)
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("read csv header: %w", err)
	}
	column := -1
	for index, name := range header {
		if strings.TrimSpace(name) == "kick_message_id" {
			column = index
			break
		}
	}
	if column == -1 {
		return fmt.Errorf("csv header has no kick_message_id column")
	}

	csvIDs := make(map[string]bool)
	for {
		row, err := reader.Read()
		if err != nil {
			break
		}
		if column < len(row) {
			if id := strings.TrimSpace(row[column]); id != "" {
				csvIDs[id] = true
			}
		}
	}

	jsonIDs := make(map[string]bool, len(jsonItems))
	for _, item := range jsonItems {
		if id := strings.TrimSpace(item.KickMessageID); id != "" {
			jsonIDs[id] = true
		}
	}

	onlyInJSON := make([]string, 0)
	for id := range jsonIDs {
		if !csvIDs[id] {
			onlyInJSON = append(onlyInJSON, id)
		}
	}
	onlyInCSV := make([]string, 0)
	for id := range csvIDs {
		if !jsonIDs[id] {
			onlyInCSV = append(onlyInCSV, id)
		}
	}

	logger.Info(
		"csv cross-check (kick_message_id sets, informational only, does not affect import)",
		"csv_path", path,
		"json_ids", len(jsonIDs),
		"csv_ids", len(csvIDs),
		"only_in_json", len(onlyInJSON),
		"only_in_csv", len(onlyInCSV),
	)
	if len(onlyInJSON) > 0 || len(onlyInCSV) > 0 {
		logger.Warn(
			"json/csv kick_message_id sets differ",
			"sample_only_in_json", sample(onlyInJSON),
			"sample_only_in_csv", sample(onlyInCSV),
		)
	}
	return nil
}

func sample(list []string) []string {
	if len(list) > 5 {
		return list[:5]
	}
	return list
}
