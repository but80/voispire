// +build prof

package main

import (
	"github.com/pkg/profile"
)

func init() {
	p := profile.Start(profile.ProfilePath("."))
	onExit = p.Stop
}
