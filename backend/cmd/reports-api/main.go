package main

import (
	"bionicpro/reports-api/cmd/reports-api/auth"
	"bionicpro/reports-api/cmd/reports-api/config"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type report struct {
	UserID            string  `json:"user_id"`
	Username          string  `json:"username"`
	FullName          string  `json:"full_name"`
	Email             string  `json:"email"`
	ProsthesisID      string  `json:"prosthesis_id"`
	Model             string  `json:"model"`
	SerialNumber      string  `json:"serial_number"`
	IssuedAt          string  `json:"issued_at"`
	PeriodStart       string  `json:"period_start"`
	PeriodEnd         string  `json:"period_end"`
	TotalEvents       uint64  `json:"total_events"`
	AvgResponseTimeMS float64 `json:"avg_response_time_ms"`
	MaxResponseTimeMS uint32  `json:"max_response_time_ms"`
	MinBatteryLevel   uint8   `json:"min_battery_level"`
	AvgSignalQuality  float64 `json:"avg_signal_quality"`
	SlowEvents        uint64  `json:"slow_events"`
	GeneratedAt       string  `json:"generated_at"`
}

type reportPeriod struct {
	Start time.Time
	End   time.Time
}

func main() {
	cfg := config.Create()

	verifier := auth.NewVerifier(cfg.Issuer, cfg.JwksURL)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/reports", withCORS(func(w http.ResponseWriter, r *http.Request) {
		handleReports(w, r, cfg, verifier)
	}))

	log.Printf("reports-api listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, mux); err != nil {
		log.Fatal(err)
	}
}

func handleReports(w http.ResponseWriter, r *http.Request, cfg *config.Config, verifier *auth.AuthVerifier) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}

	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == "" || token == r.Header.Get("Authorization") {
		writeError(w, http.StatusUnauthorized, "Необходимо войти в систему")
		return
	}

	userClaims, err := verifier.Verify(r.Context(), token)
	if err != nil {
		log.Printf("token verification failed: %v", err)
		writeError(w, http.StatusUnauthorized, "Сессия недействительна или истекла")
		return
	}

	requestedUser := r.URL.Query().Get("user_id")
	if requestedUser == "" {
		requestedUser = userClaims.PreferredUsername
	}
	if requestedUser != userClaims.PreferredUsername {
		writeError(w, http.StatusForbidden, "Можно запросить только собственный отчёт")
		return
	}

	period, periodRequested, err := parseReportPeriod(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	reports, err := loadReports(r.Context(), cfg.ClickhouseURL, cfg.ClickhouseUser, cfg.ClickhousePass, requestedUser, period)
	if err != nil {
		log.Printf("report query failed: %v", err)
		writeError(w, http.StatusBadGateway, "Хранилище отчётов временно недоступно")
		return
	}
	if periodRequested && len(reports) == 0 {
		writeError(w, http.StatusNotFound, "За выбранный период отчётов нет")
		return
	}

	response := map[string]any{
		"user_id":      requestedUser,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"reports":      reports,
	}
	if periodRequested {
		response["period_start"] = period.Start.Format("2006-01-02")
		response["period_end"] = period.End.Format("2006-01-02")
	}
	writeJSON(w, http.StatusOK, response)
}

func loadReports(ctx context.Context, clickhouseURL, clickhouseUser, clickhousePass, username string, period *reportPeriod) ([]report, error) {
	periodFilter := ""
	if period != nil {
		periodFilter = fmt.Sprintf(
			"AND user_reports.period_start >= toDateTime(%s) AND user_reports.period_end <= toDateTime(%s)",
			clickhouseString(period.Start.Format("2006-01-02 15:04:05")),
			clickhouseString(period.End.Format("2006-01-02 15:04:05")),
		)
	}

	query := fmt.Sprintf(`
		SELECT
			user_id,
			keycloak_username AS username,
			full_name,
			email,
			prosthesis_id,
			model,
			serial_number,
			toString(issued_at) AS issued_at,
			toString(period_start) AS period_start,
			toString(period_end) AS period_end,
			total_events,
			avg_response_time_ms,
			max_response_time_ms,
			min_battery_level,
			avg_signal_quality,
			slow_events,
			toString(generated_at) AS generated_at
		FROM user_reports
		WHERE keycloak_username = %s
		%s
		ORDER BY prosthesis_id
		FORMAT JSONEachRow SETTINGS output_format_json_quote_64bit_integers = 0
	`, clickhouseString(username), periodFilter)

	values := url.Values{}
	values.Set("query", query)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, clickhouseURL+"/?"+values.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(clickhouseUser, clickhousePass)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("clickhouse status %d: %s", resp.StatusCode, string(body))
	}

	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []report{}, nil
	}

	reports := make([]report, 0, len(lines))
	for _, line := range lines {
		var item report
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return nil, err
		}
		reports = append(reports, item)
	}
	return reports, nil
}

func parseReportPeriod(r *http.Request) (*reportPeriod, bool, error) {
	startValue := strings.TrimSpace(r.URL.Query().Get("period_start"))
	endValue := strings.TrimSpace(r.URL.Query().Get("period_end"))
	if startValue == "" && endValue == "" {
		return nil, false, nil
	}
	if startValue == "" || endValue == "" {
		return nil, false, fmt.Errorf("Нужно указать начало и конец периода")
	}

	start, err := time.Parse("2006-01-02", startValue)
	if err != nil {
		return nil, false, fmt.Errorf("Дата начала периода должна быть в формате YYYY-MM-DD")
	}
	end, err := time.Parse("2006-01-02", endValue)
	if err != nil {
		return nil, false, fmt.Errorf("Дата окончания периода должна быть в формате YYYY-MM-DD")
	}
	if end.Before(start) {
		return nil, false, fmt.Errorf("Дата окончания периода не может быть раньше даты начала")
	}

	return &reportPeriod{
		Start: start,
		End:   end.Add(24*time.Hour - time.Second),
	}, true, nil
}

func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("write json failed: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func clickhouseString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "\\'") + "'"
}
