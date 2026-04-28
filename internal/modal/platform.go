package modal

// SPDX-License-Identifier: BSD-2-Clause
//
// Copyright (c) Lewis Cook <hi@lcook.net>

import (
	"net/url"
	"slices"
	"strings"
)

type Platform struct {
	Name      string   `yaml:"name"`
	Default   bool     `yaml:"default"`
	Whitelist []string `yaml:"whitelist"`
	Proxy     string   `yaml:"proxy"`
	Fallback  string   `yaml:"fallback"`
}

func (p *Platform) Allowed(str string) bool {
	content, err := url.Parse(str)
	if err != nil {
		return false
	}

	return slices.Contains(
		p.Whitelist,
		strings.TrimPrefix(content.Hostname(), "www."),
	)
}

func (p *Platform) ProxyURL(str string) string {
	if p.Proxy == "" {
		return str
	}

	content, _ := url.Parse(str)
	content.Host = p.Proxy

	return content.String()
}
