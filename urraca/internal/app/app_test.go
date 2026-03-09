package app

import (
	"context"
	"testing"
)

func TestRunValidation(t *testing.T) {
	cases := []struct{
		args []string
		wantErr bool
	}{
		{[]string{}, true},
		{[]string{"example.com"}, false},
		{[]string{"https://foo"}, false},
		{[]string{"not a url"}, true},
	}
	for _, c := range cases {
		err := Run(context.Background(), c.args)
		if (err != nil) != c.wantErr {
			t.Errorf("Run(%v) error = %v, wantErr=%v", c.args, err, c.wantErr)
		}
	}
}
