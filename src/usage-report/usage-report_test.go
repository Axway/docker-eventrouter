package report

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"axway.com/qlt-router/src/connectors/file"
	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
	"github.com/a8m/envsubst"
	_ "github.com/jackc/pgx/v4/stdlib"
	_ "modernc.org/sqlite"
)

/*
Should include in file: SUMMARY and not SUMMARY events
- Simple transfers;
- Group of Files;
- Broadcasts;
- Broadcasts with group of files;
All this mixing 4 different OSs (linux, windows, zos, os400)

Leave the Date to be changed in the input file, set current month, + 2 previous months
Output verification should compare content with template content + verify filenames
*/
var (
	templateFile    = "testdata/events.2025-03-20T10:24:14.69000594Z.xml.template"
	outputFile      = "testdata/events[tag].2025-03-20T10:24:14.69000594Z.xml"
	ctx             = "TEST-USAGE-REPORT"
	current_month   = ""
	previous_month  = ""
	previous_month2 = ""
)

func TestUsageReportPG(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
		return
	}
	t.Parallel()
	tag := "pg"
	Cleanup(tag, nil)
	eventsCount := PrepareTemplate(t, tag)
	reader := &file.FileStoreRawReaderConfig{
		FilenamePrefix: strings.Split(strings.ReplaceAll(outputFile, "[tag]", tag), ".")[0],
		FilenameSuffix: ".xml",
		ReaderFilename: "./testdata/reader.pg.cursor",
	}
	url, _ := envsubst.String("${PGMETRICS:-localhost}:25432/usagereporttest")
	writer := &UsageReportWriterConf{
		DatabaseUri:     url,
		DatabaseType:    "postgres",
		DatabaseUser:    "usagereport",
		DatabaseSecret:  "usagereport",
		EnvironmentId:   "xxxxxx",
		ReportPath:      "./testdata/pg",
		ReportFrequency: 1,
	}
	RunConnectors(t, reader, writer, eventsCount)
	VerifyFiles(t, tag)
	Cleanup(tag, nil)
}

func TestUsageReportSQLite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
		return
	}
	t.Parallel()
	tag := "sqlite"
	database := "./testdata/sqlite-usage-database.dbx"
	Cleanup(tag, []string{database})
	eventsCount := PrepareTemplate(t, tag)
	reader := &file.FileStoreRawReaderConfig{
		FilenamePrefix: strings.Split(strings.ReplaceAll(outputFile, "[tag]", tag), ".")[0],
		FilenameSuffix: ".xml",
		ReaderFilename: "./testdata/reader.sqlite.cursor",
	}
	writer := &UsageReportWriterConf{
		DatabaseUri:     database,
		EnvironmentId:   "xxxxxx",
		ReportPath:      "./testdata/sqlite",
		ReportFrequency: 1,
	}
	RunConnectors(t, reader, writer, eventsCount)
	VerifyFiles(t, tag)
	Cleanup(tag, []string{database})
}

func VerifyFiles(t *testing.T, tag string) {
	t.Helper()
	/* Compare content with template content + verify filenames*/
	/* File 1:
	   {"envId": "xxxxxx", "timestamp": "2025-03-31T13:02:23Z", "granularity": 3600000, "schemaId": "https://platform.axway.com/schemas/report.json"
	   , "report": {"2025-01-10T13:00:00Z": {"usage": {"CFT.Transfers": 6,"CFT.MainframeTransfers": 3}}}, "meta": {"agentName": ""}}*/
	/* File 2:
	   {"envId": "xxxxxx", "timestamp": "2025-03-31T13:02:23Z", "granularity": 3600000, "schemaId": "https://platform.axway.com/schemas/report.json", "report": {"2025-02-10T13:00:00Z": {"usage": {"CFT.Transfers": 4,"CFT.MainframeTransfers": 0}}}, "meta": {"agentName": ""}}*/
	/* File 3:
	   {"envId": "xxxxxx", "timestamp": "2025-03-31T13:02:23Z", "granularity": 3600000, "schemaId": "https://platform.axway.com/schemas/report.json", "report": {"2025-03-10T10:00:00Z": {"usage": {"CFT.Transfers": 3,"CFT.MainframeTransfers": 0}}}, "meta": {"agentName": ""}}*/
	log.Infoc(ctx, "VerifyFiles", "tag", tag,
		"[MONTH]", current_month,
		"[MONTH-1]", previous_month,
		"[MONTH-2]", previous_month2)
	template, _ := os.ReadFile("./testdata/output/" + "current" + "_cft.json")
	content, err := os.ReadFile("./testdata/" + tag + "/" + current_month[0:7] + "_cft.json")
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	templateS := string(template)
	templateS = strings.ReplaceAll(templateS, "[MONTH]", current_month)
	for _, s := range strings.Split(templateS, "???") {
		if !strings.Contains(string(content), s) {
			t.Fatalf("failed to read output file: %v", err)
		}
	}
	log.Infoc(ctx, "File OK", "file", current_month[0:7]+"_cft.json")

	template, _ = os.ReadFile("./testdata/output/" + "previous" + "_cft.json")
	content, err = os.ReadFile("./testdata/" + tag + "/" + previous_month[0:7] + "_cft.json")
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	templateS = string(template)
	templateS = strings.ReplaceAll(templateS, "[MONTH-1]", previous_month)
	for _, s := range strings.Split(templateS, "???") {
		if !strings.Contains(string(content), s) {
			t.Fatalf("failed to read output file: %v", err)
		}
	}
	log.Infoc(ctx, "File OK", "file", previous_month[0:7]+"_cft.json")

	template, _ = os.ReadFile("./testdata/output/" + "previous2" + "_cft.json")
	content, err = os.ReadFile("./testdata/" + tag + "/" + previous_month2[0:7] + "_cft.json")
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	templateS = string(template)
	templateS = strings.ReplaceAll(templateS, "[MONTH-2]", previous_month2)
	for _, s := range strings.Split(templateS, "???") {
		if !strings.Contains(string(content), s) {
			t.Fatalf("failed to read output file: %v", err)
		}
	}
	log.Infoc(ctx, "File OK", "file", previous_month2[0:7]+"_cft.json")
}
func PrepareTemplate(t *testing.T, tag string) int {
	t.Helper()
	/* Set dates in template input files */
	/* 4 different values should be replaced by current year/month ([MONTH] and [MONTH_INV]), current month -1 ([MONTH-1]), current month -2 ([MONTH-2]) */
	current_month = time.Now().UTC().Format("2006-01-02")[0:8] + "10"
	current_time, _ := time.Parse("2006-01-02", current_month)
	current_month_inv := current_time.Format("02.01.2006")
	previous_month = current_time.AddDate(0, -1, 0).Format("2006-01-02")
	previous_month2 = current_time.AddDate(0, -2, 0).Format("2006-01-02")

	// Open the template file
	content, err := os.ReadFile(templateFile)
	if err != nil {
		t.Fatalf("failed to read template file: %v", err)
	}

	// Replace macros with actual values
	log.Infoc(ctx, "Replacing macros in template file",
		"[MONTH]", current_month,
		"[MONTH_INV]", current_month_inv,
		"[MONTH-1]", previous_month,
		"[MONTH-2]", previous_month2)
	replacedContent := string(content)
	replacedContent = strings.ReplaceAll(replacedContent, "[MONTH]", current_month)
	replacedContent = strings.ReplaceAll(replacedContent, "[MONTH_INV]", current_month_inv)
	replacedContent = strings.ReplaceAll(replacedContent, "[MONTH-1]", previous_month)
	replacedContent = strings.ReplaceAll(replacedContent, "[MONTH-2]", previous_month2)

	// Write the replaced content to the output file
	err = os.WriteFile(strings.ReplaceAll(outputFile, "[tag]", tag), []byte(replacedContent), 0644)
	if err != nil {
		t.Fatalf("failed to write output file: %v", err)
	}
	return strings.Count(replacedContent, "\n")
}
func Cleanup(tag string, files []string) {
	log.Infoc(ctx, "Cleaning up...", "tag", tag)
	os.Remove(strings.ReplaceAll(outputFile, "[tag]", tag))
	os.RemoveAll("./testdata/" + tag)
	os.Remove("./testdata/reader." + tag + ".cursor")
	for _, f := range files {
		os.Remove(f)
	}
}

func RunConnectors(t *testing.T, reader, writer processor.Connector, eventsCount int) {
	t.Helper()

	ctl := make(chan processor.ControlEvent, 100)
	channels := processor.NewChannels()

	connectorReader := processor.NewProcessor("tested-reader", reader, channels)
	cS := channels.Create("testedStream", -1)
	connectorWriter := processor.NewProcessor("tested-writer", writer, channels)

	processors := []*processor.Processor{connectorReader, connectorWriter}
	c := []*processor.Channel{nil, cS, nil}
	rProcessors := []processor.ConnectorRuntime{}

	log.Infoc(ctx, "Starting server connectors...")
	errorCount := 0
	var wgs sync.WaitGroup
	for i, p2 := range processors {
		wgs.Add(1)
		go func(i int, p2 *processor.Processor) {
			defer wgs.Done()
			p, err := p2.Start(context.Background(), ctl, c[i], c[i+1])
			if err != nil {
				t.Error("Error starting connector'"+p.Ctx()+"'", err)
				errorCount++
			}
			rProcessors = append(rProcessors, p)
		}(i, p2)
	}
	wgs.Wait()
	if errorCount > 0 {
		return
	}
	log.Infoc(ctx, "All connectors started")

	cond := false
	for !cond {
		op := <-ctl
		op.Log()
		if op.From.Name == connectorReader.Name && op.Id == "ACK_ALL_DONE" && connectorReader.Out_ack >= int64(eventsCount) {
			cond = true
			t.Logf("op %+v", op.From)
			log.Infoc(ctx, "All events read")
		}
		if op.Id == "ERROR" {
			t.Error("Error", op.Id, op.From.Name, op.Msg)
			return
		}
	}
	log.Infoc(ctx, "Sleeping for 1.5 minutes, waiting for report generation")
	time.Sleep(2 * 90 * time.Second)
	log.Infoc(ctx, "Stopping connectors")
	for _, p := range rProcessors {
		p.Close()
	}
}
