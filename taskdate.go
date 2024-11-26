package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	dateLayout = "20060102"
)

func NextDate(now time.Time, date string, repeat string) (string, error) {

	var (
		modifier    []string
		periodicity string
	)

	startDate, err := time.Parse(dateLayout, date)
	if err != nil {
		return "", err
	}

	repeatSl := strings.Split(repeat, " ")
	periodicity = repeatSl[0]
	if len(repeatSl) > 1 {
		modifier = repeatSl[1:]
	}

	switch periodicity {
	case "d":
		return calculateDays(now, startDate, modifier)
	case "y":
		return calculateYears(now, startDate)
	case "w":
		return "", errors.New("неподдерживаемый формат")
	case "m":
		return "", errors.New("неподдерживаемый формат")
	default:
		return "", errors.New("недопустимый символ")
	}

}

func calculateDays(now time.Time, startDate time.Time, modifier []string) (string, error) {

	if len(modifier) == 0 {
		return "", errors.New("не указан интервал в днях")
	}

	days, err := strconv.Atoi(modifier[0])
	if err != nil || days < 1 {
		return "", fmt.Errorf("недопустимое значение %s", modifier)
	}
	if days > 400 {
		return "", errors.New("превышен максимально допустимый интервал")
	}

	if now.Before(startDate) {
		now = startDate
	}

	nextDate := startDate
	for nextDate.Compare(now) < 1 {
		nextDate = nextDate.AddDate(0, 0, days)
	}
	return nextDate.Format(dateLayout), nil

}

func calculateYears(now time.Time, startDate time.Time) (string, error) {

	var year int
	if startDate.After(now) {
		year = startDate.Year() + 1
	} else {
		year = now.Year()
		if (now.Month() > startDate.Month()) || (now.Month() == startDate.Month() && now.Day() >= startDate.Day()) {
			year = now.Year() + 1
		}
	}

	nextDate := time.Date(year, startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	return nextDate.Format(dateLayout), nil
}
