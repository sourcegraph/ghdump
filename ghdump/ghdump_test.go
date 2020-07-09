package ghdump

import (
	"reflect"
	"testing"
)

func TestParamsFromFilename(t *testing.T) {
	cases := []struct {
		filename  string
		expParams searchParams
	}{
		{
			filename:  "lang-c#__star-20__ppg-100__pg-10.json",
			expParams: searchParams{language: "c#", minStars: 20, perPage: 100, page: 10},
		},
		{
			filename:  "lang-javascript__star-200__ppg-100__pg-1.json",
			expParams: searchParams{language: "javascript", minStars: 200, perPage: 100, page: 1},
		},
		{
			filename:  "lang-objective-c__star-200__ppg-100__pg-1.json",
			expParams: searchParams{language: "objective-c", minStars: 200, perPage: 100, page: 1},
		},
	}
	for _, testCase := range cases {
		if got := paramsFromFilename(testCase.filename); !reflect.DeepEqual(got, &testCase.expParams) {
			t.Errorf("For filename %q, got %#v, expected %#v", testCase.filename, got, testCase.expParams)
		}
	}
}
