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
	"sort"
	"strings"

	"codex-mem/internal/config"
	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/imports"
	"codex-mem/internal/domain/memory"
	"codex-mem/internal/domain/scope"
	"codex-mem/internal/domain/session"
)

type ingestImportsOptions struct {
	Source             imports.Source
	InputPath          string
	FailedOutputPath   string
	FailedManifestPath string
	CWD                string
	BranchName         string
	RepoRemote         string
	Task               string
	JSON               bool
	AuditOnly          bool
	ContinueOnError    bool
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

// IngestImportsReport is the reusable app-level report shape for imported-note batches.
type IngestImportsReport = ingestImportsReport

// IngestImportEventResult is the reusable app-level per-line result shape for imported-note batches.
type IngestImportEventResult = ingestImportEventResult

// IngestFailureDetail is the reusable app-level failure detail shape for imported-note batches.
type IngestFailureDetail = ingestFailureDetail

// IngestImportsInput is the reusable app-level request for imported-note batch ingestion.
type IngestImportsInput struct {
	Source             imports.Source
	Reader             io.Reader
	InputLabel         string
	CWD                string
	BranchName         string
	RepoRemote         string
	Task               string
	AuditOnly          bool
	ContinueOnError    bool
	FailedOutputPath   string
	FailedManifestPath string
}

type ingestImportEventResult struct {
	Line               int                  `json:"line"`
	NoteID             string               `json:"note_id,omitempty"`
	ImportID           string               `json:"import_id"`
	Materialized       bool                 `json:"materialized"`
	Suppressed         bool                 `json:"suppressed"`
	SuppressionReason  string               `json:"suppression_reason,omitempty"`
	NoteDeduplicated   bool                 `json:"note_deduplicated"`
	ImportDeduplicated bool                 `json:"import_deduplicated"`
	Error              *common.ErrorPayload `json:"error,omitempty"`
}

type ingestImportsReport struct {
	Status              string                    `json:"status"`
	Source              string                    `json:"source"`
	Input               string                    `json:"input"`
	FailedOutput        string                    `json:"failed_output,omitempty"`
	FailedOutputWritten int                       `json:"failed_output_written,omitempty"`
	FailedManifest      string                    `json:"failed_manifest,omitempty"`
	FailedManifestCount int                       `json:"failed_manifest_count,omitempty"`
	Scope               scope.Scope               `json:"scope"`
	Session             session.Session           `json:"session"`
	AuditOnly           bool                      `json:"audit_only"`
	ContinueOnError     bool                      `json:"continue_on_error"`
	Attempted           int                       `json:"attempted"`
	Processed           int                       `json:"processed"`
	Failed              int                       `json:"failed"`
	Materialized        int                       `json:"materialized"`
	Suppressed          int                       `json:"suppressed"`
	SuppressionReasons  map[string]int            `json:"suppression_reasons,omitempty"`
	WouldMaterialize    int                       `json:"would_materialize"`
	LinkedExistingNote  int                       `json:"linked_existing_note"`
	NoteDeduplicated    int                       `json:"note_deduplicated"`
	ImportDeduplicated  int                       `json:"import_deduplicated"`
	Warnings            []common.Warning          `json:"warnings,omitempty"`
	Results             []ingestImportEventResult `json:"results,omitempty"`
}

type ingestFailureDetail struct {
	Line             int                 `json:"line"`
	Error            common.ErrorPayload `json:"error"`
	RawLine          string              `json:"raw_line"`
	FailedOutputLine int                 `json:"failed_output_line,omitempty"`
}

type ingestFailureManifest struct {
	Status              string                `json:"status"`
	Source              string                `json:"source"`
	Input               string                `json:"input"`
	FailedOutput        string                `json:"failed_output,omitempty"`
	FailedOutputWritten int                   `json:"failed_output_written,omitempty"`
	FailureCount        int                   `json:"failure_count"`
	Failures            []ingestFailureDetail `json:"failures"`
}

const (
	ingestImportsInputFlag           = "--input"
	ingestImportsFailedOutputFlag    = "--failed-output"
	ingestImportsFailedManifestFlag  = "--failed-manifest"
	ingestImportsCWDFlag             = "--cwd"
	ingestImportsContinueOnErrorFlag = "--continue-on-error"
)

func runIngestImports(ctx context.Context, cfg config.Config, stdin io.Reader, stdout io.Writer, args []string) error {
	options, err := parseIngestImportsOptions(args)
	if err != nil {
		return err
	}

	input, inputLabel, closeFn, err := openIngestImportsInput(options.InputPath, options.CWD, stdin)
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

	report, err := instance.IngestImports(ctx, IngestImportsInput{
		Source:             options.Source,
		Reader:             input,
		InputLabel:         inputLabel,
		CWD:                options.CWD,
		BranchName:         options.BranchName,
		RepoRemote:         options.RepoRemote,
		Task:               options.Task,
		AuditOnly:          options.AuditOnly,
		ContinueOnError:    options.ContinueOnError,
		FailedOutputPath:   options.FailedOutputPath,
		FailedManifestPath: options.FailedManifestPath,
	})
	if err != nil {
		if options.ContinueOnError && report.Attempted > 0 {
			if writeErr := writeIngestImportsReport(stdout, report, options.JSON); writeErr != nil {
				return writeErr
			}
		}
		return err
	}
	return writeIngestImportsReport(stdout, report, options.JSON)
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
		case ingestImportsInputFlag:
			value, next, err := optionValue(args, i)
			if err != nil {
				return ingestImportsOptions{}, err
			}
			options.InputPath = value
			i = next
		case ingestImportsFailedOutputFlag:
			value, next, err := optionValue(args, i)
			if err != nil {
				return ingestImportsOptions{}, err
			}
			options.FailedOutputPath = value
			i = next
		case ingestImportsFailedManifestFlag:
			value, next, err := optionValue(args, i)
			if err != nil {
				return ingestImportsOptions{}, err
			}
			options.FailedManifestPath = value
			i = next
		case ingestImportsCWDFlag:
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
		case doctorJSONFlag:
			options.JSON = true
		case "--audit-only":
			options.AuditOnly = true
		case ingestImportsContinueOnErrorFlag:
			options.ContinueOnError = true
		default:
			return ingestImportsOptions{}, fmt.Errorf("unknown ingest-imports flag %q", arg)
		}
	}

	if err := validateIngestImportsOptions(options.Source, options.ContinueOnError, options.FailedOutputPath, options.FailedManifestPath); err != nil {
		return ingestImportsOptions{}, err
	}

	return options, nil
}

// IngestImports runs the reusable imported-note batch workflow directly against an initialized app.
func (a *App) IngestImports(ctx context.Context, input IngestImportsInput) (IngestImportsReport, error) {
	if a == nil {
		return ingestImportsReport{}, fmt.Errorf("app is required")
	}
	if err := validateIngestImportsOptions(input.Source, input.ContinueOnError, input.FailedOutputPath, input.FailedManifestPath); err != nil {
		return ingestImportsReport{}, err
	}
	if input.Reader == nil {
		return ingestImportsReport{}, fmt.Errorf("ingest-imports reader is required")
	}

	cwd, err := resolveIngestImportsCWD(input.CWD)
	if err != nil {
		return ingestImportsReport{}, err
	}
	inputLabel := strings.TrimSpace(input.InputLabel)
	if inputLabel == "" {
		inputLabel = "embedded"
	}

	failedOutputPath, err := resolveIngestImportsPath(input.FailedOutputPath, cwd)
	if err != nil {
		return ingestImportsReport{}, err
	}
	failedWriter := newIngestFailedOutputWriter(failedOutputPath)
	failedManifestPath, err := resolveIngestImportsPath(input.FailedManifestPath, cwd)
	if err != nil {
		return ingestImportsReport{}, err
	}

	scopeOutput, err := a.ScopeService.Resolve(ctx, scope.ResolveInput{
		CWD:        cwd,
		BranchName: input.BranchName,
		RepoRemote: input.RepoRemote,
	})
	if err != nil {
		return ingestImportsReport{}, err
	}

	task := strings.TrimSpace(input.Task)
	if task == "" {
		if input.AuditOnly {
			task = fmt.Sprintf("audit imported notes (%s)", input.Source)
		} else {
			task = fmt.Sprintf("ingest imported notes (%s)", input.Source)
		}
	}
	sessionOutput, err := a.SessionService.Start(ctx, session.StartInput{
		Scope:      scopeOutput.Scope,
		Task:       task,
		BranchName: input.BranchName,
	})
	if err != nil {
		return ingestImportsReport{}, err
	}

	report, failures, err := ingestImportEvents(ctx, input.Reader, inputLabel, input.Source, scopeOutput.Scope, sessionOutput.Session, scopeOutput.Warnings, input.AuditOnly, input.ContinueOnError, failedWriter, a.ImportService)
	report.FailedOutput = failedWriter.Path()
	report.FailedOutputWritten = failedWriter.Written()
	report.FailedManifest = failedManifestPath
	report.FailedManifestCount = len(failures)

	closeErr := failedWriter.Close()
	if closeErr == nil {
		closeErr = writeIngestFailureManifest(failedManifestPath, report, failures)
	}
	if closeErr != nil {
		return report, closeErr
	}
	return report, err
}

func validateIngestImportsOptions(source imports.Source, continueOnError bool, failedOutputPath string, failedManifestPath string) error {
	if err := source.Validate(); err != nil {
		if strings.TrimSpace(string(source)) == "" {
			return fmt.Errorf("ingest-imports source is required")
		}
		return err
	}
	if strings.TrimSpace(failedOutputPath) != "" && !continueOnError {
		return fmt.Errorf("ingest-imports %s requires %s", ingestImportsFailedOutputFlag, ingestImportsContinueOnErrorFlag)
	}
	if strings.TrimSpace(failedManifestPath) != "" && !continueOnError {
		return fmt.Errorf("ingest-imports %s requires %s", ingestImportsFailedManifestFlag, ingestImportsContinueOnErrorFlag)
	}
	return nil
}

func openIngestImportsInput(path string, cwd string, stdin io.Reader) (io.Reader, string, func(), error) {
	resolvedCWD, err := resolveIngestImportsCWD(cwd)
	if err != nil {
		return nil, "", nil, err
	}
	absPath, err := resolveIngestImportsPath(path, resolvedCWD)
	if err != nil {
		return nil, "", nil, err
	}
	if absPath == "" {
		return stdin, "stdin", nil, nil
	}
	file, err := os.Open(absPath)
	if err != nil {
		return nil, "", nil, fmt.Errorf("open ingest-imports input: %w", err)
	}
	return file, absPath, func() { _ = file.Close() }, nil
}

func resolveIngestImportsCWD(cwd string) (string, error) {
	cwd = strings.TrimSpace(cwd)
	if cwd != "" {
		return cwd, nil
	}
	resolved, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	return resolved, nil
}

func resolveIngestImportsPath(path string, cwd string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", nil
	}
	if !filepath.IsAbs(trimmed) {
		trimmed = filepath.Join(cwd, trimmed)
	}
	absPath, err := filepath.Abs(trimmed)
	if err != nil {
		return "", fmt.Errorf("resolve ingest-imports path: %w", err)
	}
	return absPath, nil
}

func ingestImportEvents(ctx context.Context, reader io.Reader, inputLabel string, source imports.Source, resolvedScope scope.Scope, sess session.Session, scopeWarnings []common.Warning, auditOnly bool, continueOnError bool, failedWriter *ingestFailedOutputWriter, importService *imports.Service) (ingestImportsReport, []ingestFailureDetail, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	report := ingestImportsReport{
		Status:          "ok",
		Source:          string(source),
		Input:           inputLabel,
		Scope:           resolvedScope,
		Session:         sess,
		AuditOnly:       auditOnly,
		ContinueOnError: continueOnError,
		Warnings:        scopeWarnings,
	}
	failures := make([]ingestFailureDetail, 0)

	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		rawLine := scanner.Text()
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		report.Attempted++

		var event ingestImportEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			if !continueOnError {
				return report, failures, fmt.Errorf("decode ingest-imports event on line %d: %w", lineNumber, err)
			}
			payload := common.ErrorDetails(common.WrapError(common.ErrInvalidInput, "decode ingest-imports event failed", err), common.ErrInvalidInput, "decode ingest-imports event failed")
			failedOutputLine, appendErr := failedWriter.Append(rawLine)
			if appendErr != nil {
				return report, failures, appendErr
			}
			report.appendFailure(lineNumber, payload)
			failures = append(failures, ingestFailureDetail{
				Line:             lineNumber,
				Error:            payload,
				RawLine:          rawLine,
				FailedOutputLine: failedOutputLine,
			})
			continue
		}

		saveInput := imports.SaveImportedNoteInput{
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
		}
		var (
			result imports.SaveImportedNoteOutput
			err    error
		)
		if auditOnly {
			result, err = importService.AuditImportedNote(ctx, saveInput)
		} else {
			result, err = importService.SaveImportedNote(ctx, saveInput)
		}
		if err != nil {
			if !continueOnError {
				return report, failures, fmt.Errorf("ingest-imports event on line %d: %w", lineNumber, err)
			}
			payload := common.ErrorDetails(err, common.ErrWriteFailed, "ingest-imports event failed")
			failedOutputLine, appendErr := failedWriter.Append(rawLine)
			if appendErr != nil {
				return report, failures, appendErr
			}
			report.appendFailure(lineNumber, payload)
			failures = append(failures, ingestFailureDetail{
				Line:             lineNumber,
				Error:            payload,
				RawLine:          rawLine,
				FailedOutputLine: failedOutputLine,
			})
			continue
		}

		report.Processed++
		if result.Materialized {
			report.Materialized++
		}
		if result.Suppressed {
			report.Suppressed++
			report.incrementSuppressionReason(result.Import.SuppressionReason)
		}
		if auditOnly && !result.Suppressed {
			if result.NoteDeduplicated {
				report.LinkedExistingNote++
			} else {
				report.WouldMaterialize++
			}
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
			SuppressionReason:  result.Import.SuppressionReason,
			NoteDeduplicated:   result.NoteDeduplicated,
			ImportDeduplicated: result.ImportDeduplicated,
		}
		if result.Note != nil {
			eventResult.NoteID = result.Note.ID
		}
		report.Results = append(report.Results, eventResult)
	}
	if err := scanner.Err(); err != nil {
		return report, failures, fmt.Errorf("read ingest-imports input: %w", err)
	}
	report.updateStatus()
	if report.Attempted == 0 {
		return report, failures, fmt.Errorf("ingest-imports input did not contain any events")
	}
	if continueOnError && report.Processed == 0 {
		return report, failures, common.NewError(common.ErrWriteFailed, "ingest-imports did not import any events successfully")
	}
	return report, failures, nil
}

type ingestFailedOutputWriter struct {
	path    string
	file    *os.File
	written int
}

func newIngestFailedOutputWriter(path string) *ingestFailedOutputWriter {
	return &ingestFailedOutputWriter{path: strings.TrimSpace(path)}
}

func (w *ingestFailedOutputWriter) Path() string {
	return w.path
}

func (w *ingestFailedOutputWriter) Written() int {
	return w.written
}

func (w *ingestFailedOutputWriter) Append(line string) (int, error) {
	if strings.TrimSpace(w.path) == "" {
		return 0, nil
	}
	if err := w.ensureOpen(); err != nil {
		return 0, err
	}
	if _, err := io.WriteString(w.file, line+"\n"); err != nil {
		return 0, fmt.Errorf("write ingest-imports failed output: %w", err)
	}
	w.written++
	return w.written, nil
}

func (w *ingestFailedOutputWriter) Close() error {
	if w.file == nil {
		return nil
	}
	if err := w.file.Close(); err != nil {
		return fmt.Errorf("close ingest-imports failed output: %w", err)
	}
	w.file = nil
	return nil
}

func (w *ingestFailedOutputWriter) ensureOpen() error {
	if w.file != nil || strings.TrimSpace(w.path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(w.path), 0o755); err != nil {
		return fmt.Errorf("prepare ingest-imports failed output directory: %w", err)
	}
	file, err := os.Create(w.path)
	if err != nil {
		return fmt.Errorf("open ingest-imports failed output: %w", err)
	}
	w.file = file
	return nil
}

func (r *ingestImportsReport) appendFailure(line int, payload common.ErrorPayload) {
	r.Failed++
	r.Results = append(r.Results, ingestImportEventResult{
		Line:  line,
		Error: &payload,
	})
	r.updateStatus()
}

func (r *ingestImportsReport) updateStatus() {
	switch {
	case r.Failed == 0:
		r.Status = "ok"
	case r.Processed == 0:
		r.Status = "failed"
	default:
		r.Status = "partial"
	}
}

func (r *ingestImportsReport) incrementSuppressionReason(reason string) {
	key := suppressionReasonBucket(reason)
	if key == "" {
		return
	}
	if r.SuppressionReasons == nil {
		r.SuppressionReasons = make(map[string]int)
	}
	r.SuppressionReasons[key]++
}

func suppressionReasonBucket(reason string) string {
	normalized := strings.TrimSpace(strings.ToLower(reason))
	if normalized == "" {
		return "import_policy"
	}
	return normalized
}

func sortedSuppressionReasonKeys(counts map[string]int) []string {
	return sortedPositiveCountKeys(counts)
}

func sortedPositiveCountKeys(counts map[string]int) []string {
	keys := make([]string, 0, len(counts))
	for key, count := range counts {
		if count <= 0 {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func formatIngestImportsReport(report ingestImportsReport) string {
	lines := []string{
		fmt.Sprintf("ingest imports %s", report.Status),
		fmt.Sprintf("status=%s", report.Status),
		fmt.Sprintf("source=%s", report.Source),
		fmt.Sprintf("input=%s", report.Input),
		fmt.Sprintf("failed_output=%s", fallbackString(report.FailedOutput)),
		fmt.Sprintf("failed_output_written=%d", report.FailedOutputWritten),
		fmt.Sprintf("failed_manifest=%s", fallbackString(report.FailedManifest)),
		fmt.Sprintf("failed_manifest_count=%d", report.FailedManifestCount),
		fmt.Sprintf("session_id=%s", report.Session.ID),
		fmt.Sprintf("resolved_by=%s", report.Scope.ResolvedBy),
		fmt.Sprintf("audit_only=%t", report.AuditOnly),
		fmt.Sprintf("continue_on_error=%t", report.ContinueOnError),
		fmt.Sprintf("attempted=%d", report.Attempted),
		fmt.Sprintf("processed=%d", report.Processed),
		fmt.Sprintf("failed=%d", report.Failed),
		fmt.Sprintf("materialized=%d", report.Materialized),
		fmt.Sprintf("suppressed=%d", report.Suppressed),
		fmt.Sprintf("note_deduplicated=%d", report.NoteDeduplicated),
		fmt.Sprintf("import_deduplicated=%d", report.ImportDeduplicated),
		fmt.Sprintf("warnings=%d", len(report.Warnings)),
	}
	for _, key := range sortedSuppressionReasonKeys(report.SuppressionReasons) {
		lines = append(lines, fmt.Sprintf("suppression_reason_%s=%d", key, report.SuppressionReasons[key]))
	}
	if report.AuditOnly {
		lines = append(lines,
			fmt.Sprintf("would_materialize=%d", report.WouldMaterialize),
			fmt.Sprintf("linked_existing_note=%d", report.LinkedExistingNote),
		)
	}
	return strings.Join(lines, "\n") + "\n"
}

func writeIngestFailureManifest(path string, report ingestImportsReport, failures []ingestFailureDetail) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if len(failures) == 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("prepare ingest-imports failed manifest directory: %w", err)
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("open ingest-imports failed manifest: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	body, err := marshalIndented(ingestFailureManifest{
		Status:              report.Status,
		Source:              report.Source,
		Input:               report.Input,
		FailedOutput:        report.FailedOutput,
		FailedOutputWritten: report.FailedOutputWritten,
		FailureCount:        len(failures),
		Failures:            failures,
	})
	if err != nil {
		return err
	}
	if _, err := io.WriteString(file, body); err != nil {
		return fmt.Errorf("write ingest-imports failed manifest: %w", err)
	}
	return nil
}

func fallbackString(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return stringNone
	}
	return value
}

func writeIngestImportsReport(stdout io.Writer, report ingestImportsReport, jsonOutput bool) error {
	if jsonOutput {
		body, err := marshalIndented(report)
		if err != nil {
			return err
		}
		_, err = io.WriteString(stdout, body)
		return err
	}

	_, err := io.WriteString(stdout, formatIngestImportsReport(report))
	return err
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
