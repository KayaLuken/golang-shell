package main

import (
	"reflect"
	"testing"
)

func TestParseMetas(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"foo bar", []string{"foo", "bar"}},
		{"'foo bar'", []string{"foo bar"}},
		{"foo 'bar baz'", []string{"foo", "bar baz"}},
		{"foo \"bar baz\"", []string{"foo", "bar baz"}},
		{"foo\\ bar baz", []string{"foo bar", "baz"}},
		{"foo 'bar baz' qux", []string{"foo", "bar baz", "qux"}},
	}
	for _, tt := range tests {
		got := parseMetas(tt.input)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("parseMetas(%q) = %#v, want %#v", tt.input, got, tt.want)
		}
	}
}

func TestCommonPrefix(t *testing.T) {
	tests := []struct {
		a, b string
		want string
	}{
		{"foobar", "foobaz", "fooba"},
		{"foo", "foo", "foo"},
		{"foo", "bar", ""},
		{"", "bar", ""},
		{"foo", "", ""},
		{"abc", "abcd", "abc"},
	}
	for _, tt := range tests {
		got := commonPrefix(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("commonPrefix(%q, %q) = %q, want %q", tt.a, tt.b, got, tt.want)
		}
	}
}
