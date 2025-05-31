package main

import (
	cfg "github.com/conductorone/baton-expensify/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/config"
)

func main() {
	config.Generate("expensify", cfg.Config)
}
