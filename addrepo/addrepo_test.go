package addrepo

import (
	"reflect"
	"testing"
)

func TestToTraunches(t *testing.T) {
	testCases := []struct {
		repos        []string
		traunchSize  int
		expTraunches [][]string
	}{
		{
			repos:       []string{"a", "b", "c", "d", "e", "f"},
			traunchSize: 2,
			expTraunches: [][]string{
				{"a", "b"},
				{"c", "d"},
				{"e", "f"},
			},
		},
		{
			repos:       []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12"},
			traunchSize: 10,
			expTraunches: [][]string{
				{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
				{"11", "12"},
			},
		},
	}
	for _, testCase := range testCases {
		if traunches := toTraunches(testCase.repos, testCase.traunchSize); !reflect.DeepEqual(traunches, testCase.expTraunches) {
			t.Errorf("Expected traunches %v, but got %v", testCase.expTraunches, traunches)
		}
	}

}
