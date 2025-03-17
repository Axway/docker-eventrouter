package report

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"axway.com/qlt-router/src/log"
	_ "github.com/jackc/pgx/v4/stdlib"
	_ "modernc.org/sqlite"
)

type UsageReporter struct {
	Conf        *UsageReportWriterConf
	Instance_id string
	Ctxs        string
	initialized bool
}

func (ur *UsageReporter) Report(forceSend bool) error {
	if ur.Conf.ReportPath == "" && ur.Conf.ServiceAccountClientID == "" {
		return nil
	}
	if ur.Conf.ReportPath != "" {
		if _, err := os.Stat(ur.Conf.ReportPath); os.IsNotExist(err) {
			os.MkdirAll(ur.Conf.ReportPath, 0755)
		}
	}

	dbConn, err := OpenBase(ur.Ctxs, ur.Conf, true)
	if err != nil {
		log.Errorc(ur.Ctxs, "Failed to open database to retrieve usage data", "error", err)
		return err
	}
	defer dbConn.Close()

	if !ur.initialized {
		transaction, err := dbConn.Begin()
		if err != nil {
			log.Errorc(ur.Ctxs, "SQL error (begin)", "err", err)
		} else {
			failed := false
			_, err = transaction.Exec("INSERT INTO " + lastSentTable + " (instanceId, mode, datetime)" +
				" VALUES ('" + ur.Instance_id + "','api','2020-01-01T00:00:00Z')" +
				"ON CONFLICT (instanceId, mode) DO NOTHING ;")
			if err != nil {
				log.Warnc(ur.Ctxs, "SQL error (create entry)", "err", err)
				failed = true
			}
			_, err = transaction.Exec("INSERT INTO " + lastSentTable + " (instanceId, mode, datetime)" +
				" VALUES ('" + ur.Instance_id + "','file','2020-01-01T00:00:00Z')" +
				"ON CONFLICT (instanceId, mode) DO NOTHING ;")
			if err != nil {
				log.Warnc(ur.Ctxs, "SQL error (create entry)", "err", err)
				failed = true
			}
			if failed {
				transaction.Rollback()
			} else {
				if err = transaction.Commit(); err != nil {
					log.Errorc(ur.Ctxs, "SQL error (fail to commit)", "err", err)
				}
			}
		}

		if ur.Conf.PlatformAPIUrl == "" {
			ur.Conf.PlatformAPIUrl = "https://platform.axway.com"
		}
		if ur.Conf.PlatformAuthenticationUrl == "" {
			ur.Conf.PlatformAuthenticationUrl = "https://login.axway.com"
		}

		ur.initialized = true
	}
	writeReport := true
	uploadReport := true
	if !forceSend && ur.initialized && ur.IsLastSentToday(dbConn, "api") {
		uploadReport = false
	}
	if !forceSend && ur.initialized && ur.IsLastSentToday(dbConn, "file") {
		writeReport = false
	}

	failedUpload := false
	failedWrite := false
	for _, product := range []string{"CFT"} {
		if uploadReport && ur.Conf.ServiceAccountClientID != "" {
			err := ur.UploadReport(ur.Ctxs+"-api", dbConn, product)
			if err != nil {
				log.Errorc(ur.Ctxs, "Failed to upload usage report to the platform.", "product", product, "error", err)
				failedUpload = true
			}
		}
		if writeReport && ur.Conf.ReportPath != "" {
			err := ur.WriteReport(ur.Ctxs+"-file", dbConn, product)
			if err != nil {
				log.Errorc(ur.Ctxs, "Failed to write usage report.", "product", product, "error", err)
				failedWrite = true
			}
		}
	}

	if ur.initialized {
		transaction, err := dbConn.Begin()
		if err != nil {
			log.Errorc(ur.Ctxs, "SQL error (begin)", "err", err)
		} else {
			failed := false
			if !failedUpload {
				_, err = transaction.Exec("UPDATE " + lastSentTable + " SET datetime = '" + time.Now().UTC().Format(time.RFC3339) + "'" +
					" WHERE instanceId='" + ur.Instance_id + "' AND mode='api';")
				if err != nil {
					log.Warnc(ur.Ctxs, "SQL error", "err", err)
					failed = true
				}
			}
			if !failedWrite {
				_, err = transaction.Exec("UPDATE " + lastSentTable + " SET datetime = '" + time.Now().UTC().Format(time.RFC3339) + "'" +
					" WHERE instanceId='" + ur.Instance_id + "' AND mode='file';")
				if err != nil {
					log.Warnc(ur.Ctxs, "SQL error", "err", err)
					failed = true
				}
			}
			if failed {
				transaction.Rollback()
			} else {
				if err = transaction.Commit(); err != nil {
					log.Errorc(ur.Ctxs, "Failed to update last sent event time", "err", err)
				}
			}
		}
	}

	return nil
}

func (ur *UsageReporter) IsLastSentToday(dbConn *sql.DB, mode string) bool {
	var err error
	var lastSentEvent string
	err = dbConn.QueryRow("SELECT datetime FROM " + lastSentTable + " WHERE instanceId='" + ur.Instance_id + "' AND mode='" + mode + "';").Scan(&lastSentEvent)
	if err != nil {
		log.Errorc(ur.Ctxs, "SQL error (LastSent)", "err", err)
	}
	lastSentTime, err := time.Parse(time.RFC3339, lastSentEvent)
	if err != nil {
		log.Errorc(ur.Ctxs, "Failed to parse last sent event time", "err", err)
		return false
	}
	if lastSentTime.Format("2006-01-02") == time.Now().UTC().Format("2006-01-02") {
		return true
	}
	return false
}

/**
 * UploadReport uploads a product's usage report to the platform.
 * It looks for the last sent time and uses the beginning of the previous month as the starting point.
 * Then it generates one report including all the events until now.
 * If there are no entries in the report, it will be sent anyway so we can validate the connection to the platform
 */
func (ur *UsageReporter) UploadReport(ctxS string, dbConn *sql.DB, monitorReq string) error {
	var err error

	/* General HTTP Client from both requests */
	var transport *http.Transport
	if ur.Conf.HttpProxyAddr != "" {
		transport = &http.Transport{
			Proxy: http.ProxyURL(&url.URL{
				Scheme: "http",
				User:   url.UserPassword(ur.Conf.HttpProxyUser, ur.Conf.HttpProxyPassword),
				Host:   ur.Conf.HttpProxyAddr,
			}),
			DisableKeepAlives: true,
		}
	} else {
		transport = &http.Transport{
			DisableKeepAlives: true,
		}
	}
	httpClient := &http.Client{Transport: transport}

	/* LOGIN REQUEST */
	/* Building Request Body */
	authReq := url.Values{}
	authReq.Set("grant_type", "client_credentials")
	authReq.Set("client_id", ur.Conf.ServiceAccountClientID)
	authReq.Set("client_secret", ur.Conf.ServiceAccountClientSecret)
	authReqBody := strings.NewReader(authReq.Encode())
	authUrl := ur.Conf.PlatformAuthenticationUrl + "/auth/realms/Broker/protocol/openid-connect/token"
	authRequest, err := http.NewRequest("POST", authUrl, authReqBody)
	if err != nil {
		log.Errorc(ctxS, "Failed create auth HTTP request", "error", err)
		return err
	}
	authRequest.Header.Add("Content-type", "application/x-www-form-urlencoded")

	/* Sending it */
	authResp, err := httpClient.Do(authRequest)
	if err != nil {
		log.Errorc(ctxS, "Request to "+authUrl+" failed", "error:", err)
		if authResp != nil {
			log.Errorc(ctxS, "Response", "code:", authResp.StatusCode)
		}
		return err
	}
	defer authResp.Body.Close()

	/* Getting token */
	body, _ := io.ReadAll(authResp.Body)
	if authResp.StatusCode != http.StatusOK {
		log.Errorc(ctxS, "Failed to request API token", "AuthAPI", authUrl, "httpCode", authResp.StatusCode, "body", string(body))
		return errors.New("error occurred while requesting a token! Returned code: " + strconv.Itoa(authResp.StatusCode))
	}
	var authRespMap map[string]string
	json.Unmarshal(body, &authRespMap)
	token := authRespMap["access_token"]
	if token == "" {
		log.Errorc(ctxS, "Failed to retrieve API token in response", "body", string(body))
		return errors.New("failed to get access token")
	}
	tokenType := authRespMap["token_type"]
	log.Debugc(ctxS, "API token successfully retrieved.")

	/* USAGE REQUEST */
	/* Content from Database */
	var lastSentEvent string
	err = dbConn.QueryRow("SELECT datetime FROM " + lastSentTable + " WHERE instanceId='" + ur.Instance_id + "' AND mode='api';").Scan(&lastSentEvent)
	if err != nil {
		log.Debugc(ctxS, "Can't find last datetime sent", "err", err)
		lastSentEvent = time.Now().UTC().AddDate(0, 0, -1).Format(time.RFC3339)
	}
	lastSentTime, err := time.Parse(time.RFC3339, lastSentEvent)
	if err != nil {
		log.Errorc(ctxS, "Failed to parse last sent event time", "err", err)
		return err
	}

	/* Common elements to all requests */
	apiUrl := ur.Conf.PlatformAPIUrl + "/api/v1/usage"
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="usage.json"`)
	h.Add("Content-Type", `application/json`)
	firstReport := true

	fromDate := lastSentTime.AddDate(0, -1, 0).Truncate(time.Hour)
	for {
		month := fromDate.Format("2006-01")
		fromDatetime := month + "-01T00:00:00Z"
		toDate := fromDate.AddDate(0, 1, 0)
		toDatetime := toDate.Format("2006-01") + "-01T00:00:00Z"

		if fromDate.After(time.Now().UTC()) {
			break //done
		}
		fromDate = toDate

		usageOutput, entries, err := ur.GenerateReport(dbConn, fromDatetime, toDatetime, "CFT")
		if err != nil {
			log.Errorc(ctxS, "Failed get usage data", "from", fromDatetime, "to", toDatetime, "error", err)
			break
		}
		if entries == 0 && !firstReport {
			continue
		}
		log.Debugc(ctxS, "Requested usage", "from", fromDatetime, "to", toDatetime, "n", entries)

		/* Building Request Body */
		b := new(bytes.Buffer)
		w := multipart.NewWriter(b)
		field, _ := w.CreatePart(h)
		field.Write([]byte(usageOutput))
		w.Close()

		/* Building Request */
		apiRequest, err := http.NewRequest("POST", apiUrl, b)
		if err != nil {
			log.Errorc(ctxS, "Failed create usage report upload HTTP request ", "error", err)
			break
		}
		apiRequest.Header.Set("Authorization", tokenType+" "+token)
		apiRequest.Header.Add("Content-type", w.FormDataContentType())

		/* Sending it */
		apiResp, err := httpClient.Do(apiRequest)
		if err != nil {
			log.Errorc(ctxS, "Failed to upload usage report.", "from", fromDatetime, "to", toDatetime, "ApiURL", apiUrl, "error:", err)
			break
		}

		bodyApi, _ := io.ReadAll(apiResp.Body)
		if apiResp.StatusCode != http.StatusAccepted {
			apiResp.Body.Close()
			log.Errorc(ctxS, "Usage report upload returned an error", "ApiURL", apiUrl, "httpCode", apiResp.StatusCode, "body", string(bodyApi))
			err = errors.New("HTTP Post to " + apiUrl + " returned code: " + strconv.Itoa(apiResp.StatusCode))
			break
		}
		apiResp.Body.Close()
		firstReport = false
	}
	return err
}

/**
* WriteReport writes a product's usage report to files (one file per month).
* It looks for the last sent time and picks the beginning of the previous month as the starting point (day 1 at hour 00:00:00).
* Then it generates the report for each month until now.
* If a month has no entries, the file is not written.
 */
func (ur *UsageReporter) WriteReport(ctxS string, dbConn *sql.DB, monitorReq string) error {
	var lastSentEvent string
	err := dbConn.QueryRow("SELECT datetime FROM " + lastSentTable + " WHERE instanceId='" + ur.Instance_id + "' AND mode='file';").Scan(&lastSentEvent)
	if err != nil {
		log.Debugc(ctxS, "Can't find last datetime sent", "err", err)
		lastSentEvent = time.Now().UTC().AddDate(0, 0, -1).Format(time.RFC3339)
	}
	lastSentTime, err := time.Parse(time.RFC3339, lastSentEvent)
	if err != nil {
		log.Errorc(ctxS, "Failed to parse last sent event time", "err", err)
		return err
	}

	fromDate := lastSentTime.AddDate(0, -1, 0).Truncate(time.Hour)
	for {
		month := fromDate.Format("2006-01")
		fromDatetime := month + "-01T00:00:00Z"
		toDate := fromDate.AddDate(0, 1, 0)
		toDatetime := toDate.Format("2006-01") + "-01T00:00:00Z"

		if fromDate.After(time.Now().UTC()) {
			break //done
		}
		fromDate = toDate

		content, entries, err := ur.GenerateReport(dbConn, fromDatetime, toDatetime, monitorReq)
		if err != nil {
			break
		}

		if entries == 0 {
			log.Tracec(ctxS, "No entry found for date range", "from", fromDatetime, "to", toDatetime)
			continue
		}

		log.Debugc(ctxS, "Writing usage report", "from", fromDatetime, "to", toDatetime, "n", entries)
		f, err := os.OpenFile(path.Join(ur.Conf.ReportPath, month+"_"+strings.ToLower(monitorReq)+".json"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
		if err != nil {
			break
		}
		defer f.Close()
		_, err = f.Write([]byte(content))
		if err != nil {
			break
		}
	}
	return err
}

/**
 * GenerateReport generates a json output that is compatible with the platform
 * Includes every event from fromDatetime (including) to toDatetime (excluding) for the requested monitor
 * Returns the json string, the number of entries and an error (in case there were any)
 */
func (ur *UsageReporter) GenerateReport(dbConn *sql.DB, fromDatetime string, toDatetime string, monitorReq string) (string, int, error) {
	var output string
	var err error
	var rows *sql.Rows
	entries := 0

	if monitorReq == "" {
		monitorReq = "CFT"
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)
	if toDatetime == "" {
		toDatetime = timestamp
	}

	output = `{`
	output += `"envId": "` + ur.Conf.EnvironmentId + `", `
	output += `"timestamp": "` + timestamp + `", `
	output += `"granularity": 3600000, ` // TODO: 1h, could be a parameter
	output += `"schemaId": "https://platform.axway.com/schemas/report.json", `
	output += `"report": {`

	if fromDatetime == "" {
		rows, err = dbConn.Query("SELECT * FROM " + usageTable + " WHERE  datetime < '" + toDatetime + "' ORDER BY datetime")
	} else {
		rows, err = dbConn.Query("SELECT * FROM " + usageTable + " WHERE datetime >= '" + fromDatetime + "' AND datetime < '" + toDatetime + "' ORDER BY datetime")
	}
	empty := true
	if err == nil {
		first := true
		for rows.Next() {
			var metrics string
			var tabversion int
			var monitor string
			var datetime string

			err = rows.Scan(&datetime, &monitor, &tabversion, &metrics)
			if err != nil {
				log.Errorc(ur.Ctxs, "Failed to scan row", "err", err)
				continue
			}
			if monitorReq != monitor {
				log.Tracec(ur.Ctxs, "Requested monitor different from entry monitor", "requested", monitorReq, "entry", monitor)
				continue
			}
			if !first {
				// Avoid trailing comma
				output += `,`
			}
			if monitor == "CFT" {
				var m CftMetrics
				m.FromJson(metrics)

				output += `"` + datetime + `": {`
				output += `"usage": `
				output += m.ToPlatformJson()
				output += `}`

				empty = false
			} else {
				log.Warnc(ur.Ctxs, "Monitor value not supported", "monitor", monitor)
				continue
			}
			first = false
			entries += 1
		}
	} else {
		log.Errorc(ur.Ctxs, "Failed to query usage", "err", err)
	}
	if empty {
		m := CftMetrics{TransferCount: 0, Mainframe: 0}
		output += `"` + toDatetime[0:14] + "00:00Z" + `": {`
		output += `"usage": `
		output += m.ToPlatformJson()
		output += `}`
	}
	output += `}, `
	output += `"meta": {`
	output += `"agentName": "` + ur.Instance_id + `"`
	output += `}`
	output += `}`

	return output, entries, err
}

/**
 * Main loop of the UsageReporter
 */
func (ur *UsageReporter) Reporting(ctx context.Context) {
	if ur.Conf.ReportPath == "" && ur.Conf.ServiceAccountClientID == "" {
		log.Warnc(ur.Ctxs, "No output configured for usage reporting: only storing in the database. "+
			"Set either reportPath or axwayClientID/axwayClientSecret to enable usage reporting.")
		return
	}

	var reportingTicker *time.Ticker
	if ur.Conf.ReportFrequency > 0 {
		timerDuration, _ := time.ParseDuration(strconv.Itoa(ur.Conf.ReportFrequency) + "m")
		reportingTicker = time.NewTicker(timerDuration)
	} else {
		timerDuration, _ := time.ParseDuration("15m")
		reportingTicker = time.NewTicker(timerDuration)
	}

	for {
		select {
		case <-reportingTicker.C:
			log.Debugc(ur.Ctxs, "Usage report waking up...")
			err := ur.Report(ur.Conf.ReportFrequency > 0)
			if err != nil {
				log.Errorc(ur.Ctxs, "Error occurred while generating usage report", "err", err)
			}
		case <-ctx.Done():
			log.Infoc(ur.Ctxs, "Usage Reporter Loop closed: done")
			return
		}
	}
}
