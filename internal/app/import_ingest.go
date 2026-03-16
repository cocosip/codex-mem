package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"codex-mem/internal/config"
	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/imports"
	"codex-mem/internal/domain/memory"
	"codex-mem/internal/domain/scope"
	"codex-mem/internal/domain/session"
)

type ingestImportsOptions struct {
	Source     imports.Source
	InputPath  string
	CWD        string
	BranchName string
	RepoRemote string
	Task       string
	JSON       bool
}

type ingestImportEvent struct {
	ExternalID        string   `json:"external_id,omitempty"`
	PayloadHash       string   `json:"payload_hash,omitempty"`
	Type              string   `json:"type"`
	Title             string   `json:"title"`
	Content           string   `json:"content"`
	Importance        int      `json:"importance"`
	Tags              []string `json:"tags,omitempty"`
	FilePaths         []string `json:"file_paths,omitempty"`
	RelatedProjectIDs []string `json:"related_project_ids,omitempty"`
	Status            string   `json:"status,omitempty"`
	PrivacyIntent     string   `json:"privacy_intent,omitempty"`
}

type ingestImportEventResult struct {
	Line               int    `json:"line"`
	NoteID             string `json:"note_id,omitempty"`
	ImportID           string `json:"import_id"`
	Materialized       bool   `json:"materialized"`
	Suppressed         bool   `json:"suppressed"`
	NoteDeduplicated   bool   `json:"note_deduplicated"`
	ImportDeduplicated bool   `json:"import_deduplicated"`
}

type ingestImportsReport struct {
	Source             string                    `json:"source"`
	Input              string                    `json:"input"`
	Scope              scope.Scope               `json:"scope"`
	Session            session.Session           `json:"session"`
	Processed          int                       `json:"processed"`
	Materialized       int                       `json:"materialized"`
	Suppressed         int                       `json:"suppressed"`
	NoteDeduplicated   int                       `json:"note_deduplicated"`
	ImportDeduplicated int                       `json:"import_deduplicated"`
	Warnings           []common.Warning          `json:"warnings,omitempty"`
	Results            []ingestImportEventResult `json:"results,omitempty"`
}

func runIngestImports(ctx context.Context, cfg config.Config, stdin io.Reader, stdout io.Writer, args []string) error {
	options, err := parseIngestImportsOptions(args)
	if err != nil {
		return err
	}

	cwd := strings.TrimSpace(options.CWD)
	if cwd == "" {
		cwd, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve working directory: %w", err)
		}
	}

	input, inputLabel, closeFn, err := openIngestImportsInput(options.InputPath, cwd, stdin)
	if err != nil {
		return err
	}
	if closeFn != nil {
		defer closeFn()
	}

	instance, err := New(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		_ = instance.Close()
	}()

	scopeOutput, err := instance.ScopeService.Resolve(ctx, scope.ResolveInput{
		CWD:        cwd,
		BranchName: options.BranchName,
		RepoRemote: options.RepoRemote,
	})
	if err != nil {
		return err
	}

	task := strings.TrimSpace(options.Task)
	if task == "" {
		task = fmt.Sprintf("ingest imported notes (%s)", options.Source)
	}
	sessionOutput, err := instance.SessionService.Start(ctx, session.StartInput{
		Scope:      scopeOutput.Scope,
		Task:       task,
		BranchName: options.BranchName,
	})
	if err != nil {
		return err
	}

	report, err := ingestImportEvents(ctx, input, inputLabel, options.Source, scopeOutput.Scope, sessionOutput.Session, scopeOutput.Warnings, instance.ImportService)
	if err != nil {
		return err
	}

	if options.JSON {
		body, err := marshalIndented(report)
		if err != nil {
			return err
		}
		_, err = io.WriteString(stdout, body)
		return err
	}

	_, err = io.WriteString(stdout, formatIngestImportsReport(report))
	return err
}

func parseIngestImportsOptions(args []string) (ingestImportsOptions, error) {
	var options ingestImportsOptions

	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		switch arg {
		case "":
			continue
		case "--source":
			value, next, err := optionValue(args, i)
			if err != nil {
				return ingestImportsOptions{}, err
			}
			options.Source = imports.Source(value)
			i = next
		case "--input":
			value, next, err := optionValue(args, i)
			if err != nil {
				return ingestImportsOptions{}, err
			}
			options.InputPath = value
			i = next
		case "--cwd":
			value, next, err := optionValue(args, i)
			if err != nil {
				return ingestImportsOptions{}, err
			}
			options.CWD = value
			i = next
		case "--branch-name":
			value, next, err := optionValue(args, i)
			if err != nil {
				return ingestImportsOptions{}, err
			}
			options.BranchName = value
			i = next
		case "--repo-remote":
			value, next, err := optionValue(args, i)
			if err != nil {
				return ingestImportsOptions{}, err
			}
			options.RepoRemote = value
			i = next
		case "--task":
			value, next, err := optionValue(args, i)
			if err != nil {
				return ingestImportsOptions{}, err
			}
			options.Task = value
			i = next
		case "--json":
			options.JSON = true
		default:
			return ingestImportsOptions{}, fmt.Errorf("unknown ingest-imports flag %q", arg)
		}
	}

	if err := options.Source.Validate(); err != nil {
		if strings.TrimSpace(string(options.Source)) == "" {
			return ingestImportsOptions{}, fmt.Errorf("ingest-imports source is required")
		}
		return ingestImportsOptions{}, err
	}

	return options, nil
}

func openIngestImportsInput(path string, cwd string, stdin io.Reader) (io.Reader, string, func(), error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return stdin, "stdin", nil, nil
	}
	if !filepath.IsAbs(trimmed) {
		trimmed = filepath.Join(cwd, trimmed)
	}
	absPath, err := filepath.Abs(trimmed)
	if err != nil {
		return nil, "", nil, fmt.Errorf("resolve ingest-imports input path: %w", err)
	}
	file, err := os.Open(absPath)
	if err != nil {
		return nil, "", nil, fmt.Errorf("open ingest-imports input: %w", err)
	}
	return file, absPath, func() { _ = file.Close() }, nil
}

func ingestImportEvents(ctx context.Context, reader io.Reader, inputLabel string, source imports.Source, resolvedScope scope.Scope, sess session.Session, scopeWarnings []common.Warning, importService *imports.Service) (ingestImportsReport, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	report := ingestImportsReport{
		Source:   string(source),
		Input:    inputLabel,
		Scope:    resolvedScope,
		Session:  sess,
		Warnings: scopeWarnings,
	}

	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var event ingestImportEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return ingestImportsReport{}, fmt.Errorf("decode ingest-imports event on line %d: %w", lineNumber, err)
		}

		result, err := importService.SaveImportedNote(ctx, imports.SaveImportedNoteInput{
			Scope:             resolvedScope.Ref(),
			SessionID:         sess.ID,
			Source:            source,
			ExternalID:        event.ExternalID,
			PayloadHash:       event.PayloadHash,
			Type:              memory.NoteType(event.Type),
			Title:             event.Title,
			Content:           event.Content,
			Importance:        event.Importance,
			Tags:              event.Tags,
			FilePaths:         event.FilePaths,
			RelatedProjectIDs: event.RelatedProjectIDs,
			Status:            memory.Status(event.Status),
			PrivacyIntent:     event.PrivacyIntent,
		})
		if err != nil {
			return ingestImportsReport{}, fmt.Errorf("ingest-imports event on line %d: %w", lineNumber, err)
		}

		report.Processed++
		if result.Materialized {
			report.Materialized++
		}
		if result.Suppressed {
			report.Suppressed++
		}
		if result.NoteDeduplicated {
			report.NoteDeduplicated++
		}
		if result.ImportDeduplicated {
			report.ImportDeduplicated++
		}
		report.Warnings = common.MergeWarnings(report.Warnings, result.Warnings)

		eventResult := ingestImportEventResult{
			Line:               lineNumber,
			ImportID:           result.Import.ID,
			Materialized:       result.Materialized,
			Suppressed:         result.Suppressed,
			NoteDeduplicated:   result.NoteDeduplicated,
			ImportDeduplicated: result.ImportDeduplicated,
		}
		if result.Note != nil {
			eventResult.NoteID = result.Note.ID
		}
		report.Results = append(report.Results, eventResult)
	}
	if err := scanner.Err(); err != nil {
		return ingestImportsReport{}, fmt.Errorf("read ingest-imports input: %w", err)
	}
	if report.Processed == 0 {
		return ingestImportsReport{}, fmt.Errorf("ingest-imports input did not contain any events")
	}
	return report, nil
}

func formatIngestImportsReport(report ingestImportsReport) string {
	lines := []string{
		"ingest imports ok",
		fmt.Sprintf("source=%s", report.Source),
		fmt.Sprintf("input=%s", report.Input),
		fmt.Sprintf("session_id=%s", report.Session.ID),
		fmt.Sprintf("resolved_by=%s", report.Scope.ResolvedBy),
		fmt.Sprintf("processed=%d", report.Processed),
		fmt.Sprintf("materialized=%d", report.Materialized),
		fmt.Sprintf("suppressed=%d", report.Suppressed),
		fmt.Sprintf("note_deduplicated=%d", report.NoteDeduplicated),
		fmt.Sprintf("import_deduplicated=%d", report.ImportDeduplicated),
		fmt.Sprintf("warnings=%d", len(report.Warnings)),
	}
	return strings.Join(lines, "\n") + "\n"
}

func marshalIndented(value any) (string, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return "", fmt.Errorf("marshal ingest-imports report: %w", err)
	}
	return buffer.String(), nil
}
