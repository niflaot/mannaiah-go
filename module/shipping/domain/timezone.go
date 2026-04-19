package domain

import "time"

var (
	// utcMinusFiveLocation defines the fixed UTC-05:00 timezone used by shipping documents.
	utcMinusFiveLocation = time.FixedZone("UTC-05:00", -5*60*60)
)

// ToUTCMinusFive converts one timestamp to the fixed UTC-05:00 timezone.
func ToUTCMinusFive(value time.Time) time.Time {
	return value.In(utcMinusFiveLocation)
}

// FormatUTCMinusFiveTimestamp formats one timestamp in "YYYY-MM-DD HH:MM:SS UTC-05:00" format.
func FormatUTCMinusFiveTimestamp(value time.Time) string {
	return ToUTCMinusFive(value).Format("2006-01-02 15:04:05 UTC-05:00")
}
