package main

import (
	"errors"
	"testing"

	"github.com/vcaesar/tt"
)

func TestProcessSubFields(t *testing.T) {
	var testData = []struct {
		subFileName   string
		record        string
		outputName    string
		expectedValue string
		expectedErr   error
	}{
		{"JiraLogTime", ";05/Jan/21 8:45 AM;uid:1291231203;14400", "Hours", "14400", nil},
		{"JiraLogTime", ";05/Jan/21 8:45 AM;uid:1291231203;14400", "Reporter", "uid:1291231203", nil},
		{"JiraLogTime", ";05/Jan/21 8:45 AM;uid:1291231203;14400", "Date", "05/Jan/21 8:45 AM", nil},
		{"JiraLogTime", "dasdfa;sfasdf ;05/Jan/21 8:45 AM;uid:1291231203;14400", "Date", "05/Jan/21 8:45 AM", nil},
		{"JiraLogTime", "uid:1291231203;14400", "Date", "05/Jan/21", errors.New("invalid record format, must have 4 sections")},
		{"JiraLogTime", ";05/Jan/21 8:45 AM;uid:1291231203;14400", "Content", "dfdsf", errors.New("unsupported output name")},
		{"value", ";1214400", "Test", ";1214400", nil},
	}

	for _, data := range testData {
		result, err := processSubFields(data.subFileName, data.record, data.outputName)
		tt.Equal(t, err, data.expectedErr)
		if err == nil {
			tt.Equal(t, result, data.expectedValue)
		}
	}
}
