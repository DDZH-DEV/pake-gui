package pake

import "testing"

func TestASCIIPackageName(t *testing.T) {
	cases := map[string]string{
		"HunliPai-AI":  "HunliPai-AI",
		"AI婚礼智能体":      "AIHunLiZhiNengTi",
		"婚礼派-AI":       "HunLiPai-AI",
		"  My App  ":   "My-App",
		"":             "App",
	}
	for in, want := range cases {
		if got := ASCIIPackageName(in); got != want {
			t.Errorf("ASCIIPackageName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizePackageIdentity(t *testing.T) {
	o := Options{Name: "AI婚礼智能体"}
	note := NormalizePackageIdentity(&o)
	if note == "" {
		t.Fatal("expected note")
	}
	if o.Name != "AIHunLiZhiNengTi" {
		t.Fatalf("name: got %q", o.Name)
	}
	if o.Title != "AI婚礼智能体" {
		t.Fatalf("title: got %q", o.Title)
	}

	o2 := Options{Name: "AI婚礼智能体", Title: "自定义标题"}
	_ = NormalizePackageIdentity(&o2)
	if o2.Title != "自定义标题" {
		t.Fatalf("should keep explicit title, got %q", o2.Title)
	}

	o3 := Options{Name: "HunliPai-AI"}
	if note := NormalizePackageIdentity(&o3); note != "" {
		t.Fatalf("unexpected note: %s", note)
	}
}
