package api

import (
	"fmt"
	"net/http"
	"time"
)

var nowFunc = time.Now

func parseTimeWindow(
	r *http.Request,
	startPrimary string,
	startFallback string,
	endPrimary string,
	endFallback string,
) (time.Time, time.Time, error) {
	windowValue := r.URL.Query().Get("window")
	startValue := firstQueryValue(r, startPrimary, startFallback)
	endValue := firstQueryValue(r, endPrimary, endFallback)

	if windowValue != "" {
		if startValue != "" {
			return time.Time{}, time.Time{}, fmt.Errorf("%s cannot be used with window", startPrimary)
		}

		duration, err := time.ParseDuration(windowValue)
		if err != nil || duration <= 0 {
			return time.Time{}, time.Time{}, fmt.Errorf("window must be a positive duration such as 5m or 1h")
		}

		end := nowFunc().UTC()
		if endValue != "" {
			end, err = time.Parse(time.RFC3339, endValue)
			if err != nil {
				return time.Time{}, time.Time{}, fmt.Errorf("%s must be RFC3339", endPrimary)
			}
		}

		return end.Add(-duration), end, nil
	}

	start, err := parseRequiredTime(r, startPrimary, startFallback)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	end, err := parseRequiredTime(r, endPrimary, endFallback)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	return start, end, nil
}

func parseRequiredTime(r *http.Request, primary string, fallback string) (time.Time, error) {
	value := firstQueryValue(r, primary, fallback)
	if value == "" {
		return time.Time{}, fmt.Errorf("%s is required", primary)
	}

	timestamp, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be RFC3339", primary)
	}

	return timestamp, nil
}

func firstQueryValue(r *http.Request, primary string, fallback string) string {
	value := r.URL.Query().Get(primary)
	if value == "" && fallback != "" {
		value = r.URL.Query().Get(fallback)
	}

	return value
}
