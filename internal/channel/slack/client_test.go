package slackchannel

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	goslack "github.com/slack-go/slack"
)

func TestSlackMessageSenderPostsThreadMessage(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/chat.postMessage" {
			t.Errorf("请求路径 = %s, want /chat.postMessage", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("解析 Slack 请求失败: %v", err)
		}
		if got := r.Form.Get("channel"); got != "C123" {
			t.Errorf("channel = %q, want C123", got)
		}
		if got := r.Form.Get("thread_ts"); got != "1710000000.000100" {
			t.Errorf("thread_ts = %q", got)
		}
		if got := r.Form.Get("text"); got != "Agent 回答" {
			t.Errorf("text = %q, want Agent 回答", got)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"ok":true,"channel":"C123","ts":"1710000001.000200"}`)),
			Request:    r,
		}, nil
	})}

	client := goslack.New(
		"xoxb-test",
		goslack.OptionAPIURL("https://slack.test/"),
		goslack.OptionHTTPClient(httpClient),
	)
	sender := &slackMessageSender{client: client}

	timestamp, err := sender.SendMessage(context.Background(), "C123", "1710000000.000100", "Agent 回答")
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if timestamp != "1710000001.000200" {
		t.Fatalf("SendMessage() timestamp = %q, want 1710000001.000200", timestamp)
	}
}

func TestSlackMessageSenderUpdatesMessage(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/chat.update" {
			t.Errorf("请求路径 = %s, want /chat.update", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("解析 Slack 请求失败: %v", err)
		}
		if got := r.Form.Get("channel"); got != "C123" {
			t.Errorf("channel = %q, want C123", got)
		}
		if got := r.Form.Get("ts"); got != "1710000001.000200" {
			t.Errorf("ts = %q, want 1710000001.000200", got)
		}
		if got := r.Form.Get("text"); got != "🧠 执行规划" {
			t.Errorf("text = %q, want 🧠 执行规划", got)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"ok":true,"channel":"C123","ts":"1710000001.000200","text":"🧠 执行规划"}`)),
			Request:    r,
		}, nil
	})}

	client := goslack.New(
		"xoxb-test",
		goslack.OptionAPIURL("https://slack.test/"),
		goslack.OptionHTTPClient(httpClient),
	)
	sender := &slackMessageSender{client: client}

	if err := sender.UpdateMessage(context.Background(), "C123", "1710000001.000200", "🧠 执行规划"); err != nil {
		t.Fatalf("UpdateMessage() error = %v", err)
	}
}

type roundTripFunc func(request *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
