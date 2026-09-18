package server

import "time"

const cst = 8 * 3600

func timeNow() string {
	return time.Now().In(time.FixedZone("CST", cst)).Format("2006-01-02")
}

func addDays(date string, n int) string {
	d, _ := time.Parse("2006-01-02", date)
	return d.AddDate(0, 0, n).Format("2006-01-02")
}
